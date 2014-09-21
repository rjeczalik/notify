// +build fsnotify

package notify

import fsnotifyv1 "gopkg.in/fsnotify.v1"

// Generic notify events.
const (
	Create = FSN_CREATE
	Delete = FSN_REMOVE
	Write  = FSN_WRITE
	Move   = FSN_RENAME
)

// Fsnotify events.
const (
	FSN_CREATE = Event(fsnotifyv1.Create)
	FSN_REMOVE = Event(fsnotifyv1.Remove)
	FSN_WRITE  = Event(fsnotifyv1.Write)
	FSN_RENAME = Event(fsnotifyv1.Rename)
	FSN_CHMOD  = Event(fsnotifyv1.Chmod)
	FSN_ALL    = FSN_CREATE | FSN_REMOVE | FSN_WRITE | FSN_RENAME | FSN_CHMOD
)

var estr = map[Event]string{
	FSN_CREATE: "create",
	FSN_REMOVE: "delete",
	FSN_WRITE:  "write",
	FSN_RENAME: "move",
	FSN_CHMOD:  "chmod",
}

var ekind = map[Event]Event{
	FSN_RENAME: Create,
}
