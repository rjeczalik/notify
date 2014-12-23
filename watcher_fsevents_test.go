// +build darwin,!kqueue
// +build !fsnotify

package notify

import "testing"

func TestCanonicalize(t *testing.T) {
	cases := [...]struct {
		path string
		full string
	}{
		{"/etc", "/private/etc"},
		{"/tmp", "/private/tmp"},
		{"/var", "/private/var"},
	}
	for i, cas := range cases {
		full, err := canonical(cas.path)
		if err != nil {
			t.Errorf("want err=nil; got %v (i=%d)", err, i)
			continue
		}
		if full != cas.full {
			t.Errorf("want full=%q; got %q (i=%d)", cas.full, full, i)
			continue
		}
	}
}
