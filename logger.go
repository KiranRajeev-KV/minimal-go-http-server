package main

import (
	"fmt"
	"os"
	"time"
)

// Log file path
const logFilePath = "./server.log"

// Single logger function with log level parameter
func log(level, format string, args ...any) {
	timestamp := time.Now().Format("2006/01/02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s: %s\n", timestamp, level, message)

	// Print to console
	fmt.Print(logEntry)

	// Append to log file
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Printf("Error opening log file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(logEntry)
	if err != nil {
		fmt.Printf("Error writing to log file: %v\n", err)
	}
}