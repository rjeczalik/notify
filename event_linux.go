// +build linux

package notify

import "syscall"

var (
	Create = IN_CREATE
	Delete = IN_DELETE
	Write  = IN_MODIFY
	Move   = IN_MOVE
)

// All TODO
var All = IN_ALL_EVENTS

// Events TODO
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

var estr = map[Event]string{
	Create:           "create",
	Delete:           "delete",
	Write:            "write",
	Move:             "move",
	IN_ACCESS:        "syscall.IN_ACCESS",
	IN_MODIFY:        "syscall.IN_MODIFY",
	IN_ATTRIB:        "syscall.IN_ATTRIB",
	IN_CLOSE_WRITE:   "syscall.IN_CLOSE_WRITE",
	IN_CLOSE_NOWRITE: "syscall.IN_CLOSE_NO_WRITE",
	IN_OPEN:          "syscall.IN_OPEN",
	IN_MOVED_FROM:    "syscall.IN_MOVED_FROM",
	IN_MOVED_TO:      "syscall.IN_MOVED_TO",
	IN_CREATE:        "syscall.IN_CREATE",
	IN_DELETE:        "syscall.IN_DELETE",
	IN_DELETE_SELF:   "syscall.IN_DELETE_SELF",
	IN_MOVE_SELF:     "syscall.IN_MOVE_SELF",
	IN_CLOSE:         "syscall.IN_CLOSE",
	IN_MOVE:          "syscall.IN_MOVE",
}
