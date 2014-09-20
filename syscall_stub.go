// +build !linux

package notify

// os_stub.go
var (
	Create    = EV_STUB_CREATE
	Delete    = EV_STUB_DELETE
	Write     = EV_STUB_WRITE
	Move      = EV_STUB_MOVE
	Recursive = EV_STUB_RECURSIVE
)

var All = EV_STUB_ALL

// Events
const (
	EV_STUB_CREATE = Event(0x00000001)
	EV_STUB_DELETE = Event(0x00000002)
	EV_STUB_WRITE  = Event(0x00000004)
	EV_STUB_MOVE   = Event(0x00000008)
	EV_STUB_ALL    = Event(0x0000000f)

	EV_STUB_RECURSIVE = Event(0x00000010)
)

// Event table
var events = [...]string{
	1:  "create",
	2:  "delete",
	4:  "write",
	8:  "move",
	16: "recursive",
}
