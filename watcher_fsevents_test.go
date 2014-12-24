// +build darwin,!kqueue
// +build !fsnotify

package notify

import (
	"io/ioutil"
	"os"
	"testing"
)

func tmpfile(s string) (string, error) {
	f, err := ioutil.TempFile("/tmp", s)
	if err != nil {
		return "", err
	}
	if err = nonil(f.Sync(), f.Close()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func symlink(s string) (string, error) {
	name, err := tmpfile("symlink")
	if err != nil {
		return "", err
	}
	if err = nonil(os.Remove(name), os.Symlink(s, name)); err != nil {
		return "", err
	}
	return name, nil
}

func remove(s ...string) {
	for _, s := range s {
		os.Remove(s)
	}
}

type caseCanonical struct {
	path string
	full string
}

func testCanonical(t *testing.T, cases []caseCanonical) {
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

func TestCanonicalize(t *testing.T) {
	cases := [...]caseCanonical{
		{"/etc", "/private/etc"},
		{"/etc/defaults", "/private/etc/defaults"},
		{"/etc/hosts", "/private/etc/hosts"},
		{"/tmp", "/private/tmp"},
		{"/var", "/private/var"},
	}
	testCanonical(t, cases[:])
}

func TestCanonicalizeMultiple(t *testing.T) {
	link1, err := symlink("/etc")
	if err != nil {
		t.Fatal(err)
	}
	link2, err := symlink("/tmp")
	if err != nil {
		t.Fatal(nonil(err, os.Remove(link1)))
	}
	defer remove(link1, link2)
	cases := [...]caseCanonical{
		{link1, "/private/etc"},
		{link1 + "/hosts", "/private/etc/hosts"},
		{link2, "/private/tmp"},
	}
	testCanonical(t, cases[:])
}

func TestCanonicalizeCircular(t *testing.T) {
	tmp1, err := tmpfile("circular")
	if err != nil {
		t.Fatal(err)
	}
	tmp2, err := tmpfile("circular")
	if err != nil {
		t.Fatal(nonil(err, os.Remove(tmp1)))
	}
	defer remove(tmp1, tmp2)
	// Symlink tmp1 -> tmp2.
	if err = nonil(os.Remove(tmp1), os.Symlink(tmp2, tmp1)); err != nil {
		t.Fatal(err)
	}
	// Symlnik tmp2 -> tmp1.
	if err = nonil(os.Remove(tmp2), os.Symlink(tmp1, tmp2)); err != nil {
		t.Fatal(err)
	}
	if _, err = canonical(tmp1); err == nil {
		t.Fatalf("want canonical(%q)!=nil", tmp1)
	}
	if _, ok := err.(*os.PathError); !ok {
		t.Fatalf("want canonical(%q)=os.PathError; got %T", tmp1, err)
	}
}
