package notify

import "testing"

func TestNotify(t *testing.T) {
	n := NewNotifyTest(t, "testdata/gopath.txt")
	defer n.Close()
}

func TestStop(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
}
