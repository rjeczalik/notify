package notify

import "sync"

// TODO(rjeczalik): unexport types

type Notifier interface {
	Watch(string, chan<- EventInfo, ...Event) error
	Stop(chan<- EventInfo)
}

func NewNotifier(w Watcher, c <-chan EventInfo) Notifier {
	if rw, ok := w.(RecursiveWatcher); ok {
		// return NewRecursiveTree(rw, c)
		_ = rw // TODO
		return NewTree(w, c)
	}
	return NewTree(w, c)
}

var once sync.Once
var m sync.Mutex
var g Notifier

func tree() Notifier {
	once.Do(func() {
		if g == nil {
			c := make(chan EventInfo, 128)
			g = NewNotifier(NewWatcher(c), c)
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
