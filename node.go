package notify

import (
	"path/filepath"
	"sort"
)

// Node TODO
type Node struct {
	Name  string
	Watch Watchpoint
	Child map[string]Node
}

// NewNode TODO
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

// NodeSet TODO
type NodeSet []Node

func (p NodeSet) Len() int           { return len(p) }
func (p NodeSet) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p NodeSet) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (p NodeSet) Search(nd Node) int {
	return sort.Search(len(p), func(i int) bool { return p[i].Name >= nd.Name })
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
