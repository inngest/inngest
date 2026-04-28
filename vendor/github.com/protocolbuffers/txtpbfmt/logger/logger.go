// Package logger provides a logger interface for the parser.
package logger

// Logger is a small glog-like interface.
type Logger interface {
	// Infof is used for informative messages, for testing or debugging.
	Infof(format string, args ...any)
}
