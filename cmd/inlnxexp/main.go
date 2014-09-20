// +build windows
package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	//"runtime"
	"syscall"
	"time"
)

// tmp
type Event uint8

const (
	Create Event = 1 << iota
	Write
	Remove
	Rename
	Recursive
)

const All Event = Create | Write | Remove | Rename | Recursive

type EventInfo interface {
	Name() string
	Event() Event
	Sys() interface{}
}

// tmp

func loop() {
	for {
		process()
	}
}

func watch(path string) {
	var err error
	wd, err = syscall.InotifyAddWatch(fd, path, allFlags)
	if wd == -1 {
		fmt.Println(os.NewSyscallError("InotifyAddWatch", err))
	}
}

func process() {
	var n int
	var err error

	n, err = syscall.Read(fd, buffer[:])

	fmt.Println("\n====================\n")
	fmt.Println("NumberOfBytes:", n)
	fmt.Println("Error:", err)
	fmt.Println("\n====================\n")

	if n != 0 {
	} else {
		fmt.Println("no data received")
	}
}

func action(action uint32) {
	switch action {
	case syscall.IN_ACCESS:
		fmt.Println(">> action: IN_ACCESS")
	case syscall.IN_MODIFY:
		fmt.Println(">> action: IN_MODIFY")
	case syscall.IN_ATTRIB:
		fmt.Println(">> action: IN_ATTRIB")
	case syscall.IN_CLOSE_WRITE:
		fmt.Println(">> action: IN_CLOSE_WRITE")
	case syscall.IN_CLOSE_NOWRITE:
		fmt.Println(">> action: IN_CLOSE_NOWRITE")
	case syscall.IN_OPEN:
		fmt.Println(">> action: IN_OPEN")
	case syscall.IN_MOVED_FROM:
		fmt.Println(">> action: IN_MOVED_FROM")
	case syscall.IN_MOVED_TO:
		fmt.Println(">> action: IN_MOVED_TO")
	case syscall.IN_CREATE:
		fmt.Println(">> action: IN_CREATE")
	case syscall.IN_DELETE:
		fmt.Println(">> action: IN_DELETE")
	case syscall.IN_DELETE_SELF:
		fmt.Println(">> action: IN_DELETE_SELF")
	case syscall.IN_MOVE_SELF:
		fmt.Println(">> action: IN_MOVE_SELF")
	}
}

const (
	allFlags = syscall.IN_ACCESS |
		syscall.IN_MODIFY |
		syscall.IN_ATTRIB |
		syscall.IN_CLOSE_WRITE |
		syscall.IN_CLOSE_NOWRITE |
		syscall.IN_OPEN |
		syscall.IN_MOVED_FROM |
		syscall.IN_MOVED_TO |
		syscall.IN_CREATE |
		syscall.IN_DELETE |
		syscall.IN_DELETE_SELF |
		syscall.IN_MOVE_SELF

	// syscall.IN_UNMOUNT // Backing fs was unmounted.
	// syscall.IN_Q_OVERFLOW // Event queued overflowed.
	// syscall.IN_IGNORED // File was ignored.

	// syscall.IN_ISDIR // Event occurred against dir.
	// syscall.IN_ONESHOT // Only send event once.
)

const TestFiles = 1

var files map[string]bool

func randomFileOp(path string) {
	files = make(map[string]bool)
	for {
		time.Sleep(4 * (time.Duration(rand.Intn(int(time.Second))) + time.Second))
		fileshort := fmt.Sprintf("file_longname_%d.txt", rand.Intn(TestFiles))
		filefull := filepath.Join(path, fileshort)
		if _, ok := files[fileshort]; !ok {
			if err := ioutil.WriteFile(filefull, []byte("h"), 0644); err != nil {
				fmt.Printf("[ERROR:RFO] Cannot write file %s: %v.\n", filefull, err)
				continue
			}
			files[fileshort] = true
			fmt.Printf("[INFO:RFO] File %s created.\n", filefull)
		} else {
			modDel := rand.Intn(2)
			if modDel == 0 {
				if err := ioutil.WriteFile(filefull, []byte("X"), 0644|os.ModeAppend); err != nil {
					fmt.Printf("[ERROR:RFO] Cannot modify file %s: %v.\n", filefull, err)
					continue
				}
				fmt.Printf("[INFO:RFO] File %s modified.\n", filefull)
			} else {
				if err := os.Remove(filefull); err != nil {
					fmt.Printf("[ERROR:RFO] Cannot remove file %s: %v.\n", filefull, err)
					continue
				}
				delete(files, fileshort)
				fmt.Printf("[INFO:RFO] File %s deleted.\n", filefull)
			}
		}
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	pathTmp, err := ioutil.TempDir("", "notify")
	if err != nil {
		fmt.Printf("[ERROR:MAIN] Cannot create temporary directory: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(pathTmp)

	watch(pathTmp)
	time.Sleep(time.Second)
	fmt.Println("=======================")
	go randomFileOp(pathTmp)
	var input uint32
	for {
		fmt.Println("Type `1` to exit")
		_, err := fmt.Scanln(&input)
		if err != nil {
			fmt.Println("Error: ", err)
			continue
		}
		if input == 1 {
			break
		}
	}
}
