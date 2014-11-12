package notify

import "strings"

// Event TODO
type Event uint32

// All TODO
const All = Create | Delete | Write | Move

var estr = map[Event]string{
	Create: "notify.Create",
	Delete: "notify.Delete",
	Write:  "notify.Write",
	Move:   "notify.Move",
	// Display name for Recursive internal event is added only for debugging
	// purposes. It's an internal event after all and won't be exposed to the
	// user. Having Recursive event printable is helpful, e.g. for reading
	// testing failure messages:
	//
	//    --- FAIL: TestWatchpoint (0.00 seconds)
	//    watchpoint_test.go:64: want diff=[notify.Delete notify.Create|notify.Delete];
	//    got [notify.Delete notify.Delete|notify.Create] (i=1)
	//
	// Yup, here the got diff have Recursive event inside. Go figure.
	Recursive: "internal.Recursive",
}

// String implements fmt.Stringer interface.
func (e Event) String() string {
	var s []string
	for _, strmap := range []map[Event]string{estr, osestr} {
		for ev, str := range strmap {
			if e&ev == ev {
				s = append(s, str)
			}
		}
	}
	return strings.Join(s, "|")
}

// EventInfo TODO
type EventInfo interface {
	Event() Event     // TODO
	IsDir() bool      // TODO(rjeczalik): Move to WatchInfo?
	Name() string     // TODO
	Sys() interface{} // TODO
}

// Kind gives generic event type of the EventInfo.Event(). The purpose is to
// hint the notify runtime whether the event created a file or directory or it
// deleted one. The possible values of Kind are Create or Delete, any other
// value is ignored by the notify runtime.
//
// TODO(rjeczalik): Unexported || Part of EventInfo?
func Kind(e Event) Event {
	switch e {
	case Create, Delete:
		return e
	default:
		ev, _ := ekind[e]
		return ev
	}
}
