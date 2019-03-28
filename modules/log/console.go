// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package log

import (
	"encoding/json"
	"io"
	"os"
)

type nopWriteCloser struct {
	w io.WriteCloser
}

func (n *nopWriteCloser) Write(p []byte) (int, error) {
	return n.w.Write(p)
}

func (n *nopWriteCloser) Close() error {
	return nil
}

// ConsoleLogger implements LoggerProvider and writes messages to terminal.
type ConsoleLogger struct {
	BaseLogger
	Stderr bool `json:"stderr"`
}

// NewConsoleLogger create ConsoleLogger returning as LoggerProvider.
func NewConsoleLogger() LoggerProvider {
	log := &ConsoleLogger{}
	log.createLogger(&nopWriteCloser{
		w: os.Stdout,
	})
	return log
}

// Init inits connection writer with json config.
// json config only need key "level".
func (log *ConsoleLogger) Init(config string) error {
	err := json.Unmarshal([]byte(config), log)
	if err != nil {
		return err
	}
	if log.Stderr {
		log.createLogger(&nopWriteCloser{
			w: os.Stderr,
		})
	} else {
		log.createLogger(log.out)
	}
	return nil
}

// Flush when log should be flushed
func (log *ConsoleLogger) Flush() {
}

// GetName returns the default name for this implementation
func (log *ConsoleLogger) GetName() string {
	return "console"
}

func init() {
	Register("console", NewConsoleLogger)
}
