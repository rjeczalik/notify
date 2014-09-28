// +build linux,!fsnotify

package notify

import (
	"io"
	"os"
)

// TODO(pknap)
var madev = map[Event][]Event{
	Create:    {Create},
	Delete:    {Delete},
	Write:     {Write},
	Move:      {Move},
	IN_ACCESS: {IN_OPEN, IN_MODIFY, IN_CLOSE_WRITE, IN_OPEN, IN_ACCESS, IN_CLOSE_NOWRITE},
}

var fixtureos = Fixture(FixtureFunc{
	IN_ACCESS: func(p string) ([]Event, error) {
		f, err := os.OpenFile(p, os.O_RDWR, 0755)
		if err != nil {
			return nil, err
		}
		if _, err := f.WriteString(p); err != nil {
			f.Close()
			return nil, err
		}
		f.Close()
		f, err = os.Open(p)
		if err != nil {
			return nil, err
		}
		if _, err = f.Read([]byte{0x00}); err != nil && err != io.EOF {
			f.Close()
			return nil, err
		}
		return madev[IN_ACCESS], f.Close()
	},
})
