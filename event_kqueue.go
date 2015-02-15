// +build darwin,kqueue dragonfly freebsd netbsd openbsd

package notify

import (
	"os"
	"syscall"
)

// TODO(pblaszczyk): ensure in runtime notify built-in event values do not
// overlap with platform-defined ones.

// Platform independent event values.
const (
	Create Event = 0x0100 << iota
	Delete
	Write
	Move
	_ // reserved

	// recursive is used to distinguish recursive eventsets from non-recursive ones
	recursive
	// internal is used for watching for new directories within recursive subtrees
	// by non-recrusive watchers
	internal
)

const (
	// NoteDelete is an even reported when the unlink() system call was called
	// on the file referenced by the descriptor.
	NoteDelete = Event(syscall.NOTE_DELETE)
	// NoteWrite is an event reported when a write occurred on the file
	// referenced by the descriptor.
	NoteWrite = Event(syscall.NOTE_WRITE)
	// NoteExtend is an event reported when the file referenced by the
	// descriptor was extended.
	NoteExtend = Event(syscall.NOTE_EXTEND)
	// NoteAttrib is an event reported when the file referenced
	// by the descriptor had its attributes changed.
	NoteAttrib = Event(syscall.NOTE_ATTRIB)
	// NoteLink is an event reported when the link count on the file changed.
	NoteLink = Event(syscall.NOTE_LINK)
	// NoteRename is an event reported when the file referenced
	// by the descriptor was renamed.
	NoteRename = Event(syscall.NOTE_RENAME)
	// NoteRevoke is an event reported when access to the file was revoked via
	// revoke(2) or	the underlying file system was unmounted.
	NoteRevoke = Event(syscall.NOTE_REVOKE)
)

var osestr = map[Event]string{
	NoteDelete: "notify.NoteDelete",
	NoteWrite:  "notify.NoteWrite",
	NoteExtend: "notify.NoteExtend",
	NoteAttrib: "notify.NoteAttrib",
	NoteLink:   "notify.NoteLink",
	NoteRename: "notify.NoteRename",
	NoteRevoke: "notify.NoteRevoke",
}

var ekind = map[Event]Event{
	NoteWrite:  Write,
	NoteRename: Move,
	NoteDelete: Delete,
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

// isdir TODO(pblaszczyk): native implementation for kqueue
func isdir(ei EventInfo) (bool, error) {
	fi, err := os.Stat(ei.Path())
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}
