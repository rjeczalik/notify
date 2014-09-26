package notify

var notifier *Runtime

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) {
	notifier.Watch(name, c, events...)
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	notifier.Stop(c)
}
