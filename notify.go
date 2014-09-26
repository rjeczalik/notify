package notify

var r = NewRuntime()

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) {
	r.Watch(name, c, events...)
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	r.Stop(c)
}
