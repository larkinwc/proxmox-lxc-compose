package logging

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	log *zap.SugaredLogger
	// Level is the global log level
	Level = zap.NewAtomicLevel()
)

// Config holds logging configuration
type Config struct {
	// Level is the minimum enabled logging level
	Level string
	// Development puts the logger in development mode
	Development bool
	// DisableCaller stops annotating logs with the calling function's file name and line number
	DisableCaller bool
}

// Init initializes the global logger with the given configuration
func Init(cfg Config) error {
	// Set log level
	switch cfg.Level {
	case "debug":
		Level.SetLevel(zapcore.DebugLevel)
	case "info":
		Level.SetLevel(zapcore.InfoLevel)
	case "warn":
		Level.SetLevel(zapcore.WarnLevel)
	case "error":
		Level.SetLevel(zapcore.ErrorLevel)
	default:
		return fmt.Errorf("invalid log level: %s", cfg.Level)
	}

	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	if cfg.Development {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	}
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create logger configuration
	config := zap.Config{
		Level:            Level,
		Development:      cfg.Development,
		DisableCaller:    cfg.DisableCaller,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if cfg.Development {
		config.Encoding = "console"
	}

	// Build the logger
	logger, err := config.Build()
	if err != nil {
		return fmt.Errorf("failed to build logger: %w", err)
	}

	log = logger.Sugar()
	return nil
}

// Debug logs a debug message
func Debug(msg string, keysAndValues ...interface{}) {
	log.Debugw(msg, keysAndValues...)
}

// Info logs an info message
func Info(msg string, keysAndValues ...interface{}) {
	log.Infow(msg, keysAndValues...)
}

// Warn logs a warning message
func Warn(msg string, keysAndValues ...interface{}) {
	log.Warnw(msg, keysAndValues...)
}

// Error logs an error message
func Error(msg string, keysAndValues ...interface{}) {
	log.Errorw(msg, keysAndValues...)
}

// Fatal logs a fatal message and exits
func Fatal(msg string, keysAndValues ...interface{}) {
	log.Fatalw(msg, keysAndValues...)
	os.Exit(1)
}
