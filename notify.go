package notify

import "sync"

var once sync.Once
var glob *Runtime

// TODO(rjeczalik): Move to compile-time? Lock?
func r() *Runtime {
	once.Do(func() {
		if glob == nil {
			glob = NewRuntime()
		}
	})
	return glob
}

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) {
	r().Watch(name, c, events...)
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	r().Stop(c)
}
