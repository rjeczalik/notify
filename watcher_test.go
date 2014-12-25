// +build darwin

package notify

import "testing"

// TODO(rjeczalik): add more test-cases

func TestWatcher(t *testing.T) {
	w := newWatcherTest(t, "testdata/gopath.txt")
	defer w.Stop()

	w.Expect(create(w, "src/github.com/rjeczalik/which/.which.go.swp"))
}
