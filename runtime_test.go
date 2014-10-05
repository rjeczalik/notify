package notify_test

import (
	"testing"

	"github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestRuntimeBasicWatcher(t *testing.T) {
	cases := map[notify.EventInfo][]test.Call{
		test.EI("/github.com/rjeczalik/fakerpc/", notify.Delete): {
			{F: test.Watch, P: "/github.com/rjeczalik/fakerpc/", E: notify.Delete},
		},
		// TODO(rjeczalik): Uncomment after Recursive implementation is complete.
		// test.EI("/github.com/rjeczalik/fakerpc/...", notify.Delete): {
		//   {F: test.Watch, P: "/github.com/rjeczalik/fakerpc", E: notify.Delete},
		//   {F: test.Watch, P: "/github.com/rjeczalik/fakerpc/cli", E: notify.Delete},
		//   {F: test.Watch, P: "/github.com/rjeczalik/fakerpc/cmd", E: notify.Delete},
		// },
		test.EI("/github.com/rjeczalik/fakerpc/LICENSE", notify.Write): {
			{F: test.Watch, P: "/github.com/rjeczalik/fakerpc/LICENSE", E: notify.Write},
		},
		test.EI(test.Unwatch, "/github.com/rjeczalik/fakerpc/LICENSE", notify.Write): {
			{F: test.Unwatch, P: "/github.com/rjeczalik/fakerpc/LICENSE"},
		},
	}
	test.ExpectCallsFunc(t, test.Watcher, cases)
}

func TestRuntimeBasicRewatcher(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
	cases := map[notify.EventInfo][]test.Call{}
	test.ExpectCallsFunc(t, test.Rewatcher, cases)
}

func TestRuntimeBasicRecursiveWatcher(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
	cases := map[notify.EventInfo][]test.Call{}
	test.ExpectCallsFunc(t, test.RecursiveWatcher, cases)
}

func TestRuntimeBasicInterface(t *testing.T) {
	t.Skip("TODO(rjeczalik)")
	cases := map[notify.EventInfo][]test.Call{}
	test.ExpectCalls(t, cases)
}

// TODO(rjeczalik)
func TestRuntimeExpandOrSchrinkEventSet(t *testing.T) {
	t.Skip("(wip)")
	cases := map[notify.EventInfo][]test.Call{
		test.EI("/github.com/rjeczalik/fs/", notify.Create): {
			{F: test.Watch, P: "/github.com/rjeczalik/fs/", E: notify.Create},
		},
		test.EI("/github.com/rjeczalik/fs/", notify.Delete): {
			{F: test.Rewatch, P: "/github.com/rjeczalik/fs/", E: notify.Create,
				N: notify.Create | notify.Delete},
		},
		test.EI(test.Unwatch, "/github.com/rjeczalik/fs/", notify.Create): {
			{F: test.Rewatch, P: "/github.com/rjeczalik/fs/", E: notify.Create | notify.Delete,
				N: notify.Delete},
		},
	}
	test.ExpectCalls(t, cases)
}

// TODO(rjeczalik)
func TestRuntimeMoveFilesToDirWatchpoint(t *testing.T) {
	cases := map[notify.EventInfo][]test.Call{}
	test.ExpectCalls(t, cases)
}

// TODO(rjeczalik)
func TestRuntimeWatchpointReferenceCount(t *testing.T) {
	cases := map[notify.EventInfo][]test.Call{}
	test.ExpectCalls(t, cases)
}
