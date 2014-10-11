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
		test.EI("/github.com/rjeczalik/fakerpc/LICENSE", notify.Write): {
			{F: test.Watch, P: "/github.com/rjeczalik/fakerpc/LICENSE", E: notify.Write},
		},
		test.EI(test.Unwatch, "/github.com/rjeczalik/fakerpc/LICENSE", notify.Write): {
			{F: test.Unwatch, P: "/github.com/rjeczalik/fakerpc/LICENSE"},
		},
	}
	test.ExpectCallsFunc(t, test.Watcher, cases)
}
