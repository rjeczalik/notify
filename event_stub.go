// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue

package notify

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
	// Inactive is used to bookkeep child watchpoints in the parent ones, which
	// have been set with an actual filesystem watch. This is to allow for
	// optimizing recursive watchpoint count - a single path can be watched
	// by at most 1 recursive filesystem watch.
	inactive = EvStubInactive
)

// Events TODO
const (
	EvStubCreate    = Event(0x01)
	EvStubDelete    = Event(0x02)
	EvStubWrite     = Event(0x04)
	EvStubMove      = Event(0x08)
	EvStubError     = Event(0x10)
	EvStubRecursive = Event(0x20)
	EvStubInactive  = Event(0x40)
	EvStubAll       = Event(0x7f)
)

var osestr = map[Event]string{}

var ekind = map[Event]Event{
	Move: Create,
}
