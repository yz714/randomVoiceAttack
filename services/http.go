package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"randomVoiceAttack/controller"
	"randomVoiceAttack/logger"
	"randomVoiceAttack/utils"
)

const (
	maxDetectionLogs = 100
)

type NoiseData struct {
	Timestamp    string    `json:"timestamp"`
	LowFreqRatio float64   `json:"low_freq_ratio"`
	Volume       float64   `json:"volume"`
	MaxSample    float64   `json:"max_sample"`
	Time         time.Time `json:"time"`
}

type DetectionLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Type      string `json:"type"`
}

type HTTPConfig struct {
	HTTPPort            int
	MaxNoiseDataEntries int
}

type HTTPServer struct {
	config          HTTPConfig
	noiseData       []NoiseData
	recentNoiseData []NoiseData
	detectionLogs   []DetectionLog
	audioFiles      []string
	dataMutex       sync.Mutex
	playbackCtrl    controller.PlaybackController
	dataFilePath    string
	mux             *http.ServeMux
	mainCtx         context.Context
	mainCancel      context.CancelFunc
	needsSave       bool
	saveTicker      *time.Ticker
}

func NewHTTPServer(config HTTPConfig, audioFiles []string, playbackCtrl controller.PlaybackController) *HTTPServer {
	ctx, cancel := context.WithCancel(context.Background())
	server := &HTTPServer{
		config:       config,
		audioFiles:   audioFiles,
		playbackCtrl: playbackCtrl,
		dataFilePath: "./noise_data.json",
		mux:          http.NewServeMux(),
		mainCtx:      ctx,
		mainCancel:   cancel,
	}
	server.LoadNoiseDataFromFile()
	return server
}

func (s *HTTPServer) Start(ctx context.Context) {
	s.setupRoutes()

	s.saveTicker = time.NewTicker(5 * time.Second)

	utils.GoWithName("saveData", func() {
		for {
			select {
			case <-s.saveTicker.C:
				s.dataMutex.Lock()
				needsSave := s.needsSave
				s.needsSave = false
				s.dataMutex.Unlock()

				if needsSave {
					s.SaveNoiseDataToFile()
				}
			case <-ctx.Done():
				s.saveTicker.Stop()
				s.SaveNoiseDataToFile()
				return
			}
		}
	})

	utils.GoWithName("httpServer", func() {
		addr := fmt.Sprintf(":%d", s.config.HTTPPort)
		server := &http.Server{
			Addr:    addr,
			Handler: s.mux,
		}

		utils.GoWithName("listenAndServe", func() {
			logger.Info("HTTP server started on %s", addr)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Info("Error starting HTTP server: %v", err)
			}
		})

		<-ctx.Done()

		logger.Info("Shutting down HTTP server...")
		if err := server.Shutdown(context.Background()); err != nil {
			logger.Info("Error shutting down HTTP server: %v", err)
		}
		logger.Info("HTTP server stopped")
	})
}

func (s *HTTPServer) setupRoutes() {
	s.mux.Handle("/", http.FileServer(http.Dir("./frontend")))
	s.mux.HandleFunc("/api/noise-data", s.handleNoiseData)
	s.mux.HandleFunc("/api/noise-data/realtime", s.handleRealtimeNoiseData)
	s.mux.HandleFunc("/api/detection-logs", s.handleDetectionLogs)
	s.mux.HandleFunc("/api/audio/play/random", s.handlePlayRandom)
	s.mux.HandleFunc("/api/audio/play/sequence", s.handlePlaySequence)
	s.mux.HandleFunc("/api/audio/stop", s.handleStopAudio)
}

func (s *HTTPServer) handleNoiseData(w http.ResponseWriter, r *http.Request) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.noiseData)
}

func (s *HTTPServer) handleRealtimeNoiseData(w http.ResponseWriter, r *http.Request) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	var avgLowFreqRatio, avgVolume, avgMaxSample float64
	if len(s.recentNoiseData) > 0 {
		for _, d := range s.recentNoiseData {
			avgLowFreqRatio += d.LowFreqRatio
			avgVolume += d.Volume
			avgMaxSample += d.MaxSample
		}
		avgLowFreqRatio /= float64(len(s.recentNoiseData))
		avgVolume /= float64(len(s.recentNoiseData))
		avgMaxSample /= float64(len(s.recentNoiseData))
	}

	realtimeData := map[string]interface{}{
		"low_freq_ratio": avgLowFreqRatio,
		"volume":         avgVolume,
		"max_sample":     avgMaxSample,
		"timestamp":      time.Now().Format("2006-01-02 15:04:05"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(realtimeData)
}

func (s *HTTPServer) handleDetectionLogs(w http.ResponseWriter, r *http.Request) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.detectionLogs)
}

func (s *HTTPServer) checkIsPlaying() bool {
	return s.playbackCtrl.IsPlaying()
}

func (s *HTTPServer) setIsPlaying(playing bool) {
	s.playbackCtrl.SetPlaying(playing)
}

func (s *HTTPServer) getAudioFiles() []string {
	s.dataMutex.Lock()
	audioFiles := s.audioFiles
	s.dataMutex.Unlock()
	return audioFiles
}

