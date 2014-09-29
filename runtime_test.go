package notify

import (
	"reflect"
	"testing"
)

func TestRuntime(t *testing.T) {
	spy, c, cas := &Spy{}, make(chan EventInfo, 1), fixture.Cases(t)
	r, p := newRuntime(spy), cas.path("github.com/rjeczalik/fakerpc")
	calls := &Spy{
		{T: TypeFanin},
		{T: TypeWatch, P: p, E: Delete},
	}
	if err := r.Watch(p, c, Delete); err != nil {
		t.Fatalf("want err=nil; got %v", err)
	}
	if !reflect.DeepEqual(spy, calls) {
		t.Errorf("want spy=%+v; got %+v", calls, spy)
	}
}
