// +build linux,!fsnotify

package notify

import "syscall"

const (
	Create Event = 0x100000 << iota
	Delete
	Write
	Move
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
	IN_CLOSE         = Event(syscall.IN_CLOSE)
	IN_MOVE          = Event(syscall.IN_MOVE)
	IN_ALL_EVENTS    = Event(syscall.IN_ALL_EVENTS)
)

// Inotify behavior events.
const (
	//IN_DONT_FOLLOW = Event(syscall.IN_DONT_FOLLOW) // TODO add support
	//IN_EXCL_UNLINK = Event(syscall.IN_EXCL_UNLINK) // TODO add support
	IN_MASK_ADD = Event(syscall.IN_MASK_ADD)
	//IN_ONESHOT     = Event(syscall.IN_ONESHOT)  // TODO add support
	//IN_ONLYDIR     = Event(syscall.IN_ONLYDIR)  // TODO add support
)

const invalid = ^(All | IN_ALL_EVENTS)

var estr = map[Event]string{
	Create:           "notify.Create",
	Delete:           "notify.Delete",
	Write:            "notify.Write",
	Move:             "notify.Move",
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
	IN_CLOSE:         "notify.IN_CLOSE",
	IN_MOVE:          "notify.IN_MOVE",
}

var ekind = map[Event]Event{
	syscall.IN_MOVED_FROM:  Create,
	syscall.IN_MOVED_TO:    Delete,
	syscall.IN_DELETE_SELF: Delete,
}
