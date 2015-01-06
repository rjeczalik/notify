package notify

import (
	"reflect"
	"testing"
)

func TestNodeAdd(t *testing.T) {

}

func TestTreeNodeSet(t *testing.T) {
	cases := [...]struct {
		nd  []Node
		nds NodeSet
	}{{
		[]Node{{Name: "g"}, {Name: "t"}, {Name: "u"}, {Name: "a"}, {Name: "b"}},
		NodeSet{{Name: "a"}, {Name: "b"}, {Name: "g"}, {Name: "t"}, {Name: "u"}},
	}, {
		[]Node{{Name: "aA"}, {Name: "aA"}, {Name: "aa"}, {Name: "AA"}},
		NodeSet{{Name: "AA"}, {Name: "aA"}, {Name: "aa"}},
	}, {
		[]Node{{Name: "b"}, {Name: "b"}, {Name: "a"}, {Name: "Y"}, {Name: ""}, {Name: "a"}},
		NodeSet{{Name: ""}, {Name: "Y"}, {Name: "a"}, {Name: "b"}},
	}}
Test:
	for i, cas := range cases {
		nds := NodeSet{}
		for _, nd := range cas.nd {
			nds.Add(nd)
		}
		if !reflect.DeepEqual(nds, cas.nds) {
			t.Errorf("want nds=%v; got %v (i=%d)", cas.nds, nds, i)
			continue Test
		}
		for _, nd := range cas.nd {
			if j := nds.Search(nd); nds[j].Name != nd.Name {
				t.Errorf("want nds[%d]=%v; got %v (i=%d)", j, nd, nds[j], i)
				continue Test
			}
		}
		for _, nd := range cas.nd {
			nds.Del(nd)
		}
		if n := len(nds); n != 0 {
			t.Errorf("want len(nds)=0; got %d (i=%d)", n, i)
			continue Test
		}
	}
}

func TestTreeChanNodesMap(t *testing.T) {
	ch := NewChans(10)
	cases := [...]struct {
		ch  Chans
		cnd ChanNodesMap
	}{{
		Chans{ch[0]},
		ChanNodesMap{ch[0]: {{Name: "0"}}},
	}, {
		Chans{ch[0], ch[0], ch[0]},
		ChanNodesMap{
			ch[0]: {{Name: "0"}, {Name: "1"}, {Name: "2"}},
		},
	}, {
		Chans{ch[0], ch[3], ch[2], ch[1]},
		ChanNodesMap{
			ch[0]: {{Name: "0"}},
			ch[1]: {{Name: "3"}},
			ch[2]: {{Name: "2"}},
			ch[3]: {{Name: "1"}},
		},
	}, {
		Chans{ch[0], ch[0], ch[2], ch[1], ch[3], ch[3], ch[2], ch[2], ch[4], ch[0]},
		ChanNodesMap{
			ch[0]: {{Name: "0"}, {Name: "1"}, {Name: "9"}},
			ch[1]: {{Name: "3"}},
			ch[2]: {{Name: "2"}, {Name: "6"}, {Name: "7"}},
			ch[3]: {{Name: "4"}, {Name: "5"}},
			ch[4]: {{Name: "8"}},
		},
	}}
	for i, cas := range cases {
		cnd := make(ChanNodesMap)
		cas.ch.Foreach(cnd.Add)
		if !reflect.DeepEqual(cnd, cas.cnd) {
			t.Errorf("want cnd=%v; got %v (i=%d)", cas.cnd, cnd, i)
			continue
		}
		cas.ch.Foreach(cnd.Del)
		if n := len(cnd); n != 0 {
			t.Errorf("want len(cnd)=0; got %d (i=%d)", n, i)
			continue
		}
	}
}
