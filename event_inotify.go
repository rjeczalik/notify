// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build linux

package notify

import (
	"os"
	"syscall"
)

// Platform independent event values.
const (
	osSpecificCreate Event = 0x100000 << iota
	osSpecificRemove
	osSpecificWrite
	osSpecificRename
	// internal
	// recursive is used to distinguish recursive eventsets from non-recursive ones
	recursive
	// internal is used for watching for new directories within recursive subtrees
	// by non-recrusive watchers
	internal
)

// Inotify specific event values.
const (
	InAccess       = Event(syscall.IN_ACCESS)
	InModify       = Event(syscall.IN_MODIFY)
	InAttrib       = Event(syscall.IN_ATTRIB)
	InCloseWrite   = Event(syscall.IN_CLOSE_WRITE)
	InCloseNowrite = Event(syscall.IN_CLOSE_NOWRITE)
	InOpen         = Event(syscall.IN_OPEN)
	InMovedFrom    = Event(syscall.IN_MOVED_FROM)
	InMovedTo      = Event(syscall.IN_MOVED_TO)
	InCreate       = Event(syscall.IN_CREATE)
	InDelete       = Event(syscall.IN_DELETE)
	InDeleteSelf   = Event(syscall.IN_DELETE_SELF)
	InMoveSelf     = Event(syscall.IN_MOVE_SELF)
)

var osestr = map[Event]string{
	InAccess:       "notify.InAccess",
	InModify:       "notify.InModify",
	InAttrib:       "notify.InAttrib",
	InCloseWrite:   "notify.InCloseWrite",
	InCloseNowrite: "notify.InCloseNowrite",
	InOpen:         "notify.InOpen",
	InMovedFrom:    "notify.InMovedFrom",
	InMovedTo:      "notify.InMovedTo",
	InCreate:       "notify.InCreate",
	InDelete:       "notify.InDelete",
	InDeleteSelf:   "notify.InDeleteSelf",
	InMoveSelf:     "notify.InMoveSelf",
}

// Inotify behavior events are not **currently** supported by notify package.
const (
	inDontFollow = Event(syscall.IN_DONT_FOLLOW)
	inExclUnlink = Event(syscall.IN_EXCL_UNLINK)
	inMaskAdd    = Event(syscall.IN_MASK_ADD)
	inOneshot    = Event(syscall.IN_ONESHOT)
	inOnlydir    = Event(syscall.IN_ONLYDIR)
)

var ekind = map[Event]Event{
	InMovedFrom:  Create,
	InMovedTo:    Remove,
	InDeleteSelf: Remove,
}

// TODO(ppknap) : doc.
type event struct {
	sys   syscall.InotifyEvent
	path  string
	event Event
}

func (e *event) Event() Event     { return e.event }
func (e *event) Path() string     { return e.path }
func (e *event) Sys() interface{} { return &e.sys }

func isdir(ei EventInfo) (bool, error) {
	if _, ok := ei.Sys().(syscall.InotifyEvent); ok {
		return ei.Sys().(syscall.InotifyEvent).Mask&syscall.IN_ISDIR != 0, nil
	}
	fi, err := os.Stat(ei.Path())
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}
