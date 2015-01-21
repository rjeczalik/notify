package notify

// TODO(rjeczalik)

type recursiveTree struct {
	root Root
	w    recursiveWatcher
	cnd  ChanNodesMap
}

func newRecursiveTree(w recursiveWatcher, c <-chan EventInfo) *recursiveTree {
	t := &recursiveTree{
		root: Root{nd: newnode("")},
		cnd:  make(ChanNodesMap),
		w:    w,
	}
	go t.dispatch(c)
	return t
}

func (t *recursiveTree) dispatch(c <-chan EventInfo) {
}

func (t *recursiveTree) Watch(path string, c chan<- EventInfo, events ...Event) error {
	return nil
}

func (t *recursiveTree) Stop(c chan<- EventInfo) {

}
