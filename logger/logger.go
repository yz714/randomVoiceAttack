// Package logger provides logging functionality with both file and console output.
package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger defines the interface for logging operations.
type Logger interface {
	// Init initializes the logger.
	Init() error
	// Log logs a message at info level (deprecated, use Info instead).
	Log(format string, args ...interface{})
	// Debug logs a message at debug level.
	Debug(format string, args ...interface{})
	// Info logs a message at info level.
	Info(format string, args ...interface{})
	// Warn logs a message at warning level.
	Warn(format string, args ...interface{})
	// Error logs a message at error level.
	Error(format string, args ...interface{})
	// Close flushes any buffered log entries.
	Close()
}

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

const (
	logDir     = "logs"
	logFile    = "logs/voice_attack.log"
	maxSize    = 5  // megabytes
	maxBackups = 10 // number of backups
	maxAge     = 30 // days to retain old logs
)

// Init initializes the logger with both file and console output.
// It sets up log rotation and creates the log directory if it doesn't exist.
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

// Log logs a message at info level (deprecated, use Info instead).
func Log(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Infof(format, args...)
	}
}

// Debug logs a message at debug level.
func Debug(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Debugf(format, args...)
	}
}

// Info logs a message at info level.
func Info(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Infof(format, args...)
	}
}

// Warn logs a message at warning level.
func Warn(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Warnf(format, args...)
	}
}

// Error logs a message at error level.
func Error(format string, args ...interface{}) {
	if sugar != nil {
		sugar.Errorf(format, args...)
	}
}

// Close flushes any buffered log entries.
func Close() {
	if logger != nil {
		logger.Sync()
	}
}
