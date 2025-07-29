// Package k8shard provides automation for setting up Kubernetes clusters.
// logger.go implements the Logger interface for console-based logging.
package clustersetup

import "fmt"

// NewLogger creates a new logger
func NewLogger() Logger {
	return &consoleLogger{}
}

// consoleLogger is a simple console-based logger
type consoleLogger struct{}

func (l *consoleLogger) Info(msg string, args ...interface{})  { fmt.Printf("INFO: "+msg+"\n", args...) }
func (l *consoleLogger) Error(msg string, args ...interface{}) { fmt.Printf("ERROR: "+msg+"\n", args...) }
func (l *consoleLogger) Debug(msg string, args ...interface{}) { fmt.Printf("DEBUG: "+msg+"\n", args...) }
func (l *consoleLogger) Warn(msg string, args ...interface{})  { fmt.Printf("WARN: "+msg+"\n", args...) }