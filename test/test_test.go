package test

import (
	"testing"

	"github.com/rjeczalik/notify"
)

func TestEI(t *testing.T) {
	cases := [...]struct {
		want ei
		args []interface{}
	}{
		{
			ei{i: 1, p: "/tmp", e: notify.Create, f: Watch},
			[]interface{}{notify.Create, "/tmp"},
		},
		{
			ei{i: 2, p: "/var/tmp", e: notify.All, f: Watch},
			[]interface{}{"/var/tmp"},
		},
		{
			ei{i: 3, p: "/opt", e: notify.Create | notify.Delete, f: Watch},
			[]interface{}{notify.Delete, "/opt", notify.Create},
		},
		{
			ei{i: 100, p: "/home", e: notify.All, f: Watch},
			[]interface{}{ei{i: 100, p: "/home"}},
		},
		{
			ei{i: 4, p: "/", e: notify.Delete | notify.Write, f: Watch, b: true},
			[]interface{}{"will get overwritten", notify.Delete, "/", notify.Write},
		},
		{
			ei{i: 0, p: "/tmp", e: notify.All, f: Fanin},
			[]interface{}{"/home", EI(".", notify.All, EI("/tmp", Watch)), Fanin},
		},
	}
	for i, cas := range cases {
		if got := EI(cas.args...); cas.want != got {
			t.Errorf("want EI(...)=%+v; got %+v (i=%d)", cas.want, got, i)
		}
	}
}
