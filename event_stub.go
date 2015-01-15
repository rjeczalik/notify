// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue

package notify

const (
	Create = EV_STUB_CREATE
	Delete = EV_STUB_DELETE
	Write  = EV_STUB_WRITE
	Move   = EV_STUB_MOVE
	Error  = EV_STUB_ERROR

	// Internal events TOOD(rjeczalik): unexport
	//
	// Recursive is used to distinguish recursive eventsets
	// from non-recursive ones.
	Recursive = EV_STUB_RECURSIVE
	// Inactive is used to bookkeep child watchpoints in the parent ones, which
	// have been set with an actual filesystem watch. This is to allow for
	// optimizing recursive watchpoint count - a single path can be watched
	// by at most 1 recursive filesystem watch.
	Inactive = EV_STUB_INACTIVE
)

// Events TODO
const (
	EV_STUB_CREATE    = Event(0x01)
	EV_STUB_DELETE    = Event(0x02)
	EV_STUB_WRITE     = Event(0x04)
	EV_STUB_MOVE      = Event(0x08)
	EV_STUB_ERROR     = Event(0x10)
	EV_STUB_RECURSIVE = Event(0x20)
	EV_STUB_INACTIVE  = Event(0x40)
	EV_STUB_ALL       = Event(0x7f)
)

var osestr = map[Event]string{}

var ekind = map[Event]Event{
	Move: Create,
}
