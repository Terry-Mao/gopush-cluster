package log

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

const (
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
	defaultLogLevel = Error
	defaultLogger   = &Logger{log: log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile), file: nil, level: defaultLogLevel}
	errLevels       = []string{"EMERG", "ALERT", "CRIT", "ERROR", "WARN", "NOTIC", "INFO", "DEBUG"}
)

/*
   New creates a new Logger. The out variable sets the destination to
   which log data will be written. The prefix appears at the beginning of
   each generated log line. The file argument defines the write log file path.
   if any error the os.Stdout will return
*/
func New(file string, level int) (*Logger, error) {
	if file != "" {
		f, err := os.OpenFile(Conf.Log, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			return defaultLogger, err
		}

		logger := log.New(logFile, "", log.LstdFlags)
		return &Logger{log: logger, file: logFile, level: level}, nil
	}

	return defaultLogger, nil
}

// Close closes the open log file.
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
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

// Log use the argument level write data
func (l *Logger) Log(level int, format string, args ...interface{}) {
	if l.level >= level {
		l.logCore(level, format, args...)
	}
}

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
	l.log.Print(fmt.Sprintf("%s:%d [%s] %s", file, line, errLevels[level], fmt.Sprintf(format, args...)))
}
