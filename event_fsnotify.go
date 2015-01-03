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

var osestr = map[Event]string{
	FSN_CHMOD: "notify.Chmod",
}

var ekind = map[Event]Event{
	FSN_RENAME: Create,
}

type event struct {
	name string
	ev   Event
}

func (e event) Event() Event     { return e.ev }
func (e event) Path() string     { return e.name }
func (e event) Sys() interface{} { return nil } // no-one cares about fsnotify.Event

func newEvent(ev fsnotifyv1.Event) EventInfo {
	e := event{
		name: ev.Name,
		ev:   Event(ev.Op),
	}
	return e
}
