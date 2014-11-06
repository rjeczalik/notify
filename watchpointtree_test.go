package notify_test

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"

	. "github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestNodeSet(t *testing.T) {
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
				t.Errorf("want nds[%d]=%v; got %v (i=%d)", j, nd, nds[j])
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

type Chans []chan<- EventInfo

func (c Chans) foreach(fn func(chan<- EventInfo, Node)) {
	for i, ch := range c {
		fn(ch, Node{Name: strconv.Itoa(i)})
	}
}

func TestChanNodesMap(t *testing.T) {
	ch := test.Chans(10)
	cases := [...]struct {
		ch  Chans
		cpt ChanNodesMap
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
		cpt := make(ChanNodesMap)
		cas.ch.foreach(cpt.Add)
		if !reflect.DeepEqual(cpt, cas.cpt) {
			t.Errorf("want cpt=%v; got %v (i=%d)", cas.cpt, cpt, i)
			continue
		}
		cas.ch.foreach(cpt.Del)
		if n := len(cpt); n != 0 {
			t.Errorf("want len(cpt)=0; got %d (i=%d)", n, i)
			continue
		}
	}
}

func TestPointSlice(t *testing.T) {
	cases := [...]struct{}{}
	for i, cas := range cases {
		_, _ = i, cas // TODO
	}
}

func TestWalkNode(t *testing.T) {
	cases := map[string][]string{
		"/tmp":                           {"tmp"},
		"/home/rjeczalik":                {"home", "rjeczalik"},
		"/":                              {},
		"/h/o/m/e/":                      {"h", "o", "m", "e"},
		"/home/rjeczalik/src/":           {"home", "rjeczalik", "src"},
		"/home/user/":                    {"home", "user"},
		"/home/rjeczalik/src/github.com": {"home", "rjeczalik", "src", "github.com"},
	}
	// Don't use filepath.VolumeName and make the following regular test-cases?
	if runtime.GOOS == "windows" {
		cases[`C:`] = []string{}
		cases[`C:\`] = []string{}
		cases[`C:\Windows\Temp`] = []string{"Windows", "Temp"}
		cases[`D:\Windows\Temp`] = []string{"Windows", "Temp"}
		cases[`F:\`] = []string{}
		cases[`\\host\share\`] = []string{}
		cases[`F:\abc`] = []string{"abc"}
		cases[`D:\abc`] = []string{"abc"}
		cases[`F:\Windows`] = []string{"Windows"}
		cases[`\\host\share\Windows\Temp`] = []string{"Windows", "Temp"}
		cases[`\\tsoh\erahs\Users\rjeczalik`] = []string{"Users", "rjeczalik"}
	}
	test.ExpectWalk(t, cases)
}

func TestWalkNodeCwd(t *testing.T) {
	cases := map[string]test.WalkCase{
		"/home/rjeczalik/src/github.com": {"/home/rjeczalik", []string{"src", "github.com"}},
		"/a/b/c/d/e/f/g/h/j/k":           {"/a/b/c/d/e/f", []string{"g", "h", "j", "k"}},
		"/tmp/a/b/c/d":                   {"/tmp/a/b", []string{"c", "d"}},
		"/tmp/a":                         {"/tmp", []string{"a"}},
		"/":                              {"", []string{}},
		"//":                             {"/", []string{}},
		"":                               {},
	}
	// Don't use filepath.VolumeName and make the following regular test-cases?
	if runtime.GOOS == "windows" {
		cases[`C:`] = test.WalkCase{}
		cases[`C:\`] = test.WalkCase{}
		cases[`C\Windows\Temp`] = test.WalkCase{C: `C:\Windows`, W: []string{"Temp"}}
		cases[`D:\Windows\Temp`] = test.WalkCase{C: `D:\Windows`, W: []string{"Temp"}}
		cases[`E:\Windows\Temp\Local`] = test.WalkCase{C: `E:\Windows`, W: []string{"Temp", "Local"}}
		cases[`\\host\share\Windows`] = test.WalkCase{C: `\\host\share`, W: []string{"Windows"}}
		cases[`\\host\share\Windows\Temp`] = test.WalkCase{C: `\\host\share\Windows`, W: []string{"Temp"}}
		cases[`\\host1\share\Windows\system32`] = test.WalkCase{C: `\\host1\share`, W: []string{"Windows", "system32"}}
	}
	test.ExpectWalkCwd(t, cases)
}

func TestWalkWatchPoint(t *testing.T) {
	// ...
}
