package notify

import "sync"

var once sync.Once
var m sync.Mutex
var g *Tree

func tree() *Tree {
	once.Do(func() {
		if g == nil {
			c := make(chan EventInfo, 128)
			g = NewTree(NewWatcher(c), c)
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
