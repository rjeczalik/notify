// +build windows
// +build !fsnotify

package notify

import "syscall"

const (
	Create Event = 0x1000 << iota
	Delete
	Write
	Move

	Recursive // An internal event, in final version won't be exported.
)

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
	FILE_ACTION_ADDED            = Event(syscall.FILE_ACTION_ADDED)
	FILE_ACTION_REMOVED          = Event(syscall.FILE_ACTION_REMOVED)
	FILE_ACTION_MODIFIED         = Event(syscall.FILE_ACTION_MODIFIED)
	FILE_ACTION_RENAMED_OLD_NAME = Event(syscall.FILE_ACTION_RENAMED_OLD_NAME)
	FILE_ACTION_RENAMED_NEW_NAME = Event(syscall.FILE_ACTION_RENAMED_NEW_NAME)
)

const dirmarker Event = Recursive << 1

// FIXME(someone) - actions returned by ReadDirectoryChangesW are not defined as
// bitmasks. This results in invalid return values created by String() method.
var osestr = map[Event]string{
// FILE_ACTION_ADDED:            "notify.FILE_ACTION_ADDED",
// FILE_ACTION_REMOVED:          "notify.FILE_ACTION_REMOVED",
// FILE_ACTION_MODIFIED:         "notify.FILE_ACTION_MODIFIED",
// FILE_ACTION_RENAMED_OLD_NAME: "notify.FILE_ACTION_RENAMED_OLD_NAME",
// FILE_ACTION_RENAMED_NEW_NAME: "notify.FILE_ACTION_RENAMED_NEW_NAME",
}

var ekind = map[Event]Event{}
