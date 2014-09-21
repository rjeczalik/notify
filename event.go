package notify

import "strings"

// Event TODO
type Event uint32

// String implements fmt.Stringer interface.
func (e Event) String() string {
	var s []string
	for ev, str := range estr {
		if e&ev == ev {
			s = append(s, str)
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
