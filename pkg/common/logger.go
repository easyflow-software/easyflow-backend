package common

import (
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
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
	DEBUG   LogLevel = "DEBUG"
	INFO    LogLevel = "INFO"
	WARNING LogLevel = "WARNING"
	ERROR   LogLevel = "ERROR"
)

type Logger struct {
	LogMutex sync.Mutex
	Target   io.Writer
	Module   atomic.Value
	C        *gin.Context
	logLevel LogLevel
}

//GENERAL SCHEMA:
// {color}[KIND][TIME][MODULE] MESSAGE{reset}

func NewLogger(target io.Writer, module string, c *gin.Context, logLevel LogLevel) *Logger {
	logger := &Logger{
		Target:   target,
		C:        c,
		logLevel: logLevel,
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

func getIP(c *gin.Context) string {
	var clientIP string
	if c != nil {
		clientIP = c.ClientIP()
	} else {
		clientIP = "System"
	}
	return clientIP
}

func (l *Logger) Println(message interface{}) {
	l.LogMutex.Lock()
	defer l.LogMutex.Unlock()

	module_name := l.Module.Load().(string)
	_, _ = l.Target.Write([]byte(string(green) + "[" + "SUCCESS" + "]" + "[" + getLocalTime() + "][" + getIP(l.C) + "][" + module_name + "] " + message.(string) + string(reset) + "\n"))
}

func (l *Logger) Printf(format string, args ...interface{}) {
	l.LogMutex.Lock()
	defer l.LogMutex.Unlock()

	module_name := l.Module.Load().(string)
	formattedMessage := fmt.Sprintf(format, args...)
	_, _ = l.Target.Write([]byte(string(green) + "[" + "SUCCESS" + "]" + "[" + getLocalTime() + "][" + getIP(l.C) + "][" + module_name + "] " + formattedMessage + string(reset) + "\n"))
}

func (l *Logger) PrintfError(format string, args ...interface{}) {
	l.LogMutex.Lock()
	defer l.LogMutex.Unlock()

	module_name := l.Module.Load().(string)
	formattedMessage := fmt.Sprintf(format, args...)
	_, _ = l.Target.Write([]byte(string(red) + "[" + "ERROR" + "]" + "[" + getLocalTime() + "][" + getIP(l.C) + "][" + module_name + "] " + formattedMessage + string(reset) + "\n"))
}

func (l *Logger) PrintfWarning(format string, args ...interface{}) {
	if l.logLevel == LogLevel(WARNING) || l.logLevel == LogLevel(INFO) || l.logLevel == LogLevel(DEBUG) {
		l.LogMutex.Lock()
		defer l.LogMutex.Unlock()

		module_name := l.Module.Load().(string)
		formattedMessage := fmt.Sprintf(format, args...)
		_, _ = l.Target.Write([]byte(string(yellow) + "[" + "WARNING" + "]" + "[" + getLocalTime() + "][" + getIP(l.C) + "][" + module_name + "] " + formattedMessage + string(reset) + "\n"))
	}
}

func (l *Logger) PrintfInfo(format string, args ...interface{}) {
	if l.logLevel == LogLevel(INFO) || l.logLevel == LogLevel(DEBUG) {
		l.LogMutex.Lock()
		defer l.LogMutex.Unlock()

		module_name := l.Module.Load().(string)
		formattedMessage := fmt.Sprintf(format, args...)
		_, _ = l.Target.Write([]byte(string(blue) + "[" + "INFO" + "]" + "[" + getLocalTime() + "][" + getIP(l.C) + "][" + module_name + "] " + formattedMessage + string(reset) + "\n"))
	}
}

func (l *Logger) PrintfDebug(format string, args ...interface{}) {
	if l.logLevel == LogLevel(DEBUG) {
		l.LogMutex.Lock()
		defer l.LogMutex.Unlock()

		module_name := l.Module.Load().(string)
		formattedMessage := fmt.Sprintf(format, args...)
		_, _ = l.Target.Write([]byte(string(lightBlue) + "[" + "DEBUG" + "]" + "[" + getLocalTime() + "][" + getIP(l.C) + "][" + module_name + "] " + formattedMessage + string(reset) + "\n"))
	}
}
