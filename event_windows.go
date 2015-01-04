// +build windows

package notify

import (
	"path/filepath"
	"syscall"
)

const (
	Create Event = 1 << (20 + iota)
	Delete
	Write
	Move
)

// An internal event, in final version won't be exported.
const Recursive = Move << 1

const dirmarker Event = Recursive << 1

// ReadDirectoryChangesW filters.
const (
	FILE_NOTIFY_CHANGE_FILE_NAME   = Event(syscall.FILE_NOTIFY_CHANGE_FILE_NAME)
	FILE_NOTIFY_CHANGE_DIR_NAME    = Event(syscall.FILE_NOTIFY_CHANGE_DIR_NAME)
	FILE_NOTIFY_CHANGE_ATTRIBUTES  = Event(syscall.FILE_NOTIFY_CHANGE_ATTRIBUTES)
	FILE_NOTIFY_CHANGE_SIZE        = Event(syscall.FILE_NOTIFY_CHANGE_SIZE)
	FILE_NOTIFY_CHANGE_LAST_WRITE  = Event(syscall.FILE_NOTIFY_CHANGE_LAST_WRITE)
	FILE_NOTIFY_CHANGE_LAST_ACCESS = Event(syscall.FILE_NOTIFY_CHANGE_LAST_ACCESS)
	FILE_NOTIFY_CHANGE_CREATION    = Event(syscall.FILE_NOTIFY_CHANGE_CREATION)
	FILE_NOTIFY_CHANGE_SECURITY    = Event(syscallFILE_NOTIFY_CHANGE_SECURITY)
)

// according to: http://msdn.microsoft.com/en-us/library/windows/desktop/aa365465(v=vs.85).aspx
// this flag should be declared in: http://golang.org/src/pkg/syscall/ztypes_windows.go
const syscallFILE_NOTIFY_CHANGE_SECURITY = 0x00000100

// ReadDirectoryChangesW actions.
const (
	FILE_ACTION_ADDED            = Event(syscall.FILE_ACTION_ADDED) << 12
	FILE_ACTION_REMOVED          = Event(syscall.FILE_ACTION_REMOVED) << 12
	FILE_ACTION_MODIFIED         = Event(syscall.FILE_ACTION_MODIFIED) << 14
	FILE_ACTION_RENAMED_OLD_NAME = Event(syscall.FILE_ACTION_RENAMED_OLD_NAME) << 15
	FILE_ACTION_RENAMED_NEW_NAME = Event(syscall.FILE_ACTION_RENAMED_NEW_NAME) << 16
)

var osestr = map[Event]string{
	FILE_NOTIFY_CHANGE_FILE_NAME:   "notify.FILE_NOTIFY_CHANGE_FILE_NAME",
	FILE_NOTIFY_CHANGE_DIR_NAME:    "notify.FILE_NOTIFY_CHANGE_DIR_NAME",
	FILE_NOTIFY_CHANGE_ATTRIBUTES:  "notify.FILE_NOTIFY_CHANGE_ATTRIBUTES",
	FILE_NOTIFY_CHANGE_SIZE:        "notify.FILE_NOTIFY_CHANGE_SIZE",
	FILE_NOTIFY_CHANGE_LAST_WRITE:  "notify.FILE_NOTIFY_CHANGE_LAST_WRITE",
	FILE_NOTIFY_CHANGE_LAST_ACCESS: "notify.FILE_NOTIFY_CHANGE_LAST_ACCESS",
	FILE_NOTIFY_CHANGE_CREATION:    "notify.FILE_NOTIFY_CHANGE_CREATION",
	FILE_NOTIFY_CHANGE_SECURITY:    "notify.FILE_NOTIFY_CHANGE_SECURITY",
	FILE_ACTION_ADDED:              "notify.FILE_ACTION_ADDED",
	FILE_ACTION_REMOVED:            "notify.FILE_ACTION_REMOVED",
	FILE_ACTION_MODIFIED:           "notify.FILE_ACTION_MODIFIED",
	FILE_ACTION_RENAMED_OLD_NAME:   "notify.FILE_ACTION_RENAMED_OLD_NAME",
	FILE_ACTION_RENAMED_NEW_NAME:   "notify.FILE_ACTION_RENAMED_NEW_NAME",
}

var ekind = map[Event]Event{}

const (
	ObjectUnknown uint8 = iota
	ObjectFile
	ObjectDirectory
)

// TODO(ppknap) : doc.
type event struct {
	pathw   []uint16
	name    string
	objtype uint8
	action  uint32
	filter  uint32
	e       Event
}

func (e *event) Event() Event     { return e.e }
func (e *event) Path() string     { return filepath.Join(syscall.UTF16ToString(e.pathw), e.name) }
func (e *event) Sys() interface{} { return e.objtype }
