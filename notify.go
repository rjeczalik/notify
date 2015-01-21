package notify

import "sync"

type notifier interface {
	Watch(string, chan<- EventInfo, ...Event) error
	Stop(chan<- EventInfo)
}

func newNotifier(w watcher, c <-chan EventInfo) notifier {
	if rw, ok := w.(recursiveWatcher); ok {
		// return NewRecursiveTree(rw, c)
		_ = rw // TODO
		return newTree(w, c)
	}
	return newTree(w, c)
}

var once sync.Once
var m sync.Mutex
var g notifier

func tree() notifier {
	once.Do(func() {
		if g == nil {
			c := make(chan EventInfo, 128)
			g = newNotifier(newWatcher(c), c)
		}
	})
	return g
}

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) (err error) {
	m.Lock()
	err = tree().Watch(name, c, events...)
	m.Unlock()
	return
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	m.Lock()
	tree().Stop(c)
	m.Unlock()
	return
}
