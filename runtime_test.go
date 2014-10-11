package notify_test

import (
	"testing"

	. "github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestRuntime_DirectorySimple(t *testing.T) {
	cases := map[EventInfo][]test.Call{
		test.EI("/github.com/rjeczalik/fakerpc/", Create|Delete|Move): {
			{
				F: test.Watch,
				P: "/github.com/rjeczalik/fakerpc/",
				E: Delete | Create | Move,
			},
		},
		test.EI("/github.com/rjeczalik/fakerpc/", Delete|Move): {},
		test.EI("/github.com/rjeczalik/fakerpc/", Move):        {},
		test.EI("/github.com/rjeczalik/fs/", Create|Create): {
			{
				F: test.Watch,
				P: "/github.com/rjeczalik/fs/",
				E: Create,
			},
		},
		test.EI("/github.com/rjeczalik/fs/", Create): {},
		test.EI(test.Unwatch, "/github.com/rjeczalik/fs/", Create): {
			{
				F: test.Unwatch,
				P: "/github.com/rjeczalik/fs/",
			},
		},
	}
	test.ExpectCallsFunc(t, test.Watcher, cases)
}
