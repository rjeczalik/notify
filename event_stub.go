// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd
// +build !fsnotify

package notify

const (
	Create = EV_STUB_CREATE
	Delete = EV_STUB_DELETE
	Write  = EV_STUB_WRITE
	Move   = EV_STUB_MOVE
)

// Events TODO
const (
	EV_STUB_CREATE = Event(0x00000001)
	EV_STUB_DELETE = Event(0x00000002)
	EV_STUB_WRITE  = Event(0x00000004)
	EV_STUB_MOVE   = Event(0x00000008)
	EV_STUB_ALL    = Event(0x0000000f)
)

var osestr = map[Event]string{}

var ekind = map[Event]Event{
	Move: Create, // TODO(rjeczalik): Move (fsnotify.Rename) does not work, workaround?
}
