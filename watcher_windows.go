// +build windows
// +build !fsnotify

package notify

import (
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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

// watched is made in order to check whether an action comes from a directory or
// file. This approach requires two file handlers per single monitored folder. The
// second grip handles actions which include creating or deleting a directory. If
// these processes are not monitored, only the first grip is created.
type watched struct {
	pathw  []uint16
	digrip [2]*grip
}

// makeWatched creates a new watched instance. It splits a filter variable into
// two parts. The first part is responsible for watching all events which can be
// created for a file in watched directory structure and the second one watches
// only directory Create/Delete actions. If all operations succeed, the Create
// message is sent to I/O completion port queue for further processing.
func makeWatched(cph syscall.Handle, path string, filter uint32) (wd watched, err error) {
	if wd.pathw, err = syscall.UTF16FromString(path); err != nil {
		return
	}
	var fdfilter uint32 = filter &^ uint32(FILE_NOTIFY_CHANGE_DIR_NAME)
	if fdfilter != 0 {
		if wd.digrip[0], err = newGrip(cph, wd.pathw, fdfilter); err != nil {
			return
		}
	}
	var dfilter uint32 = filter & uint32(FILE_NOTIFY_CHANGE_DIR_NAME|All)
	if dfilter^uint32(All) != 0 {
		if wd.digrip[1], err = newGrip(
			cph, wd.pathw, dfilter|uint32(dirmarker)); err != nil {
			wd.closeHandle()
			return
		}
	}
	return wd, wd.iocpMsg(cph, 1)
}

// iocpMsg posts an I/O completion packet to completion port pointed by 'cph'
// handle. Message will be passed as completion key and shall not be considered
// as a valid state until the `filter` member from grip variable is checked.
func (wd *watched) iocpMsg(cph syscall.Handle, msg uint32) (err error) {
	for _, g := range wd.digrip {
		if g != nil {
			overlapped := (*syscall.Overlapped)(unsafe.Pointer(g.ovlapped))
			if e := syscall.PostQueuedCompletionStatus(
				cph, 0, msg, overlapped); e != nil && err == nil {
				err = e
			}
		}
	}
	return
}

// closeHandle closes handles that are stored in digrip array. Function always
// tries to close all of the handlers before it exits, even when there are errors
// returned from the operating system kernel.
func (wd *watched) closeHandle() (err error) {
	for _, g := range wd.digrip {
		if g != nil {
			if e := syscall.CloseHandle(g.handle); e != nil && err == nil {
				err = e
			}
		}
	}
	return
}

// watcher implements Watcher interface. It stores a set of watched directories.
// All operations which remove watched objects from map `m` must be performed in
// loop goroutine since these structures are used internally by operating system.
type watcher struct {
	sync.RWMutex
	m   map[string]watched
	cph syscall.Handle
	c   chan<- EventInfo
}

// NewWatcher creates new non-recursive watcher backed by ReadDirectoryChangesW.
func newWatcher() *watcher {
	return &watcher{
		m:   make(map[string]watched),
		cph: syscall.InvalidHandle,
	}
}

// Watch inserts a directory to the group of watched folders. If watched folder
// already exists, function tries to rewatch it with new filters. Moreover,
// Watch starts the main event loop goroutine when called for the first time.
func (w *watcher) Watch(path string, e Event) (err error) {
	w.RLock()
	wd, ok := w.m[path]
	w.RUnlock()
	if !ok {
		if err = w.lazyinit(); err != nil {
			return
		}
		w.Lock()
		if _, ok = w.m[path]; ok {
			// TODO: Rewatch
			w.Unlock()
			return
		}
		if wd, err = makeWatched(w.cph, path, uint32(e)); err != nil {
			w.Unlock()
			return
		}
		w.m[path] = wd
		w.Unlock()
	}
	return nil
}

// lazyinit creates IO completion port and sets the finalizer in order to close
// the port's handler at the end of program execution. This method uses Double-
// Checked Locking optimization.
func (w *watcher) lazyinit() (err error) {
	invalid := uintptr(syscall.InvalidHandle)
	if atomic.LoadUintptr((*uintptr)(&w.cph)) == invalid {
		w.Lock()
		if atomic.LoadUintptr((*uintptr)(&w.cph)) == invalid {
			cph := syscall.InvalidHandle
			if cph, err = syscall.CreateIoCompletionPort(
				cph, 0, 0, 0); err != nil {
				w.Unlock()
				return
			}
			w.cph = cph
			runtime.SetFinalizer(&w.cph, func(handle *syscall.Handle) {
				syscall.CloseHandle(*handle)
			})
			go w.loop()
		}
		w.Unlock()
	}
	return
}

// TODO(ppknap) : doc
func (w *watcher) loop() {
	var n, key uint32
	var overlapped *syscall.Overlapped
	for {
		if err := syscall.GetQueuedCompletionStatus(
			w.cph, &n, &key, &overlapped, syscall.INFINITE); err != nil {
			// TODO(ppknap) : Error handling
		}
		var overEx = (*overlappedEx)(unsafe.Pointer(overlapped))
		es := []*event{}
		if n != 0 {
			var currOffset uint32
			for {
				raw := (*syscall.FileNotifyInformation)(
					unsafe.Pointer(&overEx.parent.buffer[currOffset]))
				buf := (*[syscall.MAX_PATH]uint16)(unsafe.Pointer(&raw.FileName))
				name := syscall.UTF16ToString(buf[:raw.FileNameLength/2])
				es = append(es, &event{
					pathw:  overEx.parent.pathw,
					filter: overEx.parent.filter,
					action: raw.Action,
					name:   name,
				})
				if raw.NextEntryOffset == 0 {
					break
				}
				if currOffset += raw.NextEntryOffset; currOffset >= n {
					break
				}
			}
		}
		if err := overEx.parent.readDirChanges(); err != nil {
			// TODO(ppknap) : Error handling
		}
		w.send(es)
	}
}

// TODO(ppknap) : doc
func (w *watcher) send(es []*event) {
	for _, e := range es {
		if e.e = decode(e.filter, e.action); e.e == 0 {
			continue
		}
		switch Event(e.action) {
		case FILE_ACTION_ADDED, FILE_ACTION_REMOVED:
			e.isdir = e.filter&uint32(dirmarker) != 0
		default:
			// TODO(ppknap) : or not TODO?
		}
		w.c <- e
	}
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
