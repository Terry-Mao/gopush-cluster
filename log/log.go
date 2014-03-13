// Copyright © 2014 Terry Mao, LiuDing All rights reserved.
// This file is part of gopush-cluster.

// gopush-cluster is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// gopush-cluster is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with gopush-cluster.  If not, see <http://www.gnu.org/licenses/>.

package log

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

const (
	// Log Level
	Emerg = 0
	Alert = 1
	Crit  = 2
	Error = 3
	Warn  = 4
	Notic = 5
	Info  = 6
	Debug = 7
)

/*
   A Logger represents an active logging object that generates lines of output
   to an io.Writer. Each logging operation makes a single call to the
   Writer's Write method. A Logger can be used simultaneously from multiple
   goroutines; it guarantees to serialize access to the Writer.
*/
type Logger struct {
	log   *log.Logger
	file  *os.File
	level int
}

var (
	defaultLogLevel = Debug
	DefaultLogger   = &Logger{log: log.New(os.Stdout, "", log.LstdFlags), file: nil, level: defaultLogLevel}
	errLevels       = []string{"EMERG", "ALERT", "CRIT", "ERROR", "WARN", "NOTIC", "INFO", "DEBUG"}
)

/*
   New creates a new Logger. The out variable sets the destination to
   which log data will be written. The prefix appears at the beginning of
   each generated log line. The file argument defines the write log file path.
   if any error the os.Stdout will return
*/
func New(file string, levelStr string) (*Logger, error) {
	level := defaultLogLevel
	for lv, str := range errLevels {
		if str == levelStr {
			level = lv
		}
	}
	if file != "" {
		f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return DefaultLogger, err
		}
		logger := log.New(f, "", log.LstdFlags)
		return &Logger{log: logger, file: f, level: level}, nil
	}
	return DefaultLogger, nil
}

// Close closes the open log file.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Error use the Error log level write data
func (l *Logger) Error(format string, args ...interface{}) {
	if l.level >= Error {
		l.logCore(Error, format, args...)
	}
}

// Debug use the Error log level write data
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.level >= Debug {
		l.logCore(Debug, format, args...)
	}
}

// Info use the Info log level write data
func (l *Logger) Info(format string, args ...interface{}) {
	if l.level >= Info {
		l.logCore(Info, format, args...)
	}
}

// Warn use the Warn level write data
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.level >= Warn {
		l.logCore(Warn, format, args...)
	}
}

// Crit use the Crit level write data
func (l *Logger) Crit(format string, args ...interface{}) {
	if l.level >= Crit {
		l.logCore(Crit, format, args...)
	}
}

// logCore handle the core log proc
func (l *Logger) logCore(level int, format string, args ...interface{}) {
	var (
		file string
		line int
		ok   bool
	)
	_, file, line, ok = runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	short := file
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			short = file[i+1:]
			break
		}
	}
	file = short
	l.log.Print(fmt.Sprintf("[%s] %s:%d %s", errLevels[level], file, line, fmt.Sprintf(format, args...)))
}
