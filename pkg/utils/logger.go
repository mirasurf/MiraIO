package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	infoLogger    *log.Logger
	warningLogger *log.Logger
	errorLogger   *log.Logger
	fatalLogger   *log.Logger
	debugLogger   *log.Logger
)

// InitLogger initializes the standard logger with custom settings
func InitLogger() {
	// Set log directory
	logDir := os.Getenv("MIRAIO_LOG_DIR")
	if logDir == "" {
		logDir = "/var/log/miraio"
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("Failed to create log directory: %v\n", err)
		os.Exit(1)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02-15-04-05")
	logFile := filepath.Join(logDir, fmt.Sprintf("server-%s.log", timestamp))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		os.Exit(1)
	}

	// Create multi-writer to write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, file)

	// Initialize loggers with different prefixes
	infoLogger = log.New(multiWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	warningLogger = log.New(multiWriter, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(multiWriter, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	fatalLogger = log.New(multiWriter, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)
	debugLogger = log.New(multiWriter, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	infoLogger.Printf("Logger initialized with log file: %s", logFile)
}

// LogError logs an error message
func LogError(format string, args ...interface{}) {
	errorLogger.Output(2, fmt.Sprintf(format, args...))
}

// LogWarning logs a warning message
func LogWarning(format string, args ...interface{}) {
	warningLogger.Output(2, fmt.Sprintf(format, args...))
}

// LogInfo logs an info message
func LogInfo(format string, args ...interface{}) {
	infoLogger.Output(2, fmt.Sprintf(format, args...))
}

// LogDebug logs a debug message
func LogDebug(format string, args ...interface{}) {
	debugLogger.Output(2, fmt.Sprintf(format, args...))
}

// LogFatal logs a fatal message and exits
func LogFatal(format string, args ...interface{}) {
	fatalLogger.Output(2, fmt.Sprintf(format, args...))
	os.Exit(1)
}

// Flush flushes all pending log I/O
func Flush() {
	// No-op for standard log package as it writes directly to the output
}
