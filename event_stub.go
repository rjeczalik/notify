// +build !linux

package notify

var (
	Create = EV_STUB_CREATE
	Delete = EV_STUB_DELETE
	Write  = EV_STUB_WRITE
	Move   = EV_STUB_MOVE
)

// All TODO
var All = EV_STUB_ALL

// Events TODO
const (
	EV_STUB_CREATE = Event(0x00000001)
	EV_STUB_DELETE = Event(0x00000002)
	EV_STUB_WRITE  = Event(0x00000004)
	EV_STUB_MOVE   = Event(0x00000008)
	EV_STUB_ALL    = Event(0x0000000f)
)

var estr = map[Event]string{
	Create: "create",
	Delete: "delete",
	Write:  "write",
	Move:   "move",
}

var ekind = map[Event]Event{
	Move: Create, // TODO(rjeczalik): Move (fsnotify.Rename) does not work, workaround?
}
