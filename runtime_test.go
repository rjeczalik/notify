package notify_test

import (
	"testing"

	. "github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestRuntime_Directory(t *testing.T) {
	scope, ch := test.R(t), test.Chans(3)
	calls := [...]test.CallCase{{
		Call: test.Call{
			F: test.Watch,
			C: ch[0],
			P: "/github.com/rjeczalik/fakerpc/",
			E: Create | Delete | Move,
		},
		Record: test.Record{
			test.All: {{
				F: test.Watch,
				P: "/github.com/rjeczalik/fakerpc/",
				E: Delete | Create | Move,
			}}},
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch[1],
			P: "/github.com/rjeczalik/fakerpc/",
			E: Delete | Move,
		},
		Record: nil,
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch[2],
			P: "/github.com/rjeczalik/fakerpc/",
			E: Move,
		},
		Record: nil,
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch[0],
			P: "/github.com/rjeczalik/fs/",
			E: Create | Delete,
		},
		Record: test.Record{
			test.All: {{
				F: test.Watch,
				P: "/github.com/rjeczalik/fs/",
				E: Create | Delete,
			}}},
	}, {
		Call: test.Call{
			F: test.Watch,
			C: ch[0],
			P: "/github.com/rjeczalik/fs/",
			E: Create,
		},
		Record: nil,
	}, {
		Call: test.Call{
			F: test.Stop,
			C: ch[0],
		},
		Record: test.Record{
			test.Watcher: {{
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
			test.Rewatcher | test.Recursive: {{
				F:  test.Rewatch,
				P:  "/github.com/rjeczalik/fakerpc/",
				E:  Create | Delete | Move,
				NE: Delete | Move,
			}, {
				F: test.Unwatch,
				P: "/github.com/rjeczalik/fs/",
			}},
		},
	}}
	events := [...]test.EventCase{{
		Event: test.Event{
			P: "/github.com/rjeczalik/fakerpc/.fakerpc.go.swp",
			E: Delete,
		},
		Receiver: test.Chans(ch[1]),
	}, {
		Event: test.Event{
			P: "/github.com/rjeczalik/fakerpc/.travis.yml",
			E: Move,
		},
		Receiver: test.Chans(ch[1], ch[2]),
	}}
	scope.ExpectCalls(calls[:])
	scope.ExpectEvents(events[:])
}

func TestRuntime_RecursiveDirectory(t *testing.T) {
	t.Skip("currently WIP in watchpointtree.go")
	scope, ch := test.R(t, test.Watcher), test.Chans(5)
	calls := [...]test.CallCase{{
		Call: test.Call{
			F: test.Watch,
			C: ch[0],
			P: "/github.com/rjeczalik/fakerpc/...",
			E: Create | Delete,
		},
		Record: test.Record{
			test.Watcher: {{
				F: test.Watch,
				P: "/github.com/rjeczalik/fakerpc/",
				E: Create | Delete,
			}, {
				F: test.Watch,
				P: "/github.com/rjeczalik/fakerpc/cli/",
				E: Create | Delete,
			}, {
				F: test.Watch,
				P: "/github.com/rjeczalik/fakerpc/cmd",
				E: Create | Delete,
			}, {
				F: test.Watch,
				P: "/github.com/rjeczalik/fakerpc/cmd/fakerpc",
				E: Create | Delete,
			}},
		},
	}}
	scope.ExpectCalls(calls[:])
}
