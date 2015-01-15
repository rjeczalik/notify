// +build windows

package notify

// TODO(rjeczalik): this repo works only go1.4
// import "golang.org/x/sys/windows"

import (
	"syscall"
	"unsafe"
)

var modkernel32 = syscall.NewLazyDLL("kernel32.dll")
var procSetSystemFileCacheSize = modkernel32.NewProc("SetSystemFileCacheSize")
var none = uintptr(1<<(unsafe.Sizeof(uintptr(0))*8) - 1)

// TODO
func Sync() {
	r, _, err := procSetSystemFileCacheSize.Call(none, none, 0)
	if r == 0 {
		// TODO(pknap): does not work without admin privilages, but I'm going
		// to hack it.
		dbg.Print("SetSystemFileCacheSize error:", err)
	}
}
