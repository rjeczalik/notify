// +build linux

package notify

import (
	"fmt"
	"os"
	//"path/filepath"
	"runtime"
	"sync"
	"syscall"
)

// TODO(ppknap) : doc.
const maxEventSize = syscall.SizeofInotifyEvent + syscall.PathMax + 1

// TODO(ppknap) : This type should be dropped. It is useless since
// we use it only here. Use:
//  var handlers struct {
//  	...
//  }
type handlersType struct {
	sync.RWMutex
	m      map[int]*watched
	fd     *int
	buffer [64 * maxEventSize]byte
	// TODO(ppknap) : I don't like this:<.
	c chan EventInfo
}

// TODO(ppknap) : doc.
var handlers handlersType

// TODO(ppknap) : doc.
type watched struct {
	pathname  string
	mask      uint32
	watchdesc int
}

// TODO(ppknap) : doc.
func init() {
	fd, err := syscall.InotifyInit()
	if err != nil {
		panic(os.NewSyscallError("InotifyInit", err))
	}

	handlers.fd = &fd
	runtime.SetFinalizer(handlers.fd, func(fd *int) {
		syscall.Close(*fd)
	})
	// TODO(ppknap) : this should be removed:<.
	handlers.c = make(chan EventInfo) // TODO(pknap) : rm me
	global.Watcher = &handlers
	go loop()
}

// TODO(ppknap) : doc.
func loop() {
	for {
		process()
	}
}

// TODO(ppknap) : doc.
func process() {
	var n int
	var err error

	n, err = syscall.Read(*handlers.fd, handlers.buffer[:])

	fmt.Println("\n====================\n")
	fmt.Println("NumberOfBytes:", n)
	fmt.Println("Error:", err)
	fmt.Println("\n====================\n")

	if n != 0 {
	} else {
		fmt.Println("no data received")
	}
}

func watch(p string, e Event) error {

	return nil
}

func unwatch(p string) error {
	return nil
}

// TODO(ppknap) : Does I have to know about this function's implementation?
func sendevent(ei EventInfo) {
	handlers.c <- ei
}

// TODO(rjeczalik) : could this be platform independent?
type event struct {
	name  string
	ev    Event
	isdir bool // tribool - yes, no, can't tell?
}

func (e event) Event() Event     { return e.ev }
func (e event) IsDir() bool      { return e.isdir }
func (e event) Name() string     { return e.name }
func (e event) Sys() interface{} { return nil } // no-one cares about fsnotify.Event

// IsRecursive implements notify.Watcher interface.
func (h *handlersType) IsRecursive() (nope bool) { return }

// Watch implements notify.Watcher interface.
func (h *handlersType) Watch(p string, e Event) error {
	return watch(p, e)
}

// Unwatch implements notify.Watcher interface.
func (h *handlersType) Unwatch(p string) error {
	return unwatch(p)
}

// Fanin implements notify.Watcher interface.
func (h *handlersType) Fanin(c chan<- EventInfo) {
	go func() {
		for e := range h.c {
			c <- e
		}
	}()
}
