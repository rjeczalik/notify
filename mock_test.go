package notify

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

type Spy []Call

func (s *Spy) Watch(p string, e Event) (err error) {
	*s = append(*s, Call{T: TypeWatch, P: p, E: e})
	return
}

func (s *Spy) Rewatch(p string, old, new Event) (err error) {
	*s = append(*s, Call{T: TypeWatch, P: p, E: old, N: new})
	return
}

func (s *Spy) Unwatch(p string) (err error) {
	*s = append(*s, Call{T: TypeRewatch, P: p})
	return
}

func (s *Spy) Fanin(chan<- EventInfo, <-chan struct{}) {
	*s = append(*s, Call{T: TypeFanin})
}
