package notify

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
)

func min(i, j int) int {
	if i > j {
		return j
	}
	return i
}

// Skip TODO
var Skip = errors.New("skip")

// WalkPathFunc TODO
type WalkPathFunc func(nd Node, isbase bool) error

// WalkFunc TODO
type WalkFunc func(Node) error

func errnotexist(name string) error {
	return &os.PathError{
		Op:   "Node",
		Path: name,
		Err:  os.ErrNotExist,
	}
}

// Node TODO
type Node struct {
	Name  string
	Watch Watchpoint
	Child map[string]Node
}

// child TODO
func (nd Node) child(name string) Node {
	if name == "" {
		return nd
	}
	if child, ok := nd.Child[name]; ok {
		return child
	}
	child := Node{
		Name:  nd.Name + sep + name,
		Watch: make(Watchpoint),
		Child: make(map[string]Node),
	}
	// TODO(rjeczalik): Fix it better.
	if name == filepath.VolumeName(name) {
		child.Name = name
	}
	nd.Child[name] = child
	return child
}

func newnode(name string) Node {
	return Node{
		Name:  name,
		Watch: make(Watchpoint),      // TODO lazy alloc?
		Child: make(map[string]Node), // TODO lazy alloc?
	}
}

// TODO(rjeczalik): split unix + windows
func base(root, name string) int {
	if n, m := len(root), len(name); m >= n && name[:n] == root &&
		(n == m || name[n] == os.PathSeparator) {
		return min(n+1, m)
	}
	return -1
}

func (nd Node) addchild(name, base string) Node {
	child, ok := nd.Child[base]
	if !ok {
		child = newnode(name)
		nd.Child[base] = child
	}
	return child
}

// Add TODO
func (nd Node) Add(name string) Node {
	i := base(nd.Name, name)
	if i == -1 {
		return Node{}
	}
	for j := IndexSep(name[i:]); j != -1; j = IndexSep(name[i:]) {
		nd = nd.addchild(name[:i+j], name[i:i+j])
		i += j + 1
	}
	return nd.addchild(name, name[i:])
}

// AddDir TODO
func (nd Node) AddDir(dir string, fn WalkFunc) error {
	nd = nd.Add(dir)
	if nd.Child == nil { // TODO(rjeczalik): add IsZero
		return errnotexist(dir)
	}
	stack := []Node{nd}
Traverse:
	for n := len(stack); n != 0; n = len(stack) {
		nd, stack = stack[n-1], stack[:n-1]
		switch err := fn(nd); err {
		case nil:
		case Skip:
			continue Traverse
		default:
			return err
		}
		// TODO(rjeczalik): tolerate open failures - add failed names to
		// AddDirError and notify users which names are not added to the tree.
		f, err := os.Open(nd.Name)
		if err != nil {
			return err
		}
		fi, err := f.Readdir(0)
		f.Close()
		if err != nil {
			return err
		}
		for _, fi := range fi {
			if fi.IsDir() {
				name := filepath.Join(nd.Name, fi.Name())
				stack = append(stack, nd.addchild(name, name[len(nd.Name)+1:]))
			}
		}
	}
	return nil
}

// Get TODO
func (nd Node) Get(name string) (Node, error) {
	i := base(nd.Name, name)
	if i == -1 {
		return Node{}, errnotexist(name)
	}
	ok := false
	for j := IndexSep(name[i:]); j != -1; j = IndexSep(name[i:]) {
		if nd, ok = nd.Child[name[i:i+j]]; !ok {
			return Node{}, errnotexist(name)
		}
		i += j + 1
	}
	if nd, ok = nd.Child[name[i:]]; !ok {
		return Node{}, errnotexist(name)
	}
	return nd, nil
}

// Del TODO
func (nd Node) Del(name string) error {
	i := base(nd.Name, name)
	if i == -1 {
		return errnotexist(name)
	}
	stack := []Node{nd}
	ok := false
	for j := IndexSep(name[i:]); j != -1; j = IndexSep(name[i:]) {
		if nd, ok = nd.Child[name[i:i+j]]; !ok {
			return errnotexist(name[:i+j])
		}
		stack = append(stack, nd)
	}
	if nd, ok = nd.Child[name[i:]]; !ok {
		return errnotexist(name)
	}
	nd.Child = nil
	nd.Watch = nil
	for name, i = Base(nd.Name), len(stack); i != 0; name, i = Base(nd.Name), i-1 {
		nd = stack[i-1]
		if nd := nd.Child[name]; len(nd.Watch) > 1 || len(nd.Child) != 0 {
			break
		} else {
			nd.Child = nil
			nd.Watch = nil
		}
		delete(nd.Child, name)
	}
	return nil
}

