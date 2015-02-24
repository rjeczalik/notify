package notify

// buffer TODO(rjeczalik)
const buffer = 128

// tree TODO(rjeczalik)
type tree interface {
	Watch(string, chan<- EventInfo, ...Event) error
	Stop(chan<- EventInfo)
	Close() error
}

// newTree TODO(rjeczalik)
func newTree() tree {
	c := make(chan EventInfo, buffer)
	w := newWatcher(c)
	if rw, ok := w.(recursiveWatcher); ok {
		return newRecursiveTree(rw, c)
	}
	return newNonrecursiveTree(w, c, make(chan EventInfo, buffer))
}
