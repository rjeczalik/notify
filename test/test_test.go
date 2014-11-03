package test

import (
	"path/filepath"
	"testing"

	"github.com/rjeczalik/notify"
)

var s = filepath.FromSlash

func TestEI(t *testing.T) {
	cases := [...]struct {
		args []interface{}
		want ei
	}{
		{
			[]interface{}{notify.Create, "/tmp"},
			ei{i: 1, p: s("/tmp"), e: notify.Create, f: Watch},
		},
		{
			[]interface{}{"/var/tmp"},
			ei{i: 2, p: s("/var/tmp"), e: notify.All, f: Watch},
		},
		{
			[]interface{}{notify.Delete, "/opt", notify.Create},
			ei{i: 3, p: s("/opt"), e: notify.Create | notify.Delete, f: Watch},
		},
		{
			[]interface{}{ei{i: 100, p: s("/home")}},
			ei{i: 100, p: s("/home"), e: notify.All, f: Watch},
		},
		{
			[]interface{}{"will get overwritten", notify.Delete, "/", notify.Write},
			ei{i: 4, p: s("/"), e: notify.Delete | notify.Write, f: Watch, b: true},
		},
		{
			[]interface{}{"/home", EI(".", notify.All, EI("/tmp", Watch)), Dispatch},
			ei{i: 0, p: s("/tmp"), e: notify.All, f: Dispatch},
		},
	}
	for i, cas := range cases {
		if got := EI(cas.args...); cas.want != got {
			t.Errorf("want EI(...)=%+v; got %+v (i=%d)", cas.want, got, i)
		}
	}
}