// Walk TODO
func (nd Node) Walk(fn WalkFunc) error {
	stack := []Node{nd}
Traverse:
	for n := len(stack); n != 0; n = len(stack) {
		nd, stack = stack[n-1], stack[:n-1]
		switch err := fn(nd); err {
		case nil:
		case Skip:
			continue Traverse
		default:
			return err
		}
		for _, nd = range nd.Child {
			stack = append(stack, nd)
		}
	}
	return nil
}

// WalkPath TODO
func (nd Node) WalkPath(name string, fn WalkPathFunc) error {
	i := base(nd.Name, name)
	if i == -1 {
		return errnotexist(name)
	}
	ok := false
	for j := IndexSep(name[i:]); j != -1; j = IndexSep(name[i:]) {
		if err := fn(nd, false); err != nil && err != Skip {
			return err
		}
		if nd, ok = nd.Child[name[i:i+j]]; !ok {
			return errnotexist(name[:i+j])
		}
		i += j + 1
	}
	if nd, ok = nd.Child[name[i:]]; !ok {
		return errnotexist(name)
	}
	if err := fn(nd, true); err != nil && err != Skip {
		return err
	}
	return nil
}

// Root TODO
type Root struct {
	nd Node
}

// TODO(rjeczalik): split unix + windows
func (r Root) addroot(name string) Node {
	if vol := filepath.VolumeName(name); vol != "" {
		root, ok := r.nd.Child[vol]
		if !ok {
			root = r.nd.addchild(vol, vol)
		}
		return root
	}
	return r.nd
}

// TODO(rjeczalik): split unix + windows
func (r Root) root(name string) (Node, error) {
	if vol := filepath.VolumeName(name); vol != "" {
		nd, ok := r.nd.Child[vol]
		if !ok {
			return Node{}, errnotexist(name)
		}
		return nd, nil
	}
	return r.nd, nil
}

// Add TODO
func (r Root) Add(name string) Node {
	return r.addroot(name).Add(name)
}

// WalkDir TODO
func (r Root) AddDir(dir string, fn WalkFunc) error {
	return r.addroot(dir).AddDir(dir, fn)
}

// Del TODO
func (r Root) Del(name string) error {
	nd, err := r.root(name)
	if err != nil {
		return err
	}
	return nd.Del(name)
}

// Get TODO
func (r Root) Get(name string) (Node, error) {
	nd, err := r.root(name)
	if err != nil {
		return Node{}, err
	}
	return nd.Get(name)
}

// Walk TODO
func (r Root) Walk(name string, fn WalkFunc) error {
	nd, err := r.root(name)
	if err != nil {
		return err
	}
	if nd, err = nd.Get(name); err != nil {
		return err
	}
	return nd.Walk(fn)
}

// Root TODO
func (r Root) WalkPath(name string, fn WalkPathFunc) error {
	nd, err := r.root(name)
	if err != nil {
		return err
	}
	return nd.WalkPath(name, fn)
}

// NodeSet TODO
type NodeSet []Node

func (p NodeSet) Len() int           { return len(p) }
func (p NodeSet) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p NodeSet) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p NodeSet) Search(nd Node) int {
	return sort.Search(len(p), func(i int) bool { return p[i].Name >= nd.Name })
}

func (p *NodeSet) Names() (s []string) {
	for i := range *p {
		s = append(s, (*p)[i].Name)
	}
	return
}

func (p *NodeSet) Add(nd Node) {
	switch i := p.Search(nd); {
	case i == len(*p):
		*p = append(*p, nd)
	case (*p)[i].Name == nd.Name:
	default:
		*p = append(*p, Node{})
		copy((*p)[i+1:], (*p)[i:])
		(*p)[i] = nd
	}
}

func (p *NodeSet) Del(nd Node) {
	if i, n := p.Search(nd), len(*p); i != n && (*p)[i].Name == nd.Name {
		copy((*p)[i:], (*p)[i+1:])
		*p = (*p)[:n-1]
	}
}

// ChanNodesMap TODO
type ChanNodesMap map[chan<- EventInfo]*NodeSet

func (m ChanNodesMap) Add(c chan<- EventInfo, nd Node) {
	if nds, ok := m[c]; ok {
		nds.Add(nd)
	} else {
		m[c] = &NodeSet{nd}
	}
}

func (m ChanNodesMap) Del(c chan<- EventInfo, nd Node) {
	if nds, ok := m[c]; ok {
		if nds.Del(nd); len(*nds) == 0 {
			delete(m, c)
		}
	}
}
