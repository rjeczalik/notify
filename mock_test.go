package notify

import (
	"reflect"
	"testing"

	"github.com/rjeczalik/fs"
)

// TODO(rjeczalik): fixture_test contains mocks and fixtures for Watcher tests.
// mock_test - for Runtime tests: merge them into one file.

// Type TODO
type Type string

const (
	TypeInvalid = Type("Invalid")
	TypeWatch   = Type("Watch")
	TypeRewatch = Type("Rewatch")
	TypeUnwatch = Type("Unwatch")
	TypeFanin   = Type("Fanin")
)

// TODO(rjeczalik): Merge/embed EventInfo here?
type Call struct {
	T Type
	P string
	E Event
	N Event
}

// Spy TODO
type Spy []Call

// Watch implements notify.Watcher interface.
func (s *Spy) Watch(p string, e Event) (err error) {
	*s = append(*s, Call{T: TypeWatch, P: p, E: e})
	return
}

// Rewatch implements notify.Watcher interface.
func (s *Spy) Rewatch(p string, old, new Event) (err error) {
	*s = append(*s, Call{T: TypeWatch, P: p, E: old, N: new})
	return
}

// Unwatch implements notify.Watcher interface.
func (s *Spy) Unwatch(p string) (err error) {
	*s = append(*s, Call{T: TypeRewatch, P: p})
	return
}

// Fanin implements notify.Watcher interface.
func (s *Spy) Fanin(chan<- EventInfo, <-chan struct{}) {
	*s = append(*s, Call{T: TypeFanin})
}

// Mock TODO
type Mock struct {
	fs fs.Filesystem
}

// Test TODO(rjeczalik): rename mock to mockdesc, and test to mock?
var test = Mock{fs: tree}

// CallCases TODO
type CallCases struct {
	Watch  Call   // watch call to be executed
	Expect []Call // underlying calls to Watcher captured during watch call
}

// Dummy is a dummy channel, non-nil but closed. Used for mocked calls to
// Watcher.Watch when no-one cares about getting notifications back.
var dummy = make(chan EventInfo)

func init() {
	close(dummy)
}

// ExecCall calls Watch on r passing arguments described by Call. It ignores
// value of c.Type.
func ExecCall(r *Runtime, c Call) error {
	return r.Watch(c.P, dummy, c.E)
}

// ExpectCalls TODO
func (m Mock) ExpectCalls(t *testing.T, cc []CallCases) {
	spy, n := &Spy{}, 0
	r := newRuntime(spy, m.fs)
	for _, cc := range cc {
		if err := r.Watch(cc.Watch.P, dummy, cc.Watch.E); err != nil {
			t.Fatalf("want err=nil; got %v (call=%v)", err, cc.Watch)
		}
		if cas := (*spy)[n:]; !reflect.DeepEqual(cas, Spy(cc.Expect)) {
			t.Errorf("want cas=%v; got %v (call=%v)", cc.Expect, cas, cc.Watch)
		}
		n = len(*spy)
	}
}
