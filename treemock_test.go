package notify

import "strconv"

type Chans []chan EventInfo

func (c Chans) Foreach(fn func(chan<- EventInfo, Node)) {
	for i, ch := range c {
		fn(ch, Node{Name: strconv.Itoa(i)})
	}
}

func NewChans(n int) Chans {
	ch := make([]chan EventInfo, n)
	for i := range ch {
		ch[i] = make(chan EventInfo, 16)
	}
	return ch
}
