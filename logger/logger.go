package logger

import (
	"fmt"
	"os"
	"time"

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

	now := time.Now()
	timestamp := now.Format("20060102_150405")
	logFileWithTimestamp := fmt.Sprintf("logs/voice_attack_%s.log", timestamp)

	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFileWithTimestamp,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   false,
	})

	consoleWriter := zapcore.AddSync(os.Stdout)

	// 控制台编码器 - 带彩色
	consoleEncoderConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoderConfig.TimeKey = "time"
	consoleEncoderConfig.MessageKey = "msg"

	// 文件编码器 - 不带彩色
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

	Log("Logger initialized with zap")
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
