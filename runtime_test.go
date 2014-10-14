package notify_test

import (
	"testing"

	. "github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestRuntime_DirectorySimple(t *testing.T) {
	ch := test.Channels(3)
	cases := [...]test.CallCase{{
		Call: test.Call{
			F: test.Watch,
			C: ch(0),
			P: "/github.com/rjeczalik/fakerpc/",
			E: Create | Delete | Move,
		},
		Record: []test.Call{{
			F: test.Watch,
			P: "/github.com/rjeczalik/fakerpc/",
			E: Delete | Create | Move,
		}},
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch(1),
			P: "/github.com/rjeczalik/fakerpc/",
			E: Delete | Move,
		},
		Record: nil,
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch(2),
			P: "/github.com/rjeczalik/fakerpc/",
			E: Move,
		},
		Record: nil,
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch(0),
			P: "/github.com/rjeczalik/fs/",
			E: Create | Delete,
		},
		Record: []test.Call{{
			F: test.Watch,
			P: "/github.com/rjeczalik/fs/",
			E: Create | Delete,
		}},
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch(0),
			P: "/github.com/rjeczalik/fs/",
			E: Create,
		},
		Record: nil,
	}, {
		Call: test.Call{
			F: test.Stop,
			C: ch(0),
		},
		Record: []test.Call{{
			F: test.Unwatch,
			P: "/github.com/rjeczalik/fakerpc/",
		}, {
			F: test.Watch,
			P: "/github.com/rjeczalik/fakerpc/",
			E: Delete | Move,
		}, {
			F: test.Unwatch,
			P: "/github.com/rjeczalik/fs/",
		}},
	}}
	test.ExpectCallsFunc(t, test.RuntimeWatcher, cases[:])
}
