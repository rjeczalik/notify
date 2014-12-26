// +build windows

package notify

// TODO(rjeczalik): this repo works only go1.4
// import "golang.org/x/sys/windows"

import "syscall"

var modkernel32 = syscall.NewLazyDLL("kernel32.dll")
var procSetSystemFileCacheSize = modkernel32.NewProc("SetSystemFileCacheSize")

// TODO
func Sync() {
	r, _, err := procSetSystemFileCacheSize.Call(-1, -1, 0)
	if r == 0 {
		panic(err)
	}
}
