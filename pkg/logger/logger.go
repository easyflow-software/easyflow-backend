package logger

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type termColors string

const (
	// Reset
	reset termColors = "\033[0m"
	// Regular Colors
	black     termColors = "\033[0;30m"
	red       termColors = "\033[0;31m"
	yellow    termColors = "\033[0;33m"
	blue      termColors = "\033[0;34m"
	green     termColors = "\033[0;32m"
	lightBlue termColors = "\033[0;36m"
)

type LogLevel string

const (
	SUCCESS LogLevel = "SUCCESS"
	DEBUG   LogLevel = "DEBUG"
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARNING"
	ERROR   LogLevel = "ERROR"
)

type GetIP func(...interface{}) string

type Logger struct {
	LogMutex sync.Mutex
	Target   io.Writer
	Module   atomic.Value
	logLevel LogLevel
	ip       string
}

func NewLogger(target io.Writer, module string, logLevel LogLevel, ip string) *Logger {
	logger := &Logger{
		Target:   target,
		logLevel: logLevel,
		ip:       ip,
	}
	logger.Module.Store(module)
	return logger
}

func (l *Logger) SetPrefix(prefix string) {
	l.Module.Store(prefix)
}

func getLocalTime() string {
	return time.Now().Format("2006-01-02 - 15:04:05")
}

func (l *Logger) baseLogger(color termColors, logType LogLevel, format string, args ...interface{}) {
	l.LogMutex.Lock()
	defer l.LogMutex.Unlock()

	module_name := l.Module.Load().(string)

	formattedMessage := fmt.Sprintf(format, args...)
	//GENERAL SCHEMA:
	// {color}[KIND][TIME][IP][MODULE] MESSAGE{reset}
	_, _ = l.Target.Write([]byte(string(color) + "[" + string(logType) + "]" + "[" + getLocalTime() + "][" + l.ip + "][" + module_name + "] " + formattedMessage + string(reset) + "\n"))
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.baseLogger(green, SUCCESS, format, args...)
}

func (l *Logger) PrintfError(format string, args ...interface{}) {
	l.baseLogger(red, ERROR, format, args...)
}

func (l *Logger) PrintfWarning(format string, args ...interface{}) {
	if l.logLevel == LogLevel(WARNING) || l.logLevel == LogLevel(INFO) || l.logLevel == LogLevel(DEBUG) {
		l.baseLogger(yellow, WARNING, format, args...)
	}
}

func (l *Logger) PrintfInfo(format string, args ...interface{}) {
	if l.logLevel == LogLevel(INFO) || l.logLevel == LogLevel(DEBUG) {
		l.baseLogger(blue, INFO, format, args...)
	}
}

func (l *Logger) PrintfDebug(format string, args ...interface{}) {
	if l.logLevel == LogLevel(DEBUG) {
		l.baseLogger(lightBlue, DEBUG, format, args...)
	}
}
