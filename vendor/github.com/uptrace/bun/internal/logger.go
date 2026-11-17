package internal

import (
	"fmt"
	"log"
	"os"
)

type Logging interface {
	Printf(format string, v ...any)
}

var defaultLogger = log.New(os.Stderr, "", log.LstdFlags)

var Logger Logging = &logger{
	log: defaultLogger,
}

var Warn = &wrapper{
	prefix: "WARN: bun: ",
	logger: Logger,
}

var Deprecated = &wrapper{
	prefix: "DEPRECATED: bun: ",
	logger: Logger,
}

type logger struct {
	log *log.Logger
}

func (l *logger) Printf(format string, v ...any) {
	_ = l.log.Output(2, fmt.Sprintf(format, v...))
}

type wrapper struct {
	prefix string
	logger Logging
}

func (w *wrapper) Printf(format string, v ...any) {
	w.logger.Printf(w.prefix+format, v...)
}

func SetLogger(newLogger Logging) {
	if newLogger == nil {
		Logger = &logger{log: defaultLogger}
	} else {
		Logger = newLogger
	}
	Warn.logger = Logger
	Deprecated.logger = Logger
}
