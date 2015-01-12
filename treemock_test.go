package notify

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const buffer = 16

// TreeType TODO
type TreeType uint8

const (
	TreeFakeRecursive TreeType = 1 << iota // fakes RecursiveWatcher
	TreeNativeRecursive
)

// TreeAll TODO
//
// NOTE(rjeczalik): For use only as a key of Record.
const TreeAll TreeType = TreeFakeRecursive | TreeNativeRecursive

var TreeTypes = [...]TreeType{TreeFakeRecursive, TreeNativeRecursive}

// Strings implements fmt.Stringer interface.
func (typ TreeType) String() string {
	switch typ {
	case TreeFakeRecursive:
		return "TreeFakeRecursive"
	case TreeNativeRecursive:
		return "TreeNativeRecursive"
	}
	return "<unknown tree type>"
}

// IsDir reports whether p points to a directory. A path that ends with a trailing
// path separator is assumed to be a directory.
func IsDir(p string) bool {
	return strings.HasSuffix(p, sep) || strings.HasSuffix(p, sep+"...")
}

// TreeEvent TODO
type TreeEvent struct {
	P string
	E Event
}

// TreeEvent implements EventInfo interface.
func (e TreeEvent) Event() Event     { return e.E }
func (e TreeEvent) Path() string     { return e.P }
func (e TreeEvent) String() string   { return e.E.String() + " - " + e.P }
func (e TreeEvent) Sys() interface{} { return nil }

// Record TODO
type Record map[TreeType][]Call

// CallCase describes a single test case. Call describes either Watch or Stop
// invocation made on a Tree, while Record describes all internal calls
// to the Watcher implementation made by Tree during that call.
type CallCase struct {
	Call   Call   // call invoked on Tree
	Record Record // intermediate calls recorded during above call
}

// NativeCallCases TODO
func NativeCallCases(cases []CallCase) []CallCase {
	if runtime.GOOS == "windows" {
		for i := range cases {
			cases[i].Call.P = filepath.FromSlash(cases[i].Call.P)
			for _, calls := range cases[i].Record {
				for i := range calls {
					calls[i].P = filepath.FromSlash(calls[i].P)
				}
			}
		}
	}
	return cases
}

// EventCase TODO
type EventCase struct {
	Event    TreeEvent
	Receiver []chan EventInfo // receiver only
}

// NativeEventCases is a test function which takes cases as an argument.
func NativeEventCases(cases []EventCase) []EventCase {
	if runtime.GOOS == "windows" {
		for i := range cases {
			cases[i].Event.P = filepath.FromSlash(cases[i].Event.P)
		}
	}
	return cases
}

// MockedTree TODO
type MockedTree struct {
	Spy                      // implements Watcher, RecursiveWatcher or RecursiveRewatcher
	BigTree *BigTree         // actual tree being tested
	N       int              // call start offset
	C       chan<- EventInfo // event dispatch channel
}

// Invoke TODO
func (mt *MockedTree) Invoke(call Call) error {
	switch call.F {
	case FuncWatch:
		return mt.BigTree.Watch(call.P, call.C, call.E)
	case FuncStop:
		mt.BigTree.Stop(call.C)
		return nil
	}
	panic("(*TreeFixture).invoke: invalid Tree call: " + call.F)
}

// TreeFixture represents a fixture for Tree testing.
type TreeFixture map[TreeType]*MockedTree

// TreeFixture gives new fixture for Tree testing. It initializes a new Tree
// with a spy Watcher mock, which is used for retrospecting calls to it the Tree
// makes.
func NewTreeFixture() (tf TreeFixture) {
	tf = make(map[TreeType]*MockedTree)
	for _, typ := range TreeTypes {
		// TODO(rjeczalik): Copy FS to allow for modying tree via Create and
		// Delete events.
		c := make(chan EventInfo, 128)
		mt := &MockedTree{C: c}
		tf[typ] = mt
		mt.BigTree = NewTree(SpyWatcher(typ, mt), c)
		mt.BigTree.FS = MFS
	}
	return
}

