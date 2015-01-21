// +build darwin,kqueue dragonfly freebsd netbsd openbsd

package notify

import "syscall"

// TODO(pblaszczyk): ensure in runtime notify built-in event values do not
// overlap with platform-defined ones.

const (
	Create Event = 0x0100 << iota
	Delete
	Write
	Move
	Error

	// Internal events TOOD(rjeczalik): unexport
	//
	// Recursive is used to distinguish recursive eventsets from non-recursive ones.
	recursive

	// Inactive is used to bookkeep child watchpoints in the parent ones, which
	// have been set with an actual filesystem watch. This is to allow for
	// optimizing recursive watchpoint count - a single path can be watched
	// by at most 1 recursive filesystem watch.
	inactive
)

const (
	// NOTE_DELETE is an even reported when the unlink() system call was called
	// on the file referenced by the descriptor.
	NOTE_DELETE = Event(syscall.NOTE_DELETE)
	// NOTE_WRITE is an event reported when a write occurred on the file
	// referenced by the descriptor.
	NOTE_WRITE = Event(syscall.NOTE_WRITE)
	// NOTE_EXTEND is an event reported when the file referenced by the
	// descriptor was extended.
	NOTE_EXTEND = Event(syscall.NOTE_EXTEND)
	// NOTE_ATTRIB is an event reported when the file referenced
	// by the descriptor had its attributes changed.
	NOTE_ATTRIB = Event(syscall.NOTE_ATTRIB)
	// NOTE_LINK is an event reported when the link count on the file changed.
	NOTE_LINK = Event(syscall.NOTE_LINK)
	// NOTE_RENAME is an event reported when the file referenced
	// by the descriptor was renamed.
	NOTE_RENAME = Event(syscall.NOTE_RENAME)
	// NOTE_REVOKE is an event reported when access to the file was revoked via
	// revoke(2) or	the underlying file system was unmounted.
	NOTE_REVOKE = Event(syscall.NOTE_REVOKE)
)

var osestr = map[Event]string{
	NOTE_DELETE: "notify.NOTE_DELETE",
	NOTE_WRITE:  "notify.NOTE_WRITE",
	NOTE_EXTEND: "notify.NOTE_EXTEND",
	NOTE_ATTRIB: "notify.NOTE_ATTRIB",
	NOTE_LINK:   "notify.NOTE_LINK",
	NOTE_RENAME: "notify.NOTE_RENAME",
	NOTE_REVOKE: "notify.NOTE_REVOKE",
}

var ekind = map[Event]Event{
	syscall.NOTE_WRITE:  Write,
	syscall.NOTE_RENAME: Move,
	syscall.NOTE_DELETE: Delete,
}

// event is a struct storing reported event's data.
type event struct {
	// p is a absolute path to file for which event is reported.
	p string
	// e specifies type of a reported event.
	e Event
	// kq specifies single event.
	kq KqEvent
}

// Event returns type of a reported event.
func (e *event) Event() Event { return e.e }

// Path returns path to file/directory for which event is reported.
func (e *event) Path() string { return e.p }

// Sys returns platform specific object describing reported event.
func (e *event) Sys() interface{} { return e.kq }
