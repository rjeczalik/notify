package notify

import (
	"os"
	"sync"

	old "gopkg.in/fsnotify.v1"
)

func init() {
	w, err := old.NewWatcher()
	if err != nil {
		panic(err)
	}
	impl = &fsnotify{
		w:     w,
		wtree: make(map[string]interface{}),
		m:     make(map[chan<- EventInfo][]string),
		refn:  make(map[string]uint),
	}
}

type watcher struct {
	ch []chan<- string
	n  int
}

type fsnotify struct {
	sync.RWMutex
	w     *old.Watcher
	wtree map[string]interface{}
	m     map[chan<- EventInfo][]string
	refn  map[string]uint
}

func (fs *fsnotify) Watch(name string, c chan<- EventInfo, events ...Event) {
	if c == nil {
		panic("notify: Watch using nil channel")
	}
	name = abs(name)
	fi, err := os.Stat(name)
	if err != nil {
		return
	}
	_ = joinevents(events, fi.IsDir())
	// joinevents
	fs.Lock()
	fs.m[c] = appendset(fs.m[c], name)
	// TODO reg in wtree
	fs.Unlock()
}

func (fs *fsnotify) Stop(ch chan<- EventInfo) {
	// TODO
}
