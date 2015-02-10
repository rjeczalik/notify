// +build windows

package notify

// TODO(rjeczalik): this repo works only go1.4
// import "golang.org/x/sys/windows"

import (
	"syscall"
	"time"
	"unsafe"
)

var modkernel32 = syscall.NewLazyDLL("kernel32.dll")
var procSetSystemFileCacheSize = modkernel32.NewProc("SetSystemFileCacheSize")
var zero = uintptr(1<<(unsafe.Sizeof(uintptr(0))*8) - 1)

// TODO
func Sync() {
	// TODO(pknap): does not work without admin privilages, but I'm going
	// to hack it.
	// r, _, err := procSetSystemFileCacheSize.Call(none, none, 0)
	// if r == 0 {
	//   dbg.Print("SetSystemFileCacheSize error:", err)
	// }
}

// UpdateWait pauses the program for some minimal amount of time. This function
// is required only by implementations which work asynchronously. It gives
// watcher structure time to update its internal state.
func UpdateWait() {
	time.Sleep(50 * time.Millisecond)
}
