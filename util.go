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
