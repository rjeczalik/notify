// +build windows

package notify

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// readBufferSize defines the size of an array in which read statuses are stored.
// The buffer have to be DWORD-aligned and, if notify is used in monitoring a
// directory over the network, its size must not be greater than 64KB. Each of
// watched directories uses its own buffer for storing events.
const readBufferSize = 4096

// Since all operations which go through the Windows completion routine are done
// asynchronously, filter may set one of the constants below. They were defined
// in order to distinguish whether current folder should be re-registered in
// ReadDirectoryChangesW function or some control operations need to be executed.
const (
	stateRewatch uint32 = 1 << (28 + iota)
	stateUnwatch
	stateCPClose
)

// Filter used in current implementation was split into four segments:
//  - bits  0-11 store ReadDirectoryChangesW filters,
//  - bits 12-19 store File notify actions,
//  - bits 20-27 store notify specific events and flags,
//  - bits 28-31 store states which are used in loop's FSM.
// Constants below are used as masks to retrieve only specific filter parts.
const (
	onlyNotifyChanges uint32 = 0x00000FFF
	onlyNGlobalEvents uint32 = 0x0FF00000
	onlyMachineStates uint32 = 0xF0000000
)

// grip represents a single watched directory. It stores the data required by
// ReadDirectoryChangesW function. Only the filter, recursive, and handle members
// may by modified by watcher implementation. Rest of the them have to remain
// constant since they are used by Windows completion routine. This indicates that
// grip can be removed only when all operations on the file handle are finished.
type grip struct {
	handle    syscall.Handle
	filter    uint32
	recursive bool
	pathw     []uint16
	buffer    [readBufferSize]byte
	parent    *watched
	ovlapped  *overlappedEx
}

// overlappedEx stores information used in asynchronous input and output.
// Additionally, overlappedEx contains a pointer to 'grip' item which is used in
// order to gather the structure in which the overlappedEx object was created.
type overlappedEx struct {
	syscall.Overlapped
	parent *grip
}

// newGrip creates a new file handle that can be used in overlapped operations.
// Then, the handle is associated with I/O completion port 'cph' and its value
// is stored in newly created 'grip' object.
func newGrip(cph syscall.Handle, parent *watched, filter uint32) (*grip, error) {
	g := &grip{
		handle:    syscall.InvalidHandle,
		filter:    filter,
		recursive: parent.recursive,
		pathw:     parent.pathw,
		parent:    parent,
		ovlapped:  &overlappedEx{},
	}
	if err := g.register(cph); err != nil {
		return nil, err
	}
	g.ovlapped.parent = g
	return g, nil
}

// NOTE : Thread safe
func (g *grip) register(cph syscall.Handle) (err error) {
	if g.handle, err = syscall.CreateFile(
		&g.pathw[0],
		syscall.FILE_LIST_DIRECTORY,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OVERLAPPED,
		0,
	); err != nil {
		return
	}
	if _, err = syscall.CreateIoCompletionPort(g.handle, cph, 0, 0); err != nil {
		syscall.CloseHandle(g.handle)
		return
	}
	return g.readDirChanges()
}

