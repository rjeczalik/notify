package notify

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

type visited string
type end string

func markpoint(s string) func(Point, bool) error {
	return func(pt Point, last bool) (err error) {
		if last {
			dir, ok := pt.Parent[pt.Name].(map[string]interface{})
			if !ok {
				dir = make(map[string]interface{})
				pt.Parent[pt.Name] = dir
			}
			dir[""] = end(s)
		}
		pt.Parent[""] = visited(s)
		return
	}
}

func TestWatchPointTreeWalkPoint(t *testing.T) {
	// For each test-case we're traversing path specified by a testcase's key
	// over shared WatchPointTree and marking each directory using special empty
	// key. The mark is simply the traversed path name. Each mark can be either
	// of `visited` or `end` type. Only the last item in the path is marked with
	// an `end` mark.
	cases := map[string][]string{
		"/tmp":                           {"tmp"},
		"/home/rjeczalik":                {"home", "rjeczalik"},
		"/":                              {},
		"/h/o/m/e/":                      {"h", "o", "m", "e"},
		"/home/rjeczalik/src/":           {"home", "rjeczalik", "src"},
		"/home/user/":                    {"home", "user"},
		"/home/rjeczalik/src/github.com": {"home", "rjeczalik", "src", "github.com"},
	}
	if runtime.GOOS == "windows" {
		cases[`C:`] = []string{}
		cases[`C:\`] = []string{}
		cases[`C:\Windows\Temp`] = []string{"Windows", "Temp"}
		cases[`D:\Windows\Temp`] = []string{"Windows", "Temp"}
		cases[`F:\`] = []string{}
		cases[`\\host\share\`] = []string{}
		cases[`F:\abc`] = []string{"abc"}
		cases[`D:\abc`] = []string{"abc"}
		cases[`F:\Windows`] = []string{"Windows"}
		cases[`\\host\share\Windows\Temp`] = []string{"Windows", "Temp"}
	}
	w := NewWatchPointTree()
	for path, dirs := range cases {
		path = filepath.Clean(filepath.FromSlash(path))
		if err := w.WalkPoint(path, markpoint(path)); err != nil {
			t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		it := w.root
		for i, dir := range dirs {
			v, ok := it[dir]
			if !ok {
				t.Errorf("dir=%q not found (path=%q, i=%d)", dir, path, i)
				break
			}
			if it, ok = v.(map[string]interface{}); !ok {
				t.Errorf("want typeof(v)=map[string]interface; got %+v (path=%q, i=%d)",
					v, path, i)
				break
			}
			if v, ok = it[""]; !ok {
				t.Errorf("want node to be marked as visited (path=%q, i=%d)", path, i)
				break
			}
			typ := reflect.TypeOf(visited(""))
			if i == len(dirs)-1 {
				typ = reflect.TypeOf(end(""))
			}
			if got := reflect.TypeOf(v); got != typ {
				t.Errorf("want typeof(v)=%v; got %v (path=%q, i=%d)", typ, got, path, i)
				continue
			}
			if reflect.ValueOf(v).String() != path {
				t.Errorf("want v=%v; got %v (path=%q, i=%d)", path, v, path, i)
				continue
			}
			delete(it, "") // remove visitation mark
		}
	}
	// Test for dangling marks - if a mark is present, WalkPoint went somewhere
	// it shouldn't.
	w.WalkPoint("/", func(pt Point, _ bool) error {
		if v, ok := pt.Parent[""]; ok {
			t.Errorf("dangling mark=%+v found at parent of %q", v, pt.Name)
		}
		if dir, ok := pt.Parent[pt.Name].(map[string]interface{}); ok {
			if v, ok := dir[""]; ok {
				t.Errorf("dangling mark=%+v found at %q", v, pt.Name)
			}
		} else {
			t.Errorf("dir=%q not found", pt.Name)
		}
		return nil
	})
}
