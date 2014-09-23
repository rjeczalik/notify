package notify

var notify *Runtime

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) {
	notify.Watch(name, c, events...)
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	notify.Stop(c)
}
