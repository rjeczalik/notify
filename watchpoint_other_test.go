package notify

import "testing"

func TestMatches(t *testing.T) {
	cases := [...]struct {
		set   Event
		event Event
		ok    bool
	}{
		{Create | Write, Write, true},                                        // i=0
		{Create | Write, Remove, false},                                      // i=1
		{Create | Write, Create | recursive, false},                          // i=2
		{Create | omit, Create, false},                                       // i=3
		{Create, Create | omit, false},                                       // i=4
		{Create | omit, Create | omit, true},                                 // i=5
		{Create | Write | recursive | omit, Create | recursive | omit, true}, // i=6
		{Create | Write | Remove | recursive | omit, Create | omit, true},    // i=7
		{Create | Write | Remove | omit, Create | recursive | omit, false},   // i=8
	}
	for i, cas := range cases {
		if ok := matches(cas.set, cas.event); ok != cas.ok {
			t.Errorf("want matches(%v, %v)=%v; got %v (i=%d)", cas.set, cas.event, cas.ok, ok, i)
		}
	}
}
