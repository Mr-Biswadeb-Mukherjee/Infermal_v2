package logger

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Logger struct
type Logger struct {
	mu       sync.Mutex
	filePath string
}

// NewLogger creates a new logger instance with dynamic file name
// Example: NewLogger("backup") -> "backup_2025-09-22_12-50-30.log"
func NewLogger(scenario string) *Logger {
	fileName := scenario
	if !strings.HasSuffix(strings.ToLower(scenario), ".log") {
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		fileName = fmt.Sprintf("%s_%s.log", scenario, timestamp)
	}
	return &Logger{
		filePath: fileName,
	}
}

// Info logs info-level messages
func (l *Logger) Info(format string, v ...interface{}) {
	l.writeLog("INFO", format, v...)
}

// Warning logs warning-level messages
func (l *Logger) Warning(format string, v ...interface{}) {
	l.writeLog("WARNING", format, v...)
}

// Alert logs alert-level messages
func (l *Logger) Alert(format string, v ...interface{}) {
	l.writeLog("ALERT", format, v...)
}

// writeLog writes the log entry to the file
func (l *Logger) writeLog(level, format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, v...)
	logLine := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, msg)

	// Open file in append mode, create if not exists
	f, err := os.OpenFile(l.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err) // Only in extreme case
		return
	}
	defer f.Close()

	_, _ = f.WriteString(logLine)
}

// GetFilePath returns the path of the log file
func (l *Logger) GetFilePath() string {
	return l.filePath
}
