package test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rjeczalik/notify"
)

type visited string
type end string

func mark(s string) func(notify.Point, bool) error {
	return func(pt notify.Point, last bool) (err error) {
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

func sendlast(c chan<- notify.Point) func(notify.Point, bool) error {
	return func(pt notify.Point, last bool) error {
		if last {
			c <- pt
		}
		return nil
	}
}

// WalkCase TODO
type WalkCase struct {
	C string
	W []string
}

type p struct {
	t *testing.T
	w *notify.WatchPointTree
}

// P TODO
func P(t *testing.T) *p {
	w := notify.NewWatchPointTree(nil)
	w.FS = FS
	return &p{t: t, w: w}
}

// Close TODO
func (p *p) Close() error {
	return p.w.Close()
}

func (p *p) expectmark(it map[string]interface{}, mark string, dirs []string) {
	for i, dir := range dirs {
		v, ok := it[dir]
		if !ok {
			p.t.Errorf("dir not found (mark=%q, i=%d)", mark, i)
			break
		}
		if it, ok = v.(map[string]interface{}); !ok {
			p.t.Errorf("want typeof(v)=map[string]interface; got %+v (mark=%q, i=%d)",
				v, mark, i)
			break
		}
		if v, ok = it[""]; !ok {
			p.t.Errorf("dir has no mark (mark=%q, i=%d)", mark, i)
			break
		}
		typ := reflect.TypeOf(visited(""))
		if i == len(dirs)-1 {
			typ = reflect.TypeOf(end(""))
		}
		if got := reflect.TypeOf(v); got != typ {
			p.t.Errorf("want typeof(v)=%v; got %v (mark=%q, i=%d)", typ, got, mark, i)
			continue
		}
		if reflect.ValueOf(v).String() != mark {
			p.t.Errorf("want v=%v; got %v (mark=%q, i=%d)", mark, v, mark, i)
			continue
		}
		delete(it, "") // remove visitation mark
	}
}

// Test for dangling marks - if a mark is present, WalkPoint went somewhere
// it shouldn't.
func (p *p) expectnomark() {
	p.w.WalkPoint("/", func(pt notify.Point, _ bool) error {
		if v, ok := pt.Parent[""]; ok {
			p.t.Errorf("dangling mark=%+v found at parent of %q", v, pt.Name)
		}
		if dir, ok := pt.Parent[pt.Name].(map[string]interface{}); ok {
			if v, ok := dir[""]; ok {
				p.t.Errorf("dangling mark=%+v found at %q", v, pt.Name)
			}
		} else {
			p.t.Errorf("dir=%q not found", pt.Name)
		}
		return nil
	})
}

// ExpectWalk TODO
//
// For each test-case we're traversing path specified by a testcase's key
// over shared WatchPointTree and marking each directory using special empty
// key. The mark is simply the traversed path name. Each mark can be either
// of `visited` or `end` type. Only the last item in the path is marked with
// an `end` mark.
func (p *p) ExpectWalk(cases map[string][]string) {
	for path, dirs := range cases {
		path = filepath.Clean(filepath.FromSlash(path))
		if err := p.w.WalkPoint(path, mark(path)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		p.expectmark(p.w.Root, path, dirs)
	}
	p.expectnomark()
}

// ExpectWalkCwd TODO
func (p *p) ExpectWalkCwd(cases map[string]WalkCase) {
	for path, cas := range cases {
		path = filepath.Clean(filepath.FromSlash(path))
		cas.C = filepath.Clean(filepath.FromSlash(cas.C))
		c := make(chan notify.Point, 1)
		// Prepare - look up cwd Point by walking its subpath.
		if err := p.w.WalkPoint(filepath.Join(cas.C, "test"), sendlast(c)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		select {
		case p.w.Cwd = <-c:
			p.w.Cwd.Name = cas.C
		default:
			p.t.Errorf("unable to find cwd Point (path=%q)", path)
		}
		// Actual test.
		if err := p.w.WalkPoint(path, mark(path)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		p.expectmark(p.w.Cwd.Parent, path, cas.W)
	}
	p.expectnomark()
}

// ExpectWalk TODO
func ExpectWalk(t *testing.T, cases map[string][]string) {
	p := P(t)
	defer p.Close()
	p.ExpectWalk(cases)
}

// ExpectWalkCwd TODO
func ExpectWalkCwd(t *testing.T, cases map[string]WalkCase) {
	p := P(t)
	defer p.Close()
	p.ExpectWalkCwd(cases)
}
