// +build !windows

package notify

import "golang.org/x/sys/unix"

// TODO
func Sync() {
	unix.Sync()
}
