package notify

import (
	"sort"
	"strings"
	"testing"
)

// S is a workaround for random event strings concatenation order.
func s(s string) string {
	z := strings.Split(s, "|")
	sort.StringSlice(z).Sort()
	return strings.Join(z, "|")
}

// This test is not safe to run in parallel with others.
func TestEventString(t *testing.T) {
	cases := map[Event]string{
		Create:                  "notify.Create",
		Create | Delete:         "notify.Create|notify.Delete",
		Create | Delete | Write: "notify.Create|notify.Delete|notify.Write",
		Create | Write | Move:   "notify.Create|notify.Move|notify.Write",
	}
	for e, str := range cases {
		if s := s(e.String()); s != str {
			t.Errorf("want s=%s; got %s (e=%#x)", str, s, e)
		}
	}
}
