package notify

import "testing"

func TestWatcher(t *testing.T) {
	ei := []EventInfo{
		EI("github.com/rjeczalik/fs/fs_test.go", Create),
		EI("github.com/rjeczalik/fs/binfs/", Create),
		EI("github.com/rjeczalik/fs/binfs.go", Create),
		EI("github.com/rjeczalik/fs/binfs_test.go", Create),
		EI("github.com/rjeczalik/fs/binfs/", Delete),
		EI("github.com/rjeczalik/fs/binfs/", Create),
		EI("github.com/rjeczalik/fs/virfs", Create),
		EI("github.com/rjeczalik/fs/virfs", Delete),
		EI("file", Create),
		EI("dir/", Create),
	}
	fixture.Cases(t).ExpectEvents(NewWatcher(), All, ei)
}
