package notify

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// TODO(rjeczalik): Move test fixture to test package

type visited string
type end string

func mark(s string) func(Point, bool) error {
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

func sendlast(c chan<- Point) func(Point, bool) error {
	return func(pt Point, last bool) error {
		if last {
			c <- pt
		}
		return nil
	}
}

func checkmark(t *testing.T, it map[string]interface{}, mark string, dirs []string) {
	for i, dir := range dirs {
		v, ok := it[dir]
		if !ok {
			t.Errorf("dir not found (mark=%q, i=%d)", mark, i)
			break
		}
		if it, ok = v.(map[string]interface{}); !ok {
			t.Errorf("want typeof(v)=map[string]interface; got %+v (mark=%q, i=%d)",
				v, mark, i)
			break
		}
		if v, ok = it[""]; !ok {
			t.Errorf("dir has no mark (mark=%q, i=%d)", mark, i)
			break
		}
		typ := reflect.TypeOf(visited(""))
		if i == len(dirs)-1 {
			typ = reflect.TypeOf(end(""))
		}
		if got := reflect.TypeOf(v); got != typ {
			t.Errorf("want typeof(v)=%v; got %v (mark=%q, i=%d)", typ, got, mark, i)
			continue
		}
		if reflect.ValueOf(v).String() != mark {
			t.Errorf("want v=%v; got %v (mark=%q, i=%d)", mark, v, mark, i)
			continue
		}
		delete(it, "") // remove visitation mark
	}
}

// Test for dangling marks - if a mark is present, WalkPoint went somewhere
// it shouldn't.
func checkdangling(t *testing.T, w *WatchPointTree) {
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

func TestWalkPoint(t *testing.T) {
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
	// Don't use filepath.VolumeName and make the following regular test-cases?
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
		cases[`\\tsoh\erahs\Users\rjeczalik`] = []string{"Users", "rjeczalik"}
	}
	w := NewWatchPointTree()
	for path, dirs := range cases {
		path = filepath.Clean(filepath.FromSlash(path))
		if err := w.WalkPoint(path, mark(path)); err != nil {
			t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		checkmark(t, w.root, path, dirs)
	}
	checkdangling(t, w)
}

func TestWalkPointWithCwd(t *testing.T) {
	type Case struct {
		Cwd  string
		Dirs []string
	}
	cases := map[string]Case{
		"/tmp/a/b/c/d": {
			"/tmp/a/b",
			[]string{"c", "d"},
		},
		"/tmp/a": {
			"/tmp",
			[]string{"a"},
		},
		"/home/rjeczalik/src/github.com": {
			"/home/rjeczalik",
			[]string{"src", "github.com"},
		},
		"/": {
			"",
			[]string{},
		},
		"//": {
			"/",
			[]string{},
		},
		"/a/b/c/d/e/f/g/h/j/k": {
			"/a/b/c/d/e/f",
			[]string{"g", "h", "j", "k"},
		},
		"": {},
	}
	// Don't use filepath.VolumeName and make the following regular test-cases?
	if runtime.GOOS == "windows" {
		cases[`C:`] = Case{}
		cases[`C:\`] = Case{}
		cases[`C\Windows\Temp`] = Case{`C:\Windows`, []string{"Temp"}}
		cases[`D:\Windows\Temp`] = Case{`D:\Windows`, []string{"Temp"}}
		cases[`E:\Windows\Temp\Local`] = Case{`E:\Windows`, []string{"Temp", "Local"}}
		cases[`\\host\share\Windows`] = Case{`\\host\share`, []string{"Windows"}}
		cases[`\\host\share\Windows\Temp`] = Case{`\\host\share\Windows`, []string{"Temp"}}
		cases[`\\host1\share\Windows\system32`] = Case{`\\host1\share`, []string{"Windows", "system32"}}
	}
	w := NewWatchPointTree()
	for path, cas := range cases {
		path = filepath.Clean(filepath.FromSlash(path))
		c := make(chan Point, 1)
		// Prepare - look up cwd Point by walking its subpath.
		if err := w.WalkPoint(filepath.Join(cas.Cwd, "test"), sendlast(c)); err != nil {
			t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		select {
		case w.cwd = <-c:
			w.cwd.Name = cas.Cwd
		default:
			t.Errorf("unable to find cwd Point (path=%q)", path)
		}
		// Actual test.
		if err := w.WalkPoint(path, mark(path)); err != nil {
			t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		checkmark(t, w.cwd.Parent, path, cas.Dirs)
	}
	checkdangling(t, w)
}
