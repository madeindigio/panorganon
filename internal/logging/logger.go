package logging

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
	"github.com/sevir/panorganon/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger initializes and returns a zap logger
func InitLogger(cfg config.LoggingConfig) (*zap.Logger, error) {
	// Parse log level
	level, err := parseLogLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create core
	var core zapcore.Core

	if cfg.File != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create log directory: %v, falling back to stderr\n", err)
			core = zapcore.NewCore(
				zapcore.NewJSONEncoder(encoderConfig),
				zapcore.AddSync(os.Stderr),
				level,
			)
		} else {
			// Set up log rotation with lumberjack
			writer := &lumberjack.Logger{
				Filename:   cfg.File,
				MaxSize:    100, // megabytes
				MaxBackups: 10,
				MaxAge:     30, // days
				Compress:   true,
			}

			// Write to both file and stderr for errors
			fileWriter := zapcore.AddSync(writer)
			consoleWriter := zapcore.AddSync(os.Stderr)

			core = zapcore.NewTee(
				zapcore.NewCore(
					zapcore.NewJSONEncoder(encoderConfig),
					fileWriter,
					level,
				),
				zapcore.NewCore(
					zapcore.NewConsoleEncoder(encoderConfig),
					consoleWriter,
					zapcore.ErrorLevel, // Only errors to stderr
				),
			)
		}
	} else {
		// No file specified, write to stderr only
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stderr),
			level,
		)
	}

	// Create logger
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return logger, nil
}

// parseLogLevel converts string level to zapcore.Level
func parseLogLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("invalid log level: %s", level)
	}
}

// WithComponent returns a logger with component field
func WithComponent(logger *zap.Logger, component string) *zap.Logger {
	return logger.With(zap.String("component", component))
}

// WithServer returns a logger with server_name field
func WithServer(logger *zap.Logger, serverName string) *zap.Logger {
	return logger.With(zap.String("server_name", serverName))
}

// WithTool returns a logger with tool_name field
func WithTool(logger *zap.Logger, toolName string) *zap.Logger {
	return logger.With(zap.String("tool_name", toolName))
}
