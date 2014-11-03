package notify_test

import (
	"runtime"
	"testing"

	"github.com/rjeczalik/notify/test"
)

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
