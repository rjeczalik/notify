package test

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/rjeczalik/notify"
)

type parent string
type base string

func mark(s string) notify.WalkPathFunc {
	return func(nd notify.Node, isbase bool) (err error) {
		nd.Parent[""] = parent(s)
		if isbase {
			dir, ok := nd.Value().(map[string]interface{})
			if !ok {
				dir = make(map[string]interface{})
				nd.Set(dir)
			}
			dir[""] = base(s)
		}
		return
	}
}

func markandrec(s string, spy map[string]struct{}) notify.WalkNodeFunc {
	return func(nd notify.Node, p string) error {
		nd.Parent[""] = parent(s)
		if _, ok := spy[p]; ok {
			return errors.New("duplicate path: " + p)
		}
		spy[p] = struct{}{}
		return nil
	}
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

func (p *p) expectmark(mark string, dirs []string) {
	it := p.w.Root
	// TODO(rjeczalik): Remove duplicated code.
	v, ok := it[""]
	if !ok {
		p.t.Errorf("dir has no mark (mark=%q)", mark)
	}
	typ := reflect.TypeOf(parent(""))
	if len(dirs) == 0 {
		typ = reflect.TypeOf(base(""))
	}
	if got := reflect.TypeOf(v); got != typ {
		p.t.Errorf("want typeof(v)=%v; got %v (mark=%q)", typ, got, mark)
	}
	if reflect.ValueOf(v).String() != mark {
		p.t.Errorf("want v=%v; got %v (mark=%q)", mark, v, mark)
	}
	delete(it, "")
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
		typ := reflect.TypeOf(parent(""))
		if i == len(dirs)-1 {
			typ = reflect.TypeOf(base(""))
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

func (p *p) expectmarkpaths(mark string, paths map[string]struct{}) {
	bases := notify.NodeSet{}
	for path := range paths {
		err := p.w.WalkPath(path, func(nd notify.Node, isbase bool) error {
			if isbase {
				v, ok := nd.Parent[""].(parent)
				if !ok {
					return fmt.Errorf("dir has no mark (mark=%q, path=%q)",
						mark, path)
				}
				if string(v) != mark {
					return fmt.Errorf("want v=%v; got %v (mark=%q, path=%q)",
						mark, v, mark, path)
				}
				bases.Add(nd)
			}
			return nil
		})
		if err != nil {
			p.t.Error(err)
		}
	}
	for _, nd := range bases {
		delete(nd.Parent, "")
	}
}

// Test for dangling marks - if a mark is present, WalkPoint went somewhere
// it shouldn't.
func (p *p) expectnomark() error {
	return p.w.BFS(func(v interface{}) error {
		switch v.(type) {
		case map[string]interface{}:
			return nil
		default:
			return fmt.Errorf("want typeof(...)=map[string]interface{}; got %T=%v", v, v)
		}
	})
}

// PathCase TODO
type PathCase map[string][]string

// ExpectPath TODO
//
// For each test-case we're traversing path specified by a testcase's key
// over shared WatchPointTree and marking each directory using special empty
// key. The mark is simply the traversed path name. Each mark can be either
// of `parent` or `base` type. Only the base item in the path is marked with
// a `base` mark.
func (p *p) ExpectPath(cases PathCase) {
	SlashCases(cases)
	for path, dirs := range cases {
		if err := p.w.MakePath(path, mark(path)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		p.expectmark(path, dirs)
		if err := p.w.WalkPath(path, mark(path)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		p.expectmark(path, dirs)
	}
	if err := p.expectnomark(); err != nil {
		p.t.Error(err)
	}
}

// TreeCase TODO
type TreeCase map[string]map[string]struct{}

// ExpectTree TODO
func (p *p) ExpectTree(cases TreeCase) {
	SlashCases(cases)
	for path, dirs := range cases {
		spy := make(map[string]struct{}, len(dirs))
		if err := p.w.MakeTree(path, markandrec(path, spy)); err != nil {
			p.t.Errorf("want err=nil; got %v (path=%q)", err, path)
			continue
		}
		if !reflect.DeepEqual(spy, dirs) {
			p.t.Errorf("want spy=%v; got %v (path=%q)", dirs, spy, path)
			continue
		}
		p.expectmarkpaths(path, dirs)
	}
	if err := p.expectnomark(); err != nil {
		p.t.Error(err)
	}
}

// ExpectPath TODO
func ExpectPath(t *testing.T, cases PathCase) {
	p := P(t)
	defer p.Close()
	p.ExpectPath(cases)
}

// ExpectTree TODO
func ExpectTree(t *testing.T, cases TreeCase) {
	p := P(t)
	defer p.Close()
	p.ExpectTree(cases)
}
