package notify

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestAppendset(t *testing.T) {
	cases := [...]struct {
		start []string
		vals  map[string]struct{}
		end   []string
	}{
		0: {
			[]string{"a", "e", "x"},
			map[string]struct{}{"A": {}, "E": {}, "x": {}},
			[]string{"A", "E", "a", "e", "x"},
		},
		1: {
			[]string{},
			map[string]struct{}{"/a/b/c/d": {}, "/b/c/d": {}, "/c/d": {}, "/d": {}},
			[]string{"/a/b/c/d", "/b/c/d", "/c/d", "/d"},
		},
		2: {
			[]string{"a", "b", "c"},
			map[string]struct{}{"d": {}, "e": {}, "f": {}},
			[]string{"a", "b", "c", "d", "e", "f"},
		},
	}
	for i := range cases {
		s := cases[i].start
		for v := range cases[i].vals {
			s = appendset(s, v)
		}
		if !reflect.DeepEqual(s, cases[i].end) {
			t.Errorf("want s=%v; got %v (i=%d)", cases[i].end, s, i)
		}
	}
}

func TestSplitpath(t *testing.T) {
	cases := map[string][]string{
		"C:/a/b/c/d.txt": {"a", "b", "c", "d.txt"},
		"/a/b/c/d.txt":   {"a", "b", "c", "d.txt"},
		"":               nil,
		".":              nil,
		"C:":             nil,
	}
	for path, names := range cases {
		path = filepath.FromSlash(path)
		if s := splitpath(path); !reflect.DeepEqual(s, names) {
			t.Errorf("want s=%v; got %v (path=%s)", names, s, path)
		}
	}
}

func TestJoinevents(t *testing.T) {
	allFile := All & ^Recursive
	cases := [...]struct {
		events []Event
		isdir  bool
		exp    Event
	}{
		0:  {nil, true, All},
		1:  {nil, false, allFile},
		2:  {[]Event{}, true, All},
		3:  {[]Event{}, false, allFile},
		4:  {[]Event{Create}, false, Create},
		5:  {[]Event{Create}, true, Create},
		6:  {[]Event{Recursive}, false, allFile},
		7:  {[]Event{Recursive}, true, All},
		8:  {[]Event{Rename, Recursive}, true, Rename | Recursive},
		9:  {[]Event{Rename, Recursive}, false, Rename},
		10: {[]Event{Create, Write, Remove}, true, Create | Write | Remove},
		11: {[]Event{Create, Write, Remove}, false, Create | Write | Remove},
	}
	for i, cas := range cases {
		if event := joinevents(cas.events, cas.isdir); event != cas.exp {
			t.Errorf("want event=%v; got %v (i=%d)", cas.exp, event, i)
		}
	}
}

func TestWalkpath(t *testing.T) {
	cases := map[string]struct {
		p  []string
		ok bool
	}{
		"C:/a/b/c/d.txt": {[]string{"a", "b", "c", "d.txt"}, true},
		"/a/b/c/d.txt":   {[]string{"a", "b", "c", "d.txt"}, true},
		"":               {[]string{}, false},
		".":              {[]string{}, false},
		"C:":             {[]string{}, false},
	}
	var p []string
	fn := func(s string) bool {
		p = append(p, s)
		return s != "break"
	}
	for path, cas := range cases {
		p, path = p[:0], filepath.FromSlash(path)
		if ok := walkpath(path, fn); ok != cas.ok {
			t.Errorf("want ok=%v; got %v (path=%s)", cas.ok, ok, path)
			continue
		}
		if !reflect.DeepEqual(p, cas.p) {
			t.Errorf("want p=%v; got %v (path=%s)", cas.p, p, path)
		}
	}
}
