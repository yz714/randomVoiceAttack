package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

const (
	logDir     = "logs"
	logFile    = "logs/voice_attack.log"
	maxSize    = 5 // megabytes
	maxBackups = 10
	maxAge     = 30 // days
)

func Init() error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   false,
	})

	consoleWriter := zapcore.AddSync(os.Stdout)

	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoderConfig.TimeKey = "time"
	consoleEncoderConfig.MessageKey = "msg"

	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.TimeKey = "time"
	fileEncoderConfig.MessageKey = "msg"
	fileEncoderConfig.LevelKey = "level"

	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)

	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, consoleWriter, zapcore.InfoLevel),
		zapcore.NewCore(fileEncoder, fileWriter, zapcore.InfoLevel),
	)

	logger = zap.New(core, zap.AddCallerSkip(1))
	sugar = logger.Sugar()

	Info("Logger initialized with zap")
	return nil
}

func Log(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Infof(format, args...)
	}
}

func Debug(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Debugf(format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Infof(format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Warnf(format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Errorf(format, args...)
	}
}

func Close() {
	if logger != nil {
		logger.Sync()
	}
}
