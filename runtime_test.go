package notify_test

import (
	"testing"

	"github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestRuntime(t *testing.T) {
	cases := map[notify.EventInfo][]test.Call{
		test.EI("/github.com/rjeczalik/fakerpc", notify.Delete): {
			{F: test.Watch, P: "/github.com/rjeczalik/fakerpc", E: notify.Delete},
			// },
			// test.EI("/github.com/rjeczalik/fakerpc/...", notify.Delete): {
			// {F: test.Watch, P: "/github.com/rjeczalik/fakerpc", E: notify.Delete},
			// {F: test.Watch, P: "/github.com/rjeczalik/fakerpc/cli", E: notify.Delete},
			// {F: test.Watch, P: "/github.com/rjeczalik/fakerpc/cmd", E: notify.Delete},
			// },
			// test.EI("/github.com/rjeczalik/fakerpc/LICENSE", notify.Write): {
			// {F: test.Watch, P: "/github.com/rjeczalik/fakerpc/LICENSE", E: notify.Write},
		}}
	test.ExpectCalls(t, cases)
}
