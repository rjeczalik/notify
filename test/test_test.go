package test

import (
	"path/filepath"
	"testing"

	"github.com/rjeczalik/notify"
)

var p = filepath.FromSlash

func TestEI(t *testing.T) {
	cases := [...]struct {
		args []interface{}
		want ei
	}{
		{
			[]interface{}{notify.Create, "/tmp"},
			ei{i: 1, p: p("/tmp"), e: notify.Create, f: Watch},
		},
		{
			[]interface{}{"/var/tmp"},
			ei{i: 2, p: p("/var/tmp"), e: notify.All, f: Watch},
		},
		{
			[]interface{}{notify.Delete, "/opt", notify.Create},
			ei{i: 3, p: p("/opt"), e: notify.Create | notify.Delete, f: Watch},
		},
		{
			[]interface{}{ei{i: 100, p: p("/home")}},
			ei{i: 100, p: p("/home"), e: notify.All, f: Watch},
		},
		{
			[]interface{}{"will get overwritten", notify.Delete, "/", notify.Write},
			ei{i: 4, p: p("/"), e: notify.Delete | notify.Write, f: Watch, b: true},
		},
		{
			[]interface{}{"/home", EI(".", notify.All, EI("/tmp", Watch)), Dispatch},
			ei{i: 0, p: p("/tmp"), e: notify.All, f: Dispatch},
		},
	}
	for i, cas := range cases {
		if got := EI(cas.args...); cas.want != got {
			t.Errorf("want EI(...)=%+v; got %+v (i=%d)", cas.want, got, i)
		}
	}
}
