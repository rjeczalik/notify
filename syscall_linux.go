// +build linux

package notify

import "syscall"

// os_linux.go
var (
	Create    = IN_CREATE
	Delete    = IN_DELETE
	Write     = IN_MODIFY
	Move      = IN_MOVE
	Recursive = HACK_RECURSIVE
)

var All = IN_ALL_EVENTS

// Events
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

	HACK_RECURSIVE = Event(0x00010000)
)

// Event table
var events = [...]string{
	syscall.IN_ACCESS:        "file was accessed for reading",
	syscall.IN_MODIFY:        "file was modified",
	syscall.IN_ATTRIB:        "metadata changed",
	syscall.IN_CLOSE_WRITE:   "file opened for writing was closed",
	syscall.IN_CLOSE_NOWRITE: "file not opened for writing was closed",
	syscall.IN_OPEN:          "file was opened",
	syscall.IN_MOVED_FROM:    "file moved out of watched directory",
	syscall.IN_MOVED_TO:      "file moved into watched directory",
	syscall.IN_CREATE:        "file/directory created",
	syscall.IN_DELETE:        "file/directory deleted",
	syscall.IN_DELETE_SELF:   "file/directory was itself deleted",
	syscall.IN_MOVE_SELF:     "file/directory was itself moved",
	syscall.IN_CLOSE:         "file was closed",
	syscall.IN_MOVE:          "file was moved",
}