// readDirChanges tells the system to store file change information in grip's
// buffer. Directory changes that occur between calls to this function are added
// to the buffer and then, returned with the next call.
func (g *grip) readDirChanges() error {
	return syscall.ReadDirectoryChanges(
		g.handle,
		&g.buffer[0],
		uint32(unsafe.Sizeof(g.buffer)),
		g.recursive,
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
	e := Event(filter & (onlyNGlobalEvents | onlyNotifyChanges))
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
	return uint32(e)
}

// watched is made in order to check whether an action comes from a directory or
// file. This approach requires two file handlers per single monitored folder. The
// second grip handles actions which include creating or deleting a directory. If
// these processes are not monitored, only the first grip is created.
type watched struct {
	filter    uint32
	recursive bool
	count     uint8
	pathw     []uint16
	digrip    [2]*grip
}

// newWatched creates a new watched instance. It splits the filter variable into
// two parts. The first part is responsible for watching all events which can be
// created for a file in watched directory structure and the second one watches
// only directory Create/Delete actions. If all operations succeed, the Create
// message is sent to I/O completion port queue for further processing.
func newWatched(cph syscall.Handle, filter uint32, recursive bool,
	path string) (wd *watched, err error) {
	wd = &watched{
		filter:    filter,
		recursive: recursive,
	}
	if wd.pathw, err = syscall.UTF16FromString(path); err != nil {
		return
	}
	if err = wd.recreate(cph); err != nil {
		return
	}
	return wd, nil
}

// TODO : doc
func (wd *watched) recreate(cph syscall.Handle) (err error) {
	filefilter := wd.filter &^ uint32(FILE_NOTIFY_CHANGE_DIR_NAME)
	if err = wd.updateGrip(0, cph, filefilter == 0, filefilter); err != nil {
		return
	}
	dirfilter := wd.filter & uint32(FILE_NOTIFY_CHANGE_DIR_NAME|Create|Delete)
	if err = wd.updateGrip(1, cph, dirfilter == 0,
		wd.filter|uint32(dirmarker)); err != nil {
		return
	}
	wd.filter &^= onlyMachineStates
	return
}

// TODO : doc
func (wd *watched) updateGrip(idx int, cph syscall.Handle, reset bool,
	newflag uint32) (err error) {
	if reset {
		wd.digrip[idx] = nil
	} else {
		if wd.digrip[idx] == nil {
			if wd.digrip[idx], err = newGrip(cph, wd, newflag); err != nil {
				wd.closeHandle()
				return
			}
		} else {
			wd.digrip[idx].filter = newflag
			wd.digrip[idx].recursive = wd.recursive
			if err = wd.digrip[idx].register(cph); err != nil {
				wd.closeHandle()
				return
			}
		}
		wd.count++
	}
	return
}

// closeHandle closes handles that are stored in digrip array. Function always
// tries to close all of the handlers before it exits, even when there are errors
// returned from the operating system kernel.
func (wd *watched) closeHandle() (err error) {
	for _, g := range wd.digrip {
		if g != nil && g.handle != syscall.InvalidHandle {
			switch suberr := syscall.CloseHandle(g.handle); {
			case suberr == nil:
				g.handle = syscall.InvalidHandle
			case err == nil:
				err = suberr
			}
		}
	}
	return
}

// watcher implements Watcher interface. It stores a set of watched directories.
// All operations which remove watched objects from map `m` must be performed in
// loop goroutine since these structures are used internally by operating system.
type watcher struct {
	sync.Mutex
	m   map[string]*watched
	cph syscall.Handle
	c   chan<- EventInfo
}

// NewWatcher creates new non-recursive watcher backed by ReadDirectoryChangesW.
func newWatcher(c chan<- EventInfo) (w *watcher) {
	w = &watcher{
		m:   make(map[string]*watched),
		cph: syscall.InvalidHandle,
		c:   c,
	}
	runtime.SetFinalizer(w, func(w *watcher) {
		if w.cph != syscall.InvalidHandle {
			syscall.CloseHandle(w.cph)
		}
	})
	return
}

// Watch implements notify.Watcher interface.
func (w *watcher) Watch(path string, event Event) error {
	return w.watch(path, event, false)
}

// RecursiveWatch implements notify.RecursiveWatcher interface.
func (w *watcher) RecursiveWatch(path string, event Event) error {
	return w.watch(path, event, true)
}

// watch inserts a directory to the group of watched folders. If watched folder
// already exists, function tries to rewatch it with new filters(NOT VALID). Moreover,
// watch starts the main event loop goroutine when called for the first time.
func (w *watcher) watch(path string, event Event, recursive bool) (err error) {
	w.Lock()
	wd, ok := w.m[path]
	w.Unlock()
	if !ok {
		if err = w.lazyinit(); err != nil {
			return
		}
		w.Lock()
		if wd, ok = w.m[path]; ok {
			w.Unlock()
			return
		}
		if wd, err = newWatched(w.cph, uint32(event), recursive, path); err != nil {
			w.Unlock()
			return
		}
		w.m[path] = wd
		w.Unlock()
	}
	return nil
}

// lazyinit creates an I/O completion port and starts the main event processing
// loop. This method uses Double-Checked Locking optimization.
func (w *watcher) lazyinit() (err error) {
	invalid := uintptr(syscall.InvalidHandle)
	if atomic.LoadUintptr((*uintptr)(&w.cph)) == invalid {
		w.Lock()
		defer w.Unlock()
		if atomic.LoadUintptr((*uintptr)(&w.cph)) == invalid {
			cph := syscall.InvalidHandle
			if cph, err = syscall.CreateIoCompletionPort(
				cph, 0, 0, 0); err != nil {
				return
			}
			w.cph = cph
			go w.loop()
		}
	}
	return
}

// TODO(pknap) : doc
func (w *watcher) loop() {
	var n, key uint32
	var overlapped *syscall.Overlapped
	for {
		err := syscall.GetQueuedCompletionStatus(w.cph, &n, &key,
			&overlapped, syscall.INFINITE)
		if key == stateCPClose {
			w.Lock()
			handle := w.cph
			w.cph = syscall.InvalidHandle
			w.Unlock()
			syscall.CloseHandle(handle)
			return
		}
		if overlapped == nil {
			// TODO: check key == rewatch delete or 0(panic)
			continue
		}
		overEx := (*overlappedEx)(unsafe.Pointer(overlapped))
		if n == 0 {
			w.loopstate(overEx)
		} else {
			w.loopevent(n, overEx)
			if err = overEx.parent.readDirChanges(); err != nil {
				// TODO: error handling
			}
		}
	}
}

// TODO(pknap) : doc
func (w *watcher) loopstate(overEx *overlappedEx) {
	filter := atomic.LoadUint32(&overEx.parent.parent.filter)
	if filter&onlyMachineStates == 0 {
		return
	}
	if overEx.parent.parent.count--; overEx.parent.parent.count == 0 {
		switch filter & onlyMachineStates {
		case stateRewatch:
			w.Lock()
			overEx.parent.parent.recreate(w.cph)
			w.Unlock()
		case stateUnwatch:
			w.Lock()
			delete(w.m, syscall.UTF16ToString(overEx.parent.pathw))
			w.Unlock()
		case stateCPClose:
		default:
			panic(`notify: windows loopstate logic error`)
		}
	}
}

// TODO(pknap) : doc
func (w *watcher) loopevent(n uint32, overEx *overlappedEx) {
	events := []*event{}
	var currOffset uint32
	for {
		raw := (*syscall.FileNotifyInformation)(unsafe.Pointer(
			&overEx.parent.buffer[currOffset]))
		name := syscall.UTF16ToString((*[syscall.MAX_PATH]uint16)(
			unsafe.Pointer(&raw.FileName))[:raw.FileNameLength>>1])
		events = append(events, &event{
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
	w.send(events)
}

// TODO(pknap) : doc
func (w *watcher) send(es []*event) {
	for _, e := range es {
		if e.e = decode(e.filter, e.action); e.e == 0 {
			continue
		}
		switch Event(e.action) {
		case (FILE_ACTION_ADDED >> 12), (FILE_ACTION_REMOVED >> 12):
			if e.filter&uint32(dirmarker) != 0 {
				e.objtype = ObjectDirectory
			} else {
				e.objtype = ObjectFile
			}
		default:
			e.objtype = ObjectUnknown
		}
		w.c <- e
	}
}

// Rewatch implements notify.Rewatcher interface.
func (w *watcher) Rewatch(path string, oldevent, newevent Event) error {
	return w.rewatch(path, uint32(oldevent), uint32(newevent), false)
}

// RecursiveRewatch implements notify.RecursiveRewatcher interface.
func (w *watcher) RecursiveRewatch(oldpath, newpath string, oldevent,
	newevent Event) error {
	switch {
	case oldpath != newpath:
		if err := w.unwatch(oldpath); err != nil {
			return err
		}
		return w.watch(newpath, newevent, true)
	case oldevent != newevent:
		return w.rewatch(newpath, uint32(oldevent), uint32(newevent), true)
	}
	return nil
}

// TODO : (pknap) doc.
func (w *watcher) rewatch(path string, oldevent, newevent uint32,
	recursive bool) (err error) {
	var wd *watched
	w.Lock()
	if wd, err = w.nonStateWatched(path); err != nil {
		w.Unlock()
		return
	}
	if wd.filter&(onlyNotifyChanges|onlyNGlobalEvents) != oldevent {
		panic(`notify: windows re-watcher logic error`)
	}
	wd.filter = stateRewatch | newevent
	wd.recursive, recursive = recursive, wd.recursive
	if err = wd.closeHandle(); err != nil {
		wd.filter = oldevent
		wd.recursive = recursive
		w.Unlock()
		return
	}
	w.Unlock()
	return
}

// TODO : pknap
func (w *watcher) nonStateWatched(path string) (wd *watched, err error) {
	wd, ok := w.m[path]
	if !ok || wd == nil {
		err = errors.New(`notify: ` + path + ` path is unwatched`)
		return
	}
	if filter := atomic.LoadUint32(&wd.filter); filter&onlyMachineStates != 0 {
		err = errors.New(`notify: another re/unwatching operation in progress`)
		return
	}
	return
}

// Unwatch implements notify.Watcher interface.
func (w *watcher) Unwatch(path string) error {
	return w.unwatch(path)
}

// RecursiveUnwatch implements notify.RecursiveWatcher interface.
func (w *watcher) RecursiveUnwatch(path string) error {
	return w.unwatch(path)
}

// TODO : pknap
func (w *watcher) unwatch(path string) (err error) {
	var wd *watched
	w.Lock()
	if wd, err = w.nonStateWatched(path); err != nil {
		w.Unlock()
		return
	}
	wd.filter |= stateUnwatch
	if err = wd.closeHandle(); err != nil {
		wd.filter &^= stateUnwatch
		w.Unlock()
		return
	}
	w.Unlock()
	return
}

// Close resets the whole watcher object, closes all existing file descriptors,
// and sends stateCPClose state as completion key to the main watcher's loop.
func (w *watcher) Close() (err error) {
	w.Lock()
	if w.cph == syscall.InvalidHandle {
		w.Unlock()
		return nil
	}
	for _, wd := range w.m {
		wd.filter &^= onlyMachineStates
		wd.filter |= stateCPClose
		if e := wd.closeHandle(); e != nil && err == nil {
			err = e
		}
	}
	w.Unlock()
	if e := syscall.PostQueuedCompletionStatus(
		w.cph, 0, stateCPClose, nil); e != nil && err == nil {
		return e
	}
	return
}

// decode creates a notify event from both non-raw filter and action which was
// returned from completion routine. Function may return Event(0) in case when
// filter was replaced by a new value which does not contain fields that are
// valid with passed action.
func decode(filter, action uint32) Event {
	switch action {
	case syscall.FILE_ACTION_ADDED:
		return addrmv(filter, Create, FILE_ACTION_ADDED)
	case syscall.FILE_ACTION_REMOVED:
		return addrmv(filter, Delete, FILE_ACTION_REMOVED)
	case syscall.FILE_ACTION_MODIFIED:
		return Write
	case syscall.FILE_ACTION_RENAMED_OLD_NAME:
		return addrmv(filter, Move, FILE_ACTION_RENAMED_OLD_NAME)
	case syscall.FILE_ACTION_RENAMED_NEW_NAME:
		return addrmv(filter, Move, FILE_ACTION_RENAMED_NEW_NAME)
	}
	panic(`notify: cannot decode internal mask`)
}

// addrmv decides whether the Windows action or the system-independent event
// should be returned. Since the grip's filter may be atomically changed during
// watcher lifetime, it is possible that neither Windows nor notify masks are
// present in variable memory.
func addrmv(filter uint32, e, syse Event) Event {
	isdir := filter&uint32(dirmarker) != 0
	switch {
	case isdir && filter&uint32(FILE_NOTIFY_CHANGE_DIR_NAME) != 0 ||
		!isdir && filter&uint32(FILE_NOTIFY_CHANGE_FILE_NAME) != 0:
		return syse
	case filter&uint32(e) != 0:
		return e
	default:
		return Event(0)
	}
}

// TODO(pknap) : add system-dependent event decoder for FILE_ACTION_MODIFIED action.
