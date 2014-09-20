// +build !linux

package notify

// os_stub.go
const (
	Create = EV_STUB_CREATE
	Delete = EV_STUB_DELETE
	Write  = EV_STUB_WRITE
	Move   = EV_STUB_MOVE
)

const All = Event(0x0000000f)

// Events
const (
	EV_STUB_CREATE = Event(0x00000001)
	EV_STUB_DELETE = Event(0x00000002)
	EV_STUB_WRITE  = Event(0x00000004)
	EV_STUB_MOVE   = Event(0x00000008)
	EV_STUB_ALL    = Event(0x0000000f)
)

// Event names
var evnames = map[int]string{
	1: "create",
	2: "delete",
	4: "write",
	8: "move",
}

// Event kinds
var evkinds = map[int]Event{
	1: Create,
	2: Delete,
	4: Write,
	8: Move,
}
