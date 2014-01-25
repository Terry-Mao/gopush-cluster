package log

import (
	"testing"
)

// create a log in /tmp/test.log
func TestLog(t *testing.T) {
	logger, err := New("/tmp/test.log", "ERROR")
	if err != nil {
		logger.Error("write to stdout")
		t.Error("new a logger failed")
	}
	logger.Error("come on %d", 111)
	logger.Info("won't in log")
	if err = logger.Close(); err != nil {
		t.Error("close a logger failed")
	}
	logger, err = New("", "ERROR")
	if err != nil {
		logger.Error("write to stdout")
		t.Error("new a logger failed")
	}
	logger.Error("write to stdout, %s", "test")
}
