package notify

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	why "gopkg.in/fsnotify.v1"
)

var (
	wd  string
	sep       = string(os.PathSeparator)
	all Event = Create | Write | Remove | Rename | Recursive
)

func init() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	wd = dir
	w, err := why.NewWatcher()
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

func abs(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(wd, path)
	}
	return filepath.Clean(path)
}

// TODO(rjeczalik): Sort by directory depth?
func appendset(s []string, x string) []string {
	n := len(s)
	if n == 0 {
		return []string{x}
	}
	i := sort.SearchStrings(s, x)
	if i != n && s[i] == x {
		return s
	}
	if i == n {
		return append(s, x)
	}
	s = append(s, s[n-1])
	copy(s[i+1:], s[i:n])
	s[i] = x
	return s
}

func splitabs(p string) (s []string) {
	if p == "" || p == "." || p == sep {
		return
	}
	i := strings.Index(p, sep) + 1
	if i == 0 || i == len(p) {
		return
	}
	for i < len(p) {
		j := strings.Index(p[i:], sep)
		if j == -1 {
			j = len(p) - i
		}
		s, i = append(s, p[i:i+j]), i+j+1
	}
	return
}

type watcher struct {
	ch []chan<- string
	n  int
}

type fsnotify struct {
	sync.RWMutex
	w     *why.Watcher
	wtree map[string]interface{}
	m     map[chan<- EventInfo][]string
	refn  map[string]uint
}

func (fs *fsnotify) Watch(name string, c chan<- EventInfo, events ...Event) {
	if c == nil {
		panic("fs/notify: Watch using nil channel")
	}
	name = abs(name)
	fi, err := os.Stat(name)
	if err != nil {
		return
	}
	var evs Event
	if len(events) == 0 {
		evs = all
	}
	for _, ev := range events {
		evs |= ev
	}
	if !fi.IsDir() {
		evs &= ^Recursive
	}
	fs.Lock()
	fs.m[c] = appendset(fs.m[c], name) // fix
	// TODO reg in wtree
	fs.Unlock()
}

func (fs *fsnotify) Stop(ch chan<- EventInfo) {
	// TODO
}
