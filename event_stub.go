// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue

package notify

import "errors"

const (
	Create = EvStubCreate
	Delete = EvStubDelete
	Write  = EvStubWrite
	Move   = EvStubMove
	Error  = EvStubError

	// Internal events TOOD(rjeczalik): unexport
	//
	// Recursive is used to distinguish recursive eventsets
	// from non-recursive ones.
	recursive = EvStubRecursive
)

// Events TODO
const (
	EvStubCreate    = Event(0x01)
	EvStubDelete    = Event(0x02)
	EvStubWrite     = Event(0x04)
	EvStubMove      = Event(0x08)
	EvStubError     = Event(0x10)
	EvStubRecursive = Event(0x20)
	EvStubAll       = Event(0x7f)
)

var osestr = map[Event]string{}

var ekind = map[Event]Event{
	Move: Create,
}

func isdir(EventInfo) (bool, error) {
	return false, errors.New("not implemented")
}
