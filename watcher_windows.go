// +build windows
// +build !fsnotify

package notify

import (
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"
)

// readBufferSize defines the size of an array in which read statuses are stored.
// The buffer have to be DWORD-aligned and, if notify is used in monitoring a
// directory over the network, the buffer size must not be greater than 64KB.
// Each of watched directories uses its own buffer for storing events.
const readBufferSize = 4096

// grip represents a single watched directory. It stores the data required by
// ReadDirectoryChangesW function. Only the filter mamber value may by modified
// by watcher implementation. Rest of the members have to remain constat since
// they are used by Windows completion routine. This indicates that grip can be
// removed only when all operations on the file handle are finished.
type grip struct {
	handle   syscall.Handle
	filter   uint32
	pathw    []uint16
	buffer   [readBufferSize]byte
	ovlapped *overlappedEx
}

// overlappedEx stores information used in asynchronous input and output.
// Additionally, overlappedEx contains a pointer to 'grip' item which is used in
// order to gather the structure in which the overlappedEx object was created.
type overlappedEx struct {
	syscall.Overlapped
	parent *grip
}

// newGrip creates a new file hande that can be used in overlapped operations.
// Then, the handle is associated with I/O completion port 'cph' and its value
// is stored in newly created 'grip' object.
func newGrip(cph syscall.Handle, pathw []uint16, filter uint32) (*grip, error) {
	g := &grip{
		handle:   syscall.InvalidHandle,
		pathw:    pathw,
		filter:   filter,
		ovlapped: &overlappedEx{},
	}
	var err error
	if g.handle, err = syscall.CreateFile(
		&g.pathw[0],
		syscall.FILE_LIST_DIRECTORY,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OVERLAPPED,
		0,
	); err != nil {
		return nil, err
	}
	if _, err = syscall.CreateIoCompletionPort(g.handle, cph, 0, 0); err != nil {
		syscall.CloseHandle(g.handle)
		return nil, err
	}
	g.ovlapped.parent = g
	return g, nil
}

// readDirChanges tells the system to store file change information in grip's
// buffer. Directory changes that occur between calls to this function are added
// to the buffer and then, returned with the next call.
func (g *grip) readDirChanges() error {
	return syscall.ReadDirectoryChanges(
		g.handle,
		&g.buffer[0],
		uint32(unsafe.Sizeof(g.buffer)),
		false,
		encode(g.filter),
		nil,
		(*syscall.Overlapped)(unsafe.Pointer(g.ovlapped)),
		0,
	)
}

// encode transforms a generic filter, which contains platform independent and
// implementation specific bit fields, to value that can be used as NotifyFilter
// parameter in ReadDirectoryChangesW function.
func encode(filter uint32) uint32 {
	e := Event(filter)
	if e&dirmarker != 0 {
		return uint32(FILE_NOTIFY_CHANGE_DIR_NAME)
	}
	if e&Create != 0 {
		e = (e ^ Create) | FILE_NOTIFY_CHANGE_FILE_NAME
	}
	if e&Delete != 0 {
		e = (e ^ Delete) | FILE_NOTIFY_CHANGE_FILE_NAME
	}
	if e&Write != 0 {
		e = (e ^ Write) |
			FILE_NOTIFY_CHANGE_ATTRIBUTES | FILE_NOTIFY_CHANGE_SIZE |
			FILE_NOTIFY_CHANGE_CREATION | FILE_NOTIFY_CHANGE_SECURITY
	}
	if e&Move != 0 {
		e = (e ^ Move) | FILE_NOTIFY_CHANGE_FILE_NAME
	}
	return uint32(e &^ FILE_NOTIFY_CHANGE_DIR_NAME)
}

type watcher struct {
	sync.RWMutex
	cph syscall.Handle
	c   chan<- EventInfo
}

// NewWatcher creates new non-recursive watcher backed by ReadDirectoryChangesW.
func newWatcher() *watcher {
	return &watcher{
		cph: syscall.InvalidHandle,
	}
}

// Watch implements notify.Watcher interface.
func (w *watcher) Watch(path string, e Event) (err error) {
	return nil
}

// Unwatch implements notify.Watcher interface.
func (w *watcher) Unwatch(p string) error {
	return nil
}

// Dispatch implements notify.Watcher interface.
func (w *watcher) Dispatch(c chan<- EventInfo, stop <-chan struct{}) {
	w.c = c
}

// decode creates a notify event from both non-raw filter and action which was
// redurned from completion routine. Function may return Event(0) in case when
// filter was replaced by a new value which does not contain fields that are
// valid with passed action.
func decode(filter, action uint32) Event {
	switch action {
	case syscall.FILE_ACTION_ADDED:
		return addrm(filter, Create, FILE_ACTION_ADDED)
	case syscall.FILE_ACTION_REMOVED:
		return addrm(filter, Delete, FILE_ACTION_REMOVED)
	case syscall.FILE_ACTION_MODIFIED:
		return Write
	case syscall.FILE_ACTION_RENAMED_OLD_NAME, syscall.FILE_ACTION_RENAMED_NEW_NAME:
		return Move
	}
	panic("notify: cannot decode internal mask")
}

// addrm decides whether the Windows action or the system-independent event
// should be returned. Since the grip`s filter may be atomically changed during
// watcher lifetime, it is possible that neither Windows nor notify masks are
// present in variable memory.
func addrm(filter uint32, e, syse Event) Event {
	switch {
	case filter&uint32(FILE_NOTIFY_CHANGE_FILE_NAME|FILE_NOTIFY_CHANGE_DIR_NAME) != 0:
		return syse
	case filter&uint32(e) != 0:
		return e
	default:
		return Event(0)
	}
}

// TODO(pknap) : add system-dependent event decoder for FILE_ACTION_MODIFIED,
// FILE_ACTION_RENAMED_OLD_NAME, and FILE_ACTION_RENAMED_NEW_NAME actions.

// TODO(ppknap) : doc.
type event struct {
	pathw  []uint16
	name   string
	isdir  bool
	action uint32
	filter uint32
	e      Event
}

func (e *event) Event() Event     { return e.e }
func (e *event) IsDir() bool      { return e.isdir }
func (e *event) Name() string     { return filepath.Join(syscall.UTF16ToString(e.pathw), e.name) }
func (e *event) Sys() interface{} { return nil }
