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
		{
			[]string{"a", "e", "x"},
			map[string]struct{}{"A": {}, "E": {}, "x": {}},
			[]string{"A", "E", "a", "e", "x"},
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

func TestSplitabs(t *testing.T) {
	cases := map[string][]string{
		"C:/a/b/c/d.txt": {"a", "b", "c", "d.txt"},
		"/a/b/c/d.txt":   {"a", "b", "c", "d.txt"},
	}
	for path, names := range cases {
		if s := splitabs(filepath.FromSlash(path)); !reflect.DeepEqual(s, names) {
			t.Errorf("want s=%v; got %v (path=%s)", names, s, path)
		}
	}
}
