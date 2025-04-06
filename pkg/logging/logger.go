package logging

import (
	"bytes"
	"fmt"
	"io"
	"os"
)

// LogLevel defines the severity of the message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// NewMockLogger returns a convenient mock logger for testing
func NewMockLogger() *DefaultLogger {
	return &DefaultLogger{
		writer: bytes.NewBufferString(""),
		level:  INFO,
	}
}

// Logger interface defines logging operations
//
//go:generate mockery --name=Logger --output=./mocks
type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
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
func (l *DefaultLogger) Debug(format string, args ...interface{}) {
	if l.level <= DEBUG {
		l.log("DEBUG", format, args...)
	}
}

// Info logs informational messages
func (l *DefaultLogger) Info(format string, args ...interface{}) {
	if l.level <= INFO {
		l.log("INFO", format, args...)
	}
}

// Warn logs warning messages
func (l *DefaultLogger) Warn(format string, args ...interface{}) {
	if l.level <= WARN {
		l.log("WARN", format, args...)
	}
}

// Error logs error messages
func (l *DefaultLogger) Error(format string, args ...interface{}) {
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
func (l *DefaultLogger) log(level, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	logLine := fmt.Sprintf("[%s]: %s\n", level, message)
	fmt.Fprint(l.writer, logLine)
}
