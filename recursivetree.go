package notify

// RecursiveTree TODO(rjeczalik)
type RecursiveTree struct {
	root Root
	w    RecursiveWatcher
	cnd  ChanNodesMap
}

func NewRecursiveTree(w RecursiveWatcher, c <-chan EventInfo) *RecursiveTree {
	t := &RecursiveTree{
		root: Root{nd: newnode("")},
		cnd:  make(ChanNodesMap),
		w:    w,
	}
	go t.dispatch(c)
	return t
}

func (t *RecursiveTree) dispatch(c <-chan EventInfo) {
	// TODO
}

func (t *RecursiveTree) Watch(path string, c chan<- EventInfo, events ...Event) error {
	return nil
}

func (t *RecursiveTree) Stop(c chan<- EventInfo) {

}
