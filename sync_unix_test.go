// +build !windows

package notify

// TODO(rjeczalik): this repo works only with go1.4
// import "golang.org/x/sys/unix"

import "syscall"

// Sync TODO
func Sync() {
	syscall.Sync()
}
