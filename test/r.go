package test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/rjeczalik/notify"
)

// Dummy is a dummy channel, non-nil but closed. Used for mocked calls to
// Watcher.Watch when no-one cares about getting notifications back.
var dummy chan<- notify.EventInfo = make(chan notify.EventInfo)

func init() {
	close(dummy)
}

// FuncType represents enums for notify.Watcher interface.
type FuncType string

const (
	Watch            = FuncType("Watch")
	Unwatch          = FuncType("Unwatch")
	Fanin            = FuncType("Fanin")
	Rewatch          = FuncType("Rewatch")
	RecursiveWatch   = FuncType("RecursiveWatch")
	RecursiveUnwatch = FuncType("RecursiveUnwatch")
)

// Call represents single call to notify.Watcher issued by the notify.Runtime
// and recorded by a spy notify.Watcher mock.
//
// TODO(rjeczalik): Merge/embed notify.EventInfo here?
type Call struct {
	F FuncType
	P string
	E notify.Event
	N notify.Event
}

type pe struct {
	p string
	e notify.Event
}

// WatcherFunc is used for initializing spy mock.
type WatcherFunc func(*Spy) notify.Watcher

// Interface is a default WatcherFunc which returns a Watcher which implements
// notify.Interface.
func Interface(spy *Spy) notify.Watcher {
	return struct{ notify.Interface }{spy}
}

// Watcher is a WatcherFunc returning a Watcher which implements notify.Watcher
// interface only.
func Watcher(spy *Spy) notify.Watcher {
	return struct{ notify.Watcher }{spy}
}

// Rewatcher is a WatcherFunc returning a Watcher which complementary to notify.Watcher
// implements also notify.RecursiveWatcher interface.
func Rewatcher(spy *Spy) notify.Watcher {
	return struct {
		notify.Watcher
		notify.Rewatcher
	}{spy, spy}
}

// RecursiveWatcher is a WatcherFunc returning a Watcher which complementary to
// notify.Watcher implements also notify.RecursiveWatcher interface.
func RecursiveWatcher(spy *Spy) notify.Watcher {
	return struct {
		notify.Watcher
		notify.RecursiveWatcher
	}{spy, spy}
}

// R represents a fixture for notify.Runtime testing.
type r struct {
	c   map[pe]chan<- notify.EventInfo
	t   *testing.T
	r   *notify.Runtime
	spy Spy
	n   int
}

// R gives new fixture for notify.Runtime testing. It initializes a new Runtime
// with a spy Watcher mock, which is used for retrospecting calls to it the Runtime
// makes.
func R(t *testing.T, fn WatcherFunc) *r {
	r := &r{c: make(map[pe]chan<- notify.EventInfo), t: t, n: 1}
	r.r = notify.NewRuntimeWatcher(fn(&r.spy), FS)
	return r
}

func (r *r) exec(ei notify.EventInfo) (err error) {
	pe := pe{p: ei.Name(), e: ei.Event()}
	if f, ok := ei.Sys().(FuncType); ok && f == Unwatch {
		c, ok := r.c[pe]
		if !ok {
			panic(fmt.Sprintf("notify/test: unexpected Stop of channel pe=%+v", pe))
		}
		r.r.Stop(c)
		return nil
	}
	c, ok := r.c[pe]
	if !ok {
		c = make(chan notify.EventInfo)
		r.c[pe] = c
	}
	return r.r.Watch(pe.p, c, pe.e)
}

// ExpectCalls translates values specified by the cases' keys into Watch calls
// executed on the fixture's Runtime. A spy Watcher mock records calls to it
// and compares with the expected ones for each key in the map.
//
// Eventhough cases is described by a map, events are executed in the
// order they were either defined or assigned to the cases.
func (r *r) ExpectCalls(cases map[notify.EventInfo][]Call) {
	// Sort keys to ensure cases are executed in chronological order.
	for _, ei := range SortKeys(cases) {
		if err := r.exec(ei); err != nil {
			r.t.Fatalf("want err=nil; got %v (ei=%v)", err, ei)
		}
		if want, got := cases[ei], r.spy[r.n:]; !reflect.DeepEqual(got, Spy(want)) {
			r.t.Errorf("want cas=%+v; got %+v (ei=%+v)", want, got, ei)
		}
		r.n = len(r.spy)
	}
}

// ExpectCallsFunc translates cases' keys into (*Runtime).Watch calls, records calls
// Runtime makes to a Watcher and compares them with the expected list.
//
// It uses fn to limit number of interfaces which Spy is going to implement.
// A spy is a recording mock used as a Watcher implementation to initialize
// a Runtime.
//
// Eventhough cases is described by a map, events are executed in the
// order they were either defined or assigned to the cases.
func ExpectCallsFunc(t *testing.T, fn WatcherFunc, ei map[notify.EventInfo][]Call) {
	R(t, fn).ExpectCalls(ei)
}

// ExpectCallsFunc translates cases' keys into (*Runtime).Watch calls, records calls
// Runtime makes to a Watcher and compares them with the expected list.
//
// Eventhough cases is described by a map, events are executed in the
// order they were either defined or assigned to the cases.
func ExpectCalls(t *testing.T, ei map[notify.EventInfo][]Call) {
	ExpectCallsFunc(t, Interface, ei)
}

// Spy is a mock for notify.Watcher interface, which records every call.
type Spy []Call

// Watch implements notify.Watcher interface.
func (s *Spy) Watch(p string, e notify.Event) (err error) {
	*s = append(*s, Call{F: Watch, P: p, E: e})
	return
}

// Unwatch implements notify.Watcher interface.
func (s *Spy) Unwatch(p string) (err error) {
	*s = append(*s, Call{F: Unwatch, P: p})
	return
}

// Fanin implements notify.Watcher interface.
func (s *Spy) Fanin(chan<- notify.EventInfo, <-chan struct{}) {
	*s = append(*s, Call{F: Fanin})
}

// Rewatch implements notify.Rewatcher interface.
func (s *Spy) Rewatch(p string, old, new notify.Event) (err error) {
	*s = append(*s, Call{F: Rewatch, P: p, E: old, N: new})
	return
}

// RecursiveWatch implements notify.RecursiveWatcher interface.
func (s *Spy) RecursiveWatch(p string, e notify.Event) (err error) {
	*s = append(*s, Call{F: RecursiveWatch, P: p, E: e})
	return
}

// RecursiveUnwatch implements notify.RecursiveWatcher interface.
func (s *Spy) RecursiveUnwatch(p string) (err error) {
	*s = append(*s, Call{F: RecursiveUnwatch, P: p})
	return
}
