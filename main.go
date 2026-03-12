package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"randomVoiceAttack/config"
	"randomVoiceAttack/controller"
	"randomVoiceAttack/detector"
	"randomVoiceAttack/logger"
	"randomVoiceAttack/services"
	"randomVoiceAttack/utils"
)

type App struct {
	cfg        config.Config
	ctx        context.Context
	cancel     context.CancelFunc
	audioCtrl  *controller.AudioController
	httpServer *services.HTTPServer
}

func NewApp(cfg config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		cfg:    cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (app *App) Initialize() error {
	detector.Debug = app.cfg.Debug
	detector.SetDetectorConfig(detector.DetectorConfig{
		VolumeThreshold:       app.cfg.VolumeThreshold,
		LowFreqRatioThreshold: app.cfg.LowFreqRatioThreshold,
		TotalEnergyThreshold:  app.cfg.TotalEnergyThreshold,
	})

	if err := logger.Init(); err != nil {
		return fmt.Errorf("error initializing logger: %v", err)
	}

	audioFiles, err := utils.GetAudioFiles(app.cfg.AudioDirectory)
	if err != nil {
		return fmt.Errorf("error getting audio files: %v", err)
	}

	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files found in %s directory", app.cfg.AudioDirectory)
	}

	app.audioCtrl = controller.NewAudioController(audioFiles, app.cfg.PlayCount)
	return nil
}

func (app *App) StartServices() {
	heartbeatService := services.NewHeartbeatService(services.HeartbeatConfig{
		BluetoothHeartbeatInterval: app.cfg.BluetoothHeartbeatInterval,
		AntiLockInterval:           app.cfg.AntiLockInterval,
	})
	heartbeatService.Start(app.ctx)

	app.httpServer = services.NewHTTPServer(services.HTTPConfig{
		HTTPPort: app.cfg.HTTPPort,
	}, app.audioCtrl.AudioFiles, app.audioCtrl)
	app.httpServer.Start(app.ctx)

	go app.collectNoiseData()
}

func (app *App) collectNoiseData() {
	for {
		select {
		case <-app.ctx.Done():
			logger.Info("Noise data collection stopped")
			return
		case data := <-detector.NoiseDataChan:
			app.httpServer.AddNoiseData(data)
		case log := <-detector.DetectionLogChan:
			app.httpServer.AddDetectionLog(log.Message, log.Type)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (app *App) RunDetectionLoop() {
	quit := make(chan os.Signal, 2)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("Found %d audio files in %s directory", len(app.audioCtrl.AudioFiles), app.cfg.AudioDirectory)
	logger.Info("Starting random voice attack...")
	logger.Info("Press Ctrl+C to stop (press again to force exit)")
	logger.Info("Listening for low frequency noise...")
	logger.Info("Bluetooth heartbeat started")

	go func() {
		for {
			select {
			case <-app.ctx.Done():
				return
			default:
				app.audioCtrl.DetectAndPlay(app.ctx)
			}
		}
	}()

	for {
		select {
		case <-quit:
			select {
			case <-app.ctx.Done():
				logger.Info("Force exit!")
				os.Exit(1)
			default:
				logger.Info("Stopping... (press Ctrl+C again to force exit)")
				app.cancel()
			}
		case <-app.ctx.Done():
			time.Sleep(100 * time.Millisecond)
			logger.Info("Program stopped gracefully")
			return
		}
	}
}

func (app *App) Cleanup() {
	if err := detector.CloseAudioDevice(); err != nil {
		logger.Info("Error closing audio device: %v", err)
	}
	logger.Close()
}

func main() {
	var debugFlag bool
	var configPath string
	var helpFlag bool

	flag.BoolVar(&debugFlag, "debug", false, "Enable debug mode")
	flag.BoolVar(&debugFlag, "d", false, "Enable debug mode (shorthand)")
	flag.StringVar(&configPath, "config", "config.json", "Path to config file")
	flag.StringVar(&configPath, "c", "config.json", "Path to config file (shorthand)")
	flag.BoolVar(&helpFlag, "help", false, "Show help message")
	flag.BoolVar(&helpFlag, "h", false, "Show help message (shorthand)")

	flag.Parse()

	if helpFlag {
		fmt.Println("Random Voice Attack - A low-frequency sound triggered audio player")
		fmt.Println("\nUsage:")
		fmt.Println("  randomVoiceAttack [flags]")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		return
	}

	cfg, err := config.LoadConfigFromFile(configPath)
	if err != nil {
		fmt.Printf("Error loading config from %s: %v\n", configPath, err)
		return
	}

	if debugFlag {
		cfg.Debug = true
		fmt.Println("Debug mode enabled via CLI flag")
	}

	app := NewApp(cfg)
	defer app.Cleanup()

	if err := app.Initialize(); err != nil {
		logger.Info("%v", err)
		return
	}

	app.StartServices()
	app.RunDetectionLoop()
}
