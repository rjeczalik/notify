// +build linux

package notify

import "syscall"

const (
	Create Event = 0x100000 << iota
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

// Inotify events.
const (
	IN_ACCESS        = Event(syscall.IN_ACCESS)
	IN_MODIFY        = Event(syscall.IN_MODIFY)
	IN_ATTRIB        = Event(syscall.IN_ATTRIB)
	IN_CLOSE_WRITE   = Event(syscall.IN_CLOSE_WRITE)
	IN_CLOSE_NOWRITE = Event(syscall.IN_CLOSE_NOWRITE)
	IN_OPEN          = Event(syscall.IN_OPEN)
	IN_MOVED_FROM    = Event(syscall.IN_MOVED_FROM)
	IN_MOVED_TO      = Event(syscall.IN_MOVED_TO)
	IN_CREATE        = Event(syscall.IN_CREATE)
	IN_DELETE        = Event(syscall.IN_DELETE)
	IN_DELETE_SELF   = Event(syscall.IN_DELETE_SELF)
	IN_MOVE_SELF     = Event(syscall.IN_MOVE_SELF)
)

var osestr = map[Event]string{
	IN_ACCESS:        "notify.IN_ACCESS",
	IN_MODIFY:        "notify.IN_MODIFY",
	IN_ATTRIB:        "notify.IN_ATTRIB",
	IN_CLOSE_WRITE:   "notify.IN_CLOSE_WRITE",
	IN_CLOSE_NOWRITE: "notify.IN_CLOSE_NOWRITE",
	IN_OPEN:          "notify.IN_OPEN",
	IN_MOVED_FROM:    "notify.IN_MOVED_FROM",
	IN_MOVED_TO:      "notify.IN_MOVED_TO",
	IN_CREATE:        "notify.IN_CREATE",
	IN_DELETE:        "notify.IN_DELETE",
	IN_DELETE_SELF:   "notify.IN_DELETE_SELF",
	IN_MOVE_SELF:     "notify.IN_MOVE_SELF",
}

// Inotify behavior events are not **currently** supported by notify package.
const (
	iN_DONT_FOLLOW = Event(syscall.IN_DONT_FOLLOW)
	iN_EXCL_UNLINK = Event(syscall.IN_EXCL_UNLINK)
	iN_MASK_ADD    = Event(syscall.IN_MASK_ADD)
	iN_ONESHOT     = Event(syscall.IN_ONESHOT)
	iN_ONLYDIR     = Event(syscall.IN_ONLYDIR)
)

var ekind = map[Event]Event{
	syscall.IN_MOVED_FROM:  Create,
	syscall.IN_MOVED_TO:    Delete,
	syscall.IN_DELETE_SELF: Delete,
}

// TODO(ppknap) : doc.
type event struct {
	sys  syscall.InotifyEvent
	impl watched
}

func (e *event) Event() Event     { return decode(e.impl.mask, e.sys.Mask) }
func (e *event) Path() string     { return e.impl.pathname }
func (e *event) Sys() interface{} { return e.sys }
