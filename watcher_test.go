package notify_test

import (
	"testing"

	"github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestWatcherBasic(t *testing.T) {
	ei := []notify.EventInfo{
		test.EI("github.com/rjeczalik/fs/fs_test.go", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs/", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs.go", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs_test.go", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs/", notify.Delete),
		test.EI("github.com/rjeczalik/fs/binfs/", notify.Create),
		test.EI("github.com/rjeczalik/fs/virfs", notify.Create),
		test.EI("github.com/rjeczalik/fs/virfs", notify.Delete),
		test.EI("file", notify.Create),
		test.EI("dir/", notify.Create),
	}
	test.ExpectEvent(t, notify.NewWatcher(), notify.All, ei)
}

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
	test.ExpectGroupEvents(t, notify.NewWatcher(), notify.All, ei)
}
