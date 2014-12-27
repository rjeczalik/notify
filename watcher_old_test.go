// +build ignore

package notify_test

import (
	"testing"

	"github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestWatcherGroupEvents(t *testing.T) {
	ei := [][]notify.EventInfo{
		[]notify.EventInfo{
			test.EI("github.com/rjeczalik/fs/cmd/gotree/f.go", notify.Create),
		},
		[]notify.EventInfo{
			test.EI("github.com/rjeczalik/fs/cmd/gotree/f2.go", notify.Create),
		},
		[]notify.EventInfo{
			test.EI("github.com/rjeczalik/fs/cmd/gotree/dir/", notify.Create),
		},
		[]notify.EventInfo{
			test.EI("github.com/rjeczalik/fs/cmd/gotree/", notify.Delete),
			test.EI("github.com/rjeczalik/fs/cmd/gotree/f.go", notify.Delete),
			test.EI("github.com/rjeczalik/fs/cmd/gotree/f2.go", notify.Delete),
			test.EI("github.com/rjeczalik/fs/cmd/gotree/dir/", notify.Delete),
			test.EI("github.com/rjeczalik/fs/cmd/gotree/go.go", notify.Delete),
			test.EI("github.com/rjeczalik/fs/cmd/gotree/main.go", notify.Delete),
		},
	}
	test.ExpectGroupEvents(t, notify.NewWatcher(), notify.All, ei[:0])
}
