package test

import (
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
	Stop             = FuncType("Stop")
)

// Call represents single call to notify.Watcher issued by the notify.Runtime
// and recorded by a spy notify.Watcher mock.
//
// TODO(rjeczalik): Merge/embed notify.EventInfo here?
type Call struct {
	F FuncType
	C chan<- notify.EventInfo
	P string
	E notify.Event // regular Event argument and old Event from a Rewatch call
	N notify.Event // new Event argument from a Rewatch call
}

// CallCase describes a single test case. Call describes either Watch or Stop
// invocation made on a Runtime, while Record describes all internal calls
// to the Watcher implementation made by Runtime during that call.
type CallCase struct {
	Call   Call   // call invoked on Runtime
	Record []Call // intermediate calls recorded during above call
}

// RuntimeFunc is used for initializing spy mock.
type RuntimeFunc func(*Spy) notify.Watcher

// Watcher is a RuntimeFunc returning a Watcher which implements notify.Watcher
// interface only.
func RuntimeWatcher(spy *Spy) notify.Watcher {
	return struct{ notify.Watcher }{spy}
}

// RuntimeRewatcher is a RuntimeFunc returning a Watcher which complementary to
// notify.Watcher implements also notify.Rewatcher interface.
func RuntimeRewatcher(spy *Spy) notify.Watcher {
	return struct {
		notify.Watcher
		notify.Rewatcher
	}{spy, spy}
}

// RuntimeInterface is a default RuntimeFunc which returns a Watcher which implements
// notify.RuntimeInterface.
func RuntimeInterface(spy *Spy) notify.Watcher {
	return struct{ notify.Interface }{spy}
}

// R represents a fixture for notify.Runtime testing.
type r struct {
	t   *testing.T
	r   *notify.Runtime
	spy Spy
	n   int
}

// Channels is a utility function which creates pool of chan <- notify.EventInfo.
func Channels(n int) func(int) chan<- notify.EventInfo {
	ch := make([]chan notify.EventInfo, n)
	for i := range ch {
		ch[i] = make(chan notify.EventInfo)
	}
	return func(i int) chan<- notify.EventInfo {
		return ch[i]
	}
}

// R gives new fixture for notify.Runtime testing. It initializes a new Runtime
// with a spy Watcher mock, which is used for retrospecting calls to it the Runtime
// makes.
func R(t *testing.T, fn RuntimeFunc) *r {
	r := &r{t: t, n: 1}
	r.r = notify.NewRuntimeWatcher(fn(&r.spy), FS)
	return r
}

func (r *r) invoke(call Call) error {
	switch call.F {
	case Watch:
		return r.r.Watch(call.P, call.C, call.E)
	case Stop:
		r.r.Stop(call.C)
		return nil
	}
	panic("test.invoke: invalid Runtime call: " + call.F)
}

// ExpectCalls translates values specified by the cases' keys into Watch calls
// executed on the fixture's Runtime. A spy Watcher mock records calls to it
// and compares with the expected ones for each key in the list.
func (r *r) ExpectCalls(cases []CallCase) {
	for _, cas := range cases {
		if err := r.invoke(cas.Call); err != nil {
			r.t.Fatalf("want Runtime.%s(...)=nil; got %v (call=%+v)", cas.Call.F,
				err, cas.Call)
		}
		if got := r.spy[r.n:]; (len(cas.Record) != 0 || len(got) != 0) && !reflect.DeepEqual(got, Spy(cas.Record)) {
			r.t.Errorf("want recorded=%+v; got %+v (call=%+v)", cas.Record, got,
				cas.Call)
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
func ExpectCallsFunc(t *testing.T, fn RuntimeFunc, cases []CallCase) {
	R(t, fn).ExpectCalls(cases)
}

// ExpectCallsFunc translates cases' keys into (*Runtime).Watch calls, records calls
// Runtime makes to a Watcher and compares them with the expected list.
//
// Eventhough cases is described by a map, events are executed in the
// order they were either defined or assigned to the cases.
func ExpectCalls(t *testing.T, cases []CallCase) {
	ExpectCallsFunc(t, RuntimeInterface, cases)
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
