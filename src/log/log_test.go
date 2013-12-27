package log

import (
	"testing"
)

// create a log in /tmp/test.log
func TestLog(t *testing.T) {
	logger, err := New("/tmp/test.log", Error)
	if err != nil {
		logger.Error("write to stdout")
		t.Errorf("new a logger failed")
	}

	logger.Error("come on %d", 111)
	logger.Info("won't in log")
	if err = logger.Close(); err != nil {
		t.Errorf("close a logger failed")
	}

	logger, err = New("", Error)
	if err != nil {
		logger.Error("write to stdout")
		t.Errorf("new a logger failed")
	}

	logger.Error("write to stdout, %s", "test")
}
