// +build windows

package notify

import "golang.org/x/sys/windows"

var modkernel32 = windows.NewLazyDLL("kernel32.dll")
var procSetSystemFileCacheSize = modkernel32.NewProc("SetSystemFileCacheSize")

// TODO
func Sync() {
	r, _, err := procSetSystemFileCacheSize.Call(-1, -1, 0)
	if r == 0 {
		panic(err)
	}
}
