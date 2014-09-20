package notify

// Watch TODO
func Watch(name string, c chan<- EventInfo, events ...Event) {
	global.Watch(name, c, events...)
}

// Stop TODO
func Stop(c chan<- EventInfo) {
	global.Stop(c)
}

var global = dispatch{
	Tree: make(map[string]interface{}),
}
