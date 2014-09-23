package notify

import (
	"sort"
	"strings"
	"testing"
)

func mock(m map[Event]string) func() {
	old := estr
	estr = m
	return func() {
		estr = old
	}
}

// S is a workaround for random event strings concatenation order.
func s(s string) string {
	z := strings.Split(s, "|")
	sort.StringSlice(z).Sort()
	return strings.Join(z, "|")
}

// This test is not safe to run in parallel with others.
func TestEventString(t *testing.T) {
	m := map[Event]string{
		0x01: "A",
		0x02: "B",
		0x04: "C",
		0x08: "D",
		0x0F: "E",
	}
	defer mock(m)()
	cases := map[Event]string{
		0x01: "A",
		0x03: "A|B",
		0x07: "A|B|C",
	}
	for e, str := range cases {
		if s := s(e.String()); s != str {
			t.Errorf("want s=%s; got %s (e=%#x)", str, s, e)
		}
	}
}
