package test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/rjeczalik/notify"
)

// C is just a little c, but one line it becomes c++.
var c int

// IsDir reports whether p points to a directory. A path that ends with a trailing
// path separator are assumed to be a directory.
func isDir(p string) bool {
	return strings.HasSuffix(p, sep)
}

// Nonnil TODO
func nonil(err ...error) error {
	for _, err := range err {
		if err != nil {
			return err
		}
	}
	return nil
}

// Watch TODO
func watch(w notify.Watcher, e notify.Event) filepath.WalkFunc {
	return walkfn(w, func(w notify.Watcher, p string) error { return w.Watch(p, e) })
}

// Unwatch TODO
func unwatch(w notify.Watcher) filepath.WalkFunc {
	return walkfn(w, notify.Watcher.Unwatch)
}

// Walkfn TODO
func walkfn(w notify.Watcher, fn func(notify.Watcher, string) error) filepath.WalkFunc {
	return func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			err = fn(w, p)
		}
		return err
	}
}

var eityp = reflect.TypeOf((notify.EventInfo)(nil))

type ei struct {
	p string
	f FuncType
	i int
	e notify.Event
	b bool
}

// Indexer provides an interface for assinging unique ids to elements, typically
// in an auto-increment way.
type Indexer interface {
	Index() int
}

// EI returns new EventInfo for the given path and event.
//
// In order to distinguish between a file and directory, use a trailing
// slash for the latter, like in:
//
//   - path="/var/tmp/notify", which points to a file
//   - path="/var/tmp/notify/", which points to a directory
//
// The returned EventInfo has an extra functionality - each instance holds
// an auto-incremented index. If EventInfo returned by EI is used as a map's key,
// those map keys can be sorted using SortKeys method.
//
// The arguments passed to EI can be one or more of:
//
//   - notify.EventInfo: initializes all values of the returned EventInfo
//   - notify.Event: expands the returned EventInfo event set
//   - test.FuncType: assignes an action, which will get triggered when ei is executed
//   - string: initializes path, returned via Name()
//
// The event set of the returned EventInfo gets initialized to notify.All if no
// other event is passed. EI panics when unexpected type is passed.
//
// When len(args)==0 the returned EventInfo is zero-value with auto-incremented
// index.
//
// TODO(rjeczalik): Remove altogether with w.go refactoring.
func EI(args ...interface{}) notify.EventInfo {
	e, id := ei{}, false
	for _, v := range args {
		switch v := v.(type) {
		case ei:
			e, id = v, true
		case Indexer:
			e.i, id = v.Index(), true
		case FuncType:
			e.f = v
		case notify.Event:
			e.e |= v
		case string:
			e.p = filepath.FromSlash(v)
		case notify.EventInfo:
			e.p, e.e, e.b = v.Name(), v.Event(), v.IsDir()
			if i, ok := v.(Indexer); ok {
				e.i = i.Index()
			}
		default:
			panic(fmt.Sprintf("test.EI(%+v): %T: unexpected argument type", args, v))
		}
	}
	if len(args) != 0 {
		if e.e == 0 {
			e.e = notify.All
		}
		if e.f == "" {
			e.f = Watch
		}
		e.b = isDir(e.p)
	}
	if !id {
		e.i = c
		c++
	}
	return e
}

// Implements notify.EventInfo interface.
func (e ei) Event() notify.Event { return e.e }
func (e ei) IsDir() bool         { return e.b }
func (e ei) Name() string        { return e.p }
func (e ei) Sys() interface{}    { return e.f }

// Index implements Indexer interface.
func (e ei) Index() int { return e.i }

// Eislice implements sort.Interface for notify.EventInfo slice. It expects each
// element to be implemented by a ei type.
type eislice []notify.EventInfo

func (e eislice) Len() int           { return len(e) }
func (e eislice) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e eislice) Less(i, j int) bool { return e[i].(Indexer).Index() < e[j].(Indexer).Index() }

// SortKeys sorts and returns keys of the m map. The m must be of map[notify.EventInfo]T
// type. The type implementing notify.EventInfo interface must alse implement
// an Indexer one.
//
// For the following map:
//
//   ei := map[EventInfo]int{
//     EI("p1", notify.Create): 10,
//     EI("p2", notify.Delete): 3,
//     EI("p3", notify.Write): 5,
//   }
//
// SortKeys will always return events in the following order: p1, p2 and p3.
func SortKeys(m interface{}) (key []notify.EventInfo) {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		panic(fmt.Sprintf("want m to be map; got %T", m))
	}
	for _, v := range v.MapKeys() {
		if e, ok := v.Interface().(notify.EventInfo); ok {
			if _, ok = e.(Indexer); ok {
				key = append(key, e)
				continue
			}
		}
		panic(fmt.Sprintf("want typeof(m)=map[notify.EventInfo]T; got %T", m))
	}
	if len(key) == 0 {
		return nil
	}
	sort.Stable(eislice(key))
	return
}
