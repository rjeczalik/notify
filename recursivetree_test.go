package notify

import "testing"

func TestRecursiveTreeWatch(t *testing.T) {
	n := NewRecursiveTreeTest(t, "testdata/vfs.txt")
	defer n.Close()

	t.Skip("TODO(rjeczalik)")
}