// ExpectCalls translates values specified by the cases' keys into Watch calls
// executed on the fixture's Tree. A spy Watcher mock records calls to it
// and compares with the expected ones for each key in the list.
func (tf TreeFixture) TestCalls(t *testing.T, cases []CallCase) {
	cs := NativeCallCases(cases)
	for typ, tree := range tf {
		dbg.Printf("[TREEFIXTURE] TestCalls (len(cases)=%d, typ=%v)",
			len(cs), typ)
		for i, cas := range cs {
			var record []Call
			// Invoke call and record underlying calls.
			if err := tree.Invoke(cas.Call); err != nil {
				t.Fatalf("want Tree.%s(...)=nil; got %v (i=%d, typ=%v)",
					cas.Call.F, err, i, typ)
			}
			// Skip if expected and got records were empty.
			got := tree.Spy[tree.N:]
			if len(cas.Record) == 0 && len(got) == 0 {
				continue
			}
			// Find expected records for typ Tree.
			if rec, ok := cas.Record[typ]; ok {
				record = rec
			} else {
				for rectyp, rec := range cas.Record {
					if rectyp&typ != 0 {
						record = rec
						break
					}
				}
			}
			tree.N = len(tree.Spy)
			if len(got) == 0 && len(record) == 0 {
				continue
			}
			if !reflect.DeepEqual(got, Spy(record)) {
				t.Errorf("want recorded=%+v; got %+v (i=%d, typ=%v)",
					record, got, i, typ)
			}
		}
	}
}

var timeout = 50 * time.Millisecond

func str(ei EventInfo) string {
	if ei == nil {
		return "<nil>"
	}
	return fmt.Sprintf("{Name()=%q, Event()=%v}", ei.Path(), ei.Event())
}

func equal(want, got EventInfo) error {
	if (want == nil && got != nil) || (want != nil && got == nil) {
		return fmt.Errorf("want EventInfo=%s; got %s", str(want), str(got))
	}
	if want.Path() != got.Path() || want.Event() != got.Event() {
		return fmt.Errorf("want EventInfo=%s; got %s", str(want), str(got))
	}
	return nil
}

// TestEvents TODO
func (tf TreeFixture) TestEvents(t *testing.T, cases []EventCase) {
	cs := NativeEventCases(cases)
	for typ, tree := range tf {
		dbg.Printf("[TREEFIXTURE] TestEvent (len(cases)=%d, typ=%v)",
			len(cases), typ)
		for i, cas := range cs {
			n := len(cas.Receiver)
			done := make(chan struct{})
			ev := make(map[<-chan EventInfo]EventInfo)
			go func() {
				cases := make([]reflect.SelectCase, n)
				for i := range cases {
					cases[i].Chan = reflect.ValueOf(cas.Receiver[i])
					cases[i].Dir = reflect.SelectRecv
				}
				for n := len(cases); n > 0; n = len(cases) {
					j, v, ok := reflect.Select(cases)
					if !ok {
						t.Fatalf("notify/test: unexpected chan close (i=%d, "+
							"typ=%v, j=%d", i, typ, j)
					}
					ch := cases[j].Chan.Interface().(chan EventInfo)
					ev[ch] = v.Interface().(EventInfo)
					cases[j], cases = cases[n-1], cases[:n-1]
				}
				close(done)
			}()
			tree.C <- cas.Event
			select {
			case <-done:
			case <-time.After(timeout):
				t.Fatalf("ExpectEvents has timed out after %v (i=%d, typ=%v)",
					timeout, i, typ)
			}
			for _, got := range ev {
				if err := equal(cas.Event, got); err != nil {
					t.Errorf("want err=nil; got %v (i=%d, typ=%v)", err, i, typ)
				}
			}
		}
	}
}

// SpyWatcher TODO
func SpyWatcher(typ TreeType, tree *MockedTree) Watcher {
	switch typ {
	case TreeFakeRecursive:
		return struct {
			Watcher
		}{tree}
	case TreeNativeRecursive:
		return struct {
			Watcher
			RecursiveWatcher
		}{tree, tree}
	}
	panic(fmt.Sprintf("notify/test: unsupported runtime type: %d (%s)", typ, typ))
}
