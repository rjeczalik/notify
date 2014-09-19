package notify

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	wd  string
	sep = string(os.PathSeparator)
)

func init() {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	wd = dir
}

// Abs
func abs(path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(wd, path)
	}
	return filepath.Clean(path)
}

// Appendset
//
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

// Splitpath
func splitpath(p string) []string {
	if p == "" || p == "." || p == sep {
		return nil
	}
	i := strings.Index(p, sep) + 1
	if i == 0 || i == len(p) {
		return nil
	}
	var s []string
	for i < len(p) {
		j := strings.Index(p[i:], sep)
		if j == -1 {
			j = len(p) - i
		}
		s, i = append(s, p[i:i+j]), i+j+1
	}
	return s
}

// TODO(rjeczalik): Remove isdir? Don't care if path is a file or a directory at
// the moment of creating a watcher?
func joinevents(events []Event, isdir bool) (e Event) {
	if len(events) == 0 || (len(events) == 1 && events[0] == Recursive) {
		e = All
	} else {
		for _, event := range events {
			e |= event
		}
	}
	if !isdir {
		e &= ^Recursive
	}
	return
}

func splitevents(e Event) (events []Event) {
	for _, event := range []Event{Create, Delete, Write, Move, Recursive} {
		if e&event != 0 {
			events = append(events, event)
		}
	}
	return
}

// Walkpath
func walkpath(p string, fn func(string) bool) bool {
	if p == "" || p == "." {
		return false
	}
	i, n := strings.Index(p, sep)+1, len(p)
	if i == 0 || i == n {
		return false
	}
	for i < n {
		j := strings.Index(p[i:], sep)
		if j == -1 {
			j = n - i
		}
		if !fn(p[i : i+j]) {
			return i+j+2 > n
		}
		i += j + 1
	}
	return true
}
