package notify

var runtime *Runtime

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) {
	runtime.Watch(name, c, events...)
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	runtime.Stop(c)
}
