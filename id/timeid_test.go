package id

import (
	"testing"
)

func TestTimeID(t *testing.T) {
	tid := NewTimeID()
	a := tid.ID()
	b := tid.ID()
	if a > b {
		t.Error("time a > b")
	}
}
