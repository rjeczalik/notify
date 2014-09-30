package test

import (
	"fmt"
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

var eityp = reflect.TypeOf((notify.EventInfo)(nil))

type ei struct {
	p string
	i int
	e notify.Event
	b bool
}

// EI returns new EventInfo for the given path and event.
//
// The returned EventInfo has an extra functionality - each instance holds
// an auto-incremented ID. If EventInfo returned by EI is used as a map's key,
// those map keys can be sorted using SortKeys method.
func EI(path string, event notify.Event) notify.EventInfo {
	ei := ei{
		p: path,
		i: c,
		e: event,
		b: isDir(path),
	}
	c++
	return ei
}

// Implements notify.EventInfo interface.
func (e ei) Event() notify.Event { return e.e }
func (e ei) IsDir() bool         { return e.b }
func (e ei) Name() string        { return e.p }
func (e ei) Sys() interface{}    { return nil }

// Eislice implements sort.Interface for notify.EventInfo slice. It expects each
// element to be implemented by a ei type.
type eislice []notify.EventInfo

func (e eislice) Len() int           { return len(e) }
func (e eislice) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }
func (e eislice) Less(i, j int) bool { return e[i].(ei).i < e[i].(ei).i }

// SortKeys sorts and returns keys of the m map. The m must be of map[notify.EventInfo]T
// type. The notify.EventInfo interface must be implemented by ei type.
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
			if ei, ok := e.(ei); ok {
				key = append(key, ei)
				continue
			}
		}
		panic(fmt.Sprintf("want typeof(m)=map[notify.EventInfo]T; got %T", m))
	}
	sort.Sort(eislice(key))
	return
}
