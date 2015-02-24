// Copyright (c) 2014-2015 The Notify Authors. All rights reserved.
// Use of this source code is governed by the MIT license that can be
// found in the LICENSE file.

// +build windows

package notify

import (
	"os"
	"path/filepath"
	"syscall"
)

// Platform independent event values.
const (
	osSpecificCreate Event = 1 << (20 + iota)
	osSpecificRemove
	osSpecificWrite
	osSpecificRename
	// recursive is used to distinguish recursive eventsets from non-recursive ones
	recursive
	// internal is used for watching for new directories within recursive subtrees
	// by non-recrusive watchers
	internal
	// dirmarker TODO(pknap)
	dirmarker
)

// ReadDirectoryChangesW filters.
const (
	FileNotifyChangeFileName   = actFRW | Event(syscall.FILE_NOTIFY_CHANGE_FILE_NAME)
	FileNotifyChangeDirName    = actFRW | Event(syscall.FILE_NOTIFY_CHANGE_DIR_NAME)
	FileNotifyChangeAttributes = actFMV | Event(syscall.FILE_NOTIFY_CHANGE_ATTRIBUTES)
	FileNotifyChangeSize       = actFMV | Event(syscall.FILE_NOTIFY_CHANGE_SIZE)
	FileNotifyChangeLastWrite  = actFMV | Event(syscall.FILE_NOTIFY_CHANGE_LAST_WRITE)
	FileNotifyChangeLastAccess = actFMV | Event(syscall.FILE_NOTIFY_CHANGE_LAST_ACCESS)
	FileNotifyChangeCreation   = actFMV | Event(syscall.FILE_NOTIFY_CHANGE_CREATION)
	FileNotifyChangeSecurity   = actFMV | Event(syscallFileNotifyChangeSecurity)
)

// logical sum of all syscall.FILE_NOTIFY_CHANGE_* and FileAction* events.
const fileNotifyChangeAndActionAll = Event(0x17f) | actAll

// according to: http://msdn.microsoft.com/en-us/library/windows/desktop/aa365465(v=vs.85).aspx
// this flag should be declared in: http://golang.org/src/pkg/syscall/ztypes_windows.go
const syscallFileNotifyChangeSecurity = 0x00000100

// ReadDirectoryChangesW actions.
const (
	FileActionAdded          = Event(syscall.FILE_ACTION_ADDED) << 12
	FileActionRemoved        = Event(syscall.FILE_ACTION_REMOVED) << 12
	FileActionModified       = Event(syscall.FILE_ACTION_MODIFIED) << 14
	FileActionRenamedOldName = Event(syscall.FILE_ACTION_RENAMED_OLD_NAME) << 15
	FileActionRenamedNewName = Event(syscall.FILE_ACTION_RENAMED_NEW_NAME) << 16
)

var osestr = map[Event]string{
	FileNotifyChangeFileName:   "notify.FileNotifyChangeFileName",
	FileNotifyChangeDirName:    "notify.FileNotifyChangeDirName",
	FileNotifyChangeAttributes: "notify.FileNotifyChangeAttributes",
	FileNotifyChangeSize:       "notify.FileNotifyChangeSize",
	FileNotifyChangeLastWrite:  "notify.FileNotifyChangeLastWrite",
	FileNotifyChangeLastAccess: "notify.FileNotifyChangeLastAccess",
	FileNotifyChangeCreation:   "notify.FileNotifyChangeCreation",
	FileNotifyChangeSecurity:   "notify.FileNotifyChangeSecurity",

	FileActionAdded:          "notify.FileActionAdded",
	FileActionRemoved:        "notify.FileActionRemoved",
	FileActionModified:       "notify.FileActionModified",
	FileActionRenamedOldName: "notify.FileActionRenamedOldName",
	FileActionRenamedNewName: "notify.FileActionRenamedNewName",
}

const (
	actFRW = FileActionAdded | FileActionRemoved | FileActionRenamedOldName | FileActionRenamedNewName
	actFMV = FileActionModified
	actAll = actFRW | actFMV
)

var ekind = map[Event]Event{}

const (
	fTypeUnknown uint8 = iota
	fTypeFile
	fTypeDirectory
)

// TODO(ppknap) : doc.
type event struct {
	pathw  []uint16
	name   string
	ftype  uint8
	action uint32
	filter uint32
	e      Event
}

func (e *event) Event() Event     { return e.e }
func (e *event) Path() string     { return filepath.Join(syscall.UTF16ToString(e.pathw), e.name) }
func (e *event) Sys() interface{} { return nil }

func (e *event) isDir() (bool, error) {
	if e.ftype != fTypeUnknown {
		return e.ftype == fTypeDirectory, nil
	}
	fi, err := os.Stat(e.Path())
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}
