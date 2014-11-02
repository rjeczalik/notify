// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd
// +build !fsnotify

package notify

const (
	Create    = EV_STUB_CREATE
	Delete    = EV_STUB_DELETE
	Write     = EV_STUB_WRITE
	Move      = EV_STUB_MOVE
	Recursive = EV_STUB_RECURSIVE // internal event
)

// Events TODO
const (
	EV_STUB_CREATE    = Event(0x01)
	EV_STUB_DELETE    = Event(0x02)
	EV_STUB_WRITE     = Event(0x04)
	EV_STUB_MOVE      = Event(0x08)
	EV_STUB_RECURSIVE = Event(0x0f)
	EV_STUB_ALL       = Event(0x1f)
)

var osestr = map[Event]string{}

var ekind = map[Event]Event{
	Move: Create, // TODO(rjeczalik): Move (fsnotify.Rename) does not work, workaround?
}
