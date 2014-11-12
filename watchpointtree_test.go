package notify_test

import (
	"runtime"
	"testing"

	"github.com/rjeczalik/notify/test"
)

func TestMakePath(t *testing.T) {
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
	test.ExpectPath(t, cases)
}

func TestMakeTree(t *testing.T) {
	// TODO(rjeczalik): Add more test-cases.
	cases := test.TreeCase{
		"/github.com/rjeczalik/fs": {
			"/github.com/rjeczalik/fs":            {},
			"/github.com/rjeczalik/fs/cmd":        {},
			"/github.com/rjeczalik/fs/cmd/gotree": {},
			"/github.com/rjeczalik/fs/cmd/mktree": {},
			"/github.com/rjeczalik/fs/fsutil":     {},
			"/github.com/rjeczalik/fs/memfs":      {},
		},
	}
	test.ExpectTree(t, cases)
}
