package notify

// Watcher
type watcher interface {
	Watch(string, Event) error //
	Unwatch(string) error      //
	Mux(chan<- EventInfo)      //
}
