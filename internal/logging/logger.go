package logging

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel defines the severity of the message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger interface defines logging operations
//
//go:generate mockery --name=Logger --output=./mocks
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	SetOutput(w io.Writer)
	SetLevel(level LogLevel)
}

// DefaultLogger provides a standard implementation
type DefaultLogger struct {
	writer io.Writer
	level  LogLevel
}

// NewDefaultLogger creates a new logger instance
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		writer: os.Stdout,
		level:  INFO,
	}
}

// Debug logs debug messages
func (l *DefaultLogger) Debug(format string, args ...any) {
	if l.level <= DEBUG {
		l.log("DEBUG", format, args...)
	}
}

// Info logs informational messages
func (l *DefaultLogger) Info(format string, args ...any) {
	if l.level <= INFO {
		l.log("INFO", format, args...)
	}
}

// Warn logs warning messages
func (l *DefaultLogger) Warn(format string, args ...any) {
	if l.level <= WARN {
		l.log("WARN", format, args...)
	}
}

// Error logs error messages
func (l *DefaultLogger) Error(format string, args ...any) {
	if l.level <= ERROR {
		l.log("ERROR", format, args...)
	}
}

// SetOutput sets the output destination for the logger
func (l *DefaultLogger) SetOutput(w io.Writer) {
	l.writer = w
}

// SetLevel sets the logging level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// log formats and writes a log message
func (l *DefaultLogger) log(level, format string, args ...any) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)
	fmt.Fprint(l.writer, logLine)
}

// StringToLogLevel converts a string representation to a LogLevel
func StringToLogLevel(level string) LogLevel {
	switch level {
	case "debug":
		return DEBUG
	case "info":
		return INFO
	case "warn":
		return WARN
	case "error":
		return ERROR
	default:
		return INFO
	}
}
