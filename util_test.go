package notify

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func tmpfile(s string) (string, error) {
	f, err := ioutil.TempFile(filepath.Split(s))
	if err != nil {
		return "", err
	}
	if err = nonil(f.Sync(), f.Close()); err != nil {
		return "", err
	}
	return f.Name(), nil
}

func symlink(src, dst string) (string, error) {
	name, err := tmpfile(dst)
	if err != nil {
		return "", err
	}
	if err = nonil(os.Remove(name), os.Symlink(src, name)); err != nil {
		return "", err
	}
	return name, nil
}

func removeall(s ...string) {
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

func TestCanonical(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal("os.Getwd()=%v", err)
	}
	wdsym, err := symlink(wd, "")
	if err != nil {
		t.Fatalf(`symlink(%q, "")=%v`, wd, err)
	}
	td := filepath.Join(wd, "testdata")
	tdsym, err := symlink(td, td)
	if err != nil {
		t.Errorf("symlink(%q, %q)=%v", td, td, nonil(err, os.Remove(wdsym)))
	}
	defer removeall(wdsym, tdsym)
	vfstxt := filepath.Join(td, "vfs.txt")
	cases := [...]caseCanonical{
		{wdsym, wd},
		{tdsym, td},
		{filepath.Join(wdsym, "notify.go"), filepath.Join(wd, "notify.go")},
		{filepath.Join(tdsym, "vfs.txt"), vfstxt},
		{filepath.Join(wdsym, filepath.Base(tdsym), "vfs.txt"), vfstxt},
	}
	testCanonical(t, cases[:])
}

func TestCanonicalCircular(t *testing.T) {
	tmp1, err := tmpfile("circular")
	if err != nil {
		t.Fatal(err)
	}
	tmp2, err := tmpfile("circular")
	if err != nil {
		t.Fatal(nonil(err, os.Remove(tmp1)))
	}
	defer removeall(tmp1, tmp2)
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

func TestJoinevents(t *testing.T) {
	cases := [...]struct {
		evs []Event
		ev  Event
	}{
		0: {nil, All},
		1: {[]Event{}, All},
		2: {[]Event{Create}, Create},
		3: {[]Event{Move}, Move},
		4: {[]Event{Create, Write, Delete}, Create | Write | Delete},
	}
	for i, cas := range cases {
		if ev := joinevents(cas.evs); ev != cas.ev {
			t.Errorf("want event=%v; got %v (i=%d)", cas.ev, ev, i)
		}
	}
}

func TestTreeSplit(t *testing.T) {
	cases := [...]struct {
		path string
		dir  string
		base string
	}{
		{"/github.com/rjeczalik/fakerpc", "/github.com/rjeczalik", "fakerpc"},
		{"/home/rjeczalik/src", "/home/rjeczalik", "src"},
		{"/Users/pknap/porn/gopher.avi", "/Users/pknap/porn", "gopher.avi"},
		{"C:/Documents and Users", "C:", "Documents and Users"},
		{"C:/Documents and Users/pblaszczyk/wiertarka.exe", "C:/Documents and Users/pblaszczyk", "wiertarka.exe"},
		{"/home/(╯°□°）╯︵ ┻━┻", "/home", "(╯°□°）╯︵ ┻━┻"},
	}
	for i, cas := range cases {
		dir, base := Split(filepath.FromSlash(cas.path))
		if want := filepath.FromSlash(cas.dir); dir != want {
			t.Errorf("want dir=%s; got %s (i=%d)", want, dir, i)
		}
		if want := filepath.FromSlash(cas.base); base != want {
			t.Errorf("want base=%s; got %s (i=%d)", want, base, i)
		}
	}
}

func TestTreeBase(t *testing.T) {
	cases := [...]struct {
		path string
		base string
	}{
		{"/github.com/rjeczalik/fakerpc", "fakerpc"},
		{"/home/rjeczalik/src", "src"},
		{"/Users/pknap/porn/gopher.avi", "gopher.avi"},
		{"C:/Documents and Users", "Documents and Users"},
		{"C:/Documents and Users/pblaszczyk/wiertarka.exe", "wiertarka.exe"},
		{"/home/(╯°□°）╯︵ ┻━┻", "(╯°□°）╯︵ ┻━┻"},
	}
	for i, cas := range cases {
		if base := Base(filepath.FromSlash(cas.path)); base != cas.base {
			t.Errorf("want base=%s; got %s (i=%d)", cas.base, base, i)
		}
	}
}

func TestTreeIndexSep(t *testing.T) {
	cases := [...]struct {
		path string
		n    int
	}{
		{"github.com/rjeczalik/fakerpc", 10},
		{"home/rjeczalik/src", 4},
		{"Users/pknap/porn/gopher.avi", 5},
		{"C:/Documents and Users", 2},
		{"Documents and Users/pblaszczyk/wiertarka.exe", 19},
		{"(╯°□°）╯︵ ┻━┻/Downloads", 30},
	}
	for i, cas := range cases {
		if n := IndexSep(filepath.FromSlash(cas.path)); n != cas.n {
			t.Errorf("want n=%d; got %d (i=%d)", cas.n, n, i)
		}
	}
}

func TestTreeLastIndexSep(t *testing.T) {
	cases := [...]struct {
		path string
		n    int
	}{
		{"github.com/rjeczalik/fakerpc", 20},
		{"home/rjeczalik/src", 14},
		{"Users/pknap/porn/gopher.avi", 16},
		{"C:/Documents and Users", 2},
		{"Documents and Users/pblaszczyk/wiertarka.exe", 30},
		{"/home/(╯°□°）╯︵ ┻━┻", 5},
	}
	for i, cas := range cases {
		if n := LastIndexSep(filepath.FromSlash(cas.path)); n != cas.n {
			t.Errorf("want n=%d; got %d (i=%d)", cas.n, n, i)
		}
	}
}