func (s *HTTPServer) respondJSON(w http.ResponseWriter, success bool, message string, extra map[string]interface{}) {
	response := map[string]interface{}{
		"success": success,
		"message": message,
	}
	for k, v := range extra {
		response[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *HTTPServer) handlePlayRandom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.checkIsPlaying() {
		s.respondJSON(w, false, "Audio is already playing", map[string]interface{}{"error": "Audio is already playing"})
		return
	}

	go func() {
		s.playbackCtrl.PlayRandomAudios(s.mainCtx)
	}()

	s.respondJSON(w, true, "Playing random audio", nil)
}

func (s *HTTPServer) handlePlaySequence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.checkIsPlaying() {
		s.respondJSON(w, false, "Audio is already playing", map[string]interface{}{"error": "Audio is already playing"})
		return
	}

	go func() {
		s.playbackCtrl.PlayRandomAudios(s.mainCtx)
	}()

	s.respondJSON(w, true, "Starting audio sequence playback", nil)
}

func (s *HTTPServer) handleStopAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	logger.Info("Stop audio requested")
	s.playbackCtrl.Stop()
	s.respondJSON(w, true, "Audio stop requested", nil)
}

func (s *HTTPServer) AddNoiseData(data map[string]interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	lowFreqRatio, ok := data["low_freq_ratio"].(float64)
	if !ok {
		logger.Warn("Invalid type for low_freq_ratio, expected float64")
	}

	volume, ok := data["volume"].(float64)
	if !ok {
		logger.Warn("Invalid type for volume, expected float64")
	}

	maxSample, ok := data["max_sample"].(float64)
	if !ok {
		logger.Warn("Invalid type for max_sample, expected float64")
	}

	t, ok := data["time"].(time.Time)
	if !ok {
		logger.Warn("Invalid type for time, expected time.Time")
		t = time.Now()
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	s.noiseData = append(s.noiseData, NoiseData{
		Timestamp:    timestamp,
		LowFreqRatio: lowFreqRatio,
		Volume:       volume,
		MaxSample:    maxSample,
		Time:         t,
	})

	if len(s.noiseData) > s.config.MaxNoiseDataEntries {
		newData := make([]NoiseData, s.config.MaxNoiseDataEntries)
		copy(newData, s.noiseData[len(s.noiseData)-s.config.MaxNoiseDataEntries:])
		s.noiseData = newData
	}

	s.recentNoiseData = append(s.recentNoiseData, NoiseData{
		Timestamp:    timestamp,
		LowFreqRatio: lowFreqRatio,
		Volume:       volume,
		MaxSample:    maxSample,
		Time:         t,
	})

	oneSecondAgo := time.Now().Add(-1 * time.Second)
	var filteredData []NoiseData
	for _, d := range s.recentNoiseData {
		if d.Time.After(oneSecondAgo) {
			filteredData = append(filteredData, d)
		}
	}
	s.recentNoiseData = filteredData

	s.needsSave = true
}

func (s *HTTPServer) SaveNoiseDataToFile() {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	if len(s.noiseData) == 0 {
		logger.Warn("No noise data to save, skipping file write")
		return
	}

	data, err := json.Marshal(s.noiseData)
	if err != nil {
		logger.Error("Error marshaling noise data: %v", err)
		return
	}

	tempFilePath := s.dataFilePath + ".tmp"

	err = ioutil.WriteFile(tempFilePath, data, 0644)
	if err != nil {
		logger.Error("Error writing noise data to temp file: %v", err)
		return
	}

	err = os.Rename(tempFilePath, s.dataFilePath)
	if err != nil {
		logger.Warn("Failed to rename temp file to %s, keeping temp file: %v", s.dataFilePath, err)
		return
	}

	logger.Info("Saved %d noise data entries to file", len(s.noiseData))
}

func (s *HTTPServer) AddDetectionLog(message string, logType string) {
	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	s.detectionLogs = append(s.detectionLogs, DetectionLog{
		Timestamp: timestamp,
		Message:   message,
		Type:      logType,
	})

	if len(s.detectionLogs) > maxDetectionLogs {
		newLogs := make([]DetectionLog, maxDetectionLogs)
		copy(newLogs, s.detectionLogs[len(s.detectionLogs)-maxDetectionLogs:])
		s.detectionLogs = newLogs
	}
}

func (s *HTTPServer) Shutdown() {
	logger.Info("Saving noise data before shutdown...")
	s.SaveNoiseDataToFile()
	logger.Info("Noise data saved successfully")
}

func (s *HTTPServer) LoadNoiseDataFromFile() {
	if _, err := os.Stat(s.dataFilePath); os.IsNotExist(err) {
		logger.Info("Noise data file not found, creating new one: %s", s.dataFilePath)
		return
	}

	data, err := ioutil.ReadFile(s.dataFilePath)
	if err != nil {
		logger.Error("Error reading noise data from file: %v", err)
		return
	}

	var noiseData []NoiseData
	err = json.Unmarshal(data, &noiseData)
	if err != nil {
		logger.Error("Error unmarshaling noise data: %v", err)
		return
	}

	s.dataMutex.Lock()
	defer s.dataMutex.Unlock()
	s.noiseData = noiseData
	logger.Info("Loaded %d noise data entries from file: %s", len(noiseData), s.dataFilePath)
}
