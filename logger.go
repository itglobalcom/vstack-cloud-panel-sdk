package sdk

import (
	"fmt"
	"io"
	"log"
)

// LogLevel represents the logging level
type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warn
	Error
)

// Logger - original interface
type Logger interface {
	Printf(format string, v ...any)
}

// LeveledLogger - wrapper over Logger with level support
type LeveledLogger struct {
	logger   Logger
	minLevel LogLevel
}

// NewLeveledLogger creates a new logger with level filtering
func NewLeveledLogger(logger Logger, minLevel LogLevel) *LeveledLogger {
	return &LeveledLogger{
		logger:   logger,
		minLevel: minLevel,
	}
}

// Debug logs a Debug level message
func (l *LeveledLogger) Debug(format string, v ...any) {
	if Debug >= l.minLevel {
		msg := fmt.Sprintf("[DEBUG] "+format, v...)
		l.logger.Printf(msg)
	}
}

// Info logs an Info level message
func (l *LeveledLogger) Info(format string, v ...any) {
	if Info >= l.minLevel {
		msg := fmt.Sprintf("[INFO] "+format, v...)
		l.logger.Printf(msg)
	}
}

// Error logs an Error level message
func (l *LeveledLogger) Error(format string, v ...any) {
	if Error >= l.minLevel {
		msg := fmt.Sprintf("[ERROR] "+format, v...)
		l.logger.Printf(msg)
	}
}

// Warn logs a warning level message
func (l *LeveledLogger) Warn(format string, v ...any) {
	if Error >= l.minLevel {
		msg := fmt.Sprintf("[Warning] "+format, v...)
		l.logger.Printf(msg)
	}
}

// NewNopLogger returns a logger that does nothing.
func NewNopLogger() Logger {
	return log.New(io.Discard, "", 0)
}
