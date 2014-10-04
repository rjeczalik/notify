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
