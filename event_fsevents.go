// +build darwin,!kqueue
// +build !fsnotify

package notify

const (
	Create = Event(FSEventsCreated)
	Delete = Event(FSEventsRemoved)
	Write  = Event(FSEventsModified)
	Move   = Event(FSEventsRenamed)

	Recursive = Event(0x80000) // An internal event, in final version won't be exported.
)

const (
	FSEventsMustScanSubDirs Event = 0x00001
	FSEventsUserDropped           = 0x00002
	FSEventsKernelDropped         = 0x00004
	FSEventsEventIdsWrapped       = 0x00008
	FSEventsHistoryDone           = 0x00010
	FSEventsRootChanged           = 0x00020
	FSEventsMount                 = 0x00040
	FSEventsUnmount               = 0x00080
	FSEventsCreated               = 0x00100
	FSEventsRemoved               = 0x00200
	FSEventsInodeMetaMod          = 0x00400
	FSEventsRenamed               = 0x00800
	FSEventsModified              = 0x01000
	FSEventsFinderInfoMod         = 0x02000
	FSEventsChangeOwner           = 0x04000
	FSEventsXattrMod              = 0x08000
	FSEventsIsFile                = 0x10000
	FSEventsIsDir                 = 0x20000
	FSEventsIsSymlink             = 0x40000
)

var osestr = map[Event]string{
	FSEventsMustScanSubDirs: "notify.FSEventsMustScanSubDirs",
	FSEventsUserDropped:     "notify.FSEventsUserDropped",
	FSEventsKernelDropped:   "notify.FSEventsKernelDropped",
	FSEventsEventIdsWrapped: "notify.FSEventsEventIdsWrapped",
	FSEventsHistoryDone:     "notify.FSEventsHistoryDone",
	FSEventsRootChanged:     "notify.FSEventsRootChanged",
	FSEventsMount:           "notify.FSEventsMount",
	FSEventsUnmount:         "notify.FSEventsUnmount",
	FSEventsInodeMetaMod:    "notify.FSEventsInodeMetaMod",
	FSEventsFinderInfoMod:   "notify.FSEventsFinderInfoMod",
	FSEventsChangeOwner:     "notify.FSEventsChangeOwner",
	FSEventsXattrMod:        "notify.FSEventsXattrMod",
	FSEventsIsFile:          "notify.FSEventsIsFile",
	FSEventsIsDir:           "notify.FSEventsIsDir",
	FSEventsIsSymlink:       "notify.FSEventsIsSymlink",
}

var ekind = map[Event]Event{}

type event struct {
	fse   FSEvent
	event Event
	isdir bool
}

func (ei *event) Event() Event         { return ei.event }
func (ei *event) Path() string         { return ei.fse.Path }
func (ei *event) IsDir() (bool, error) { return ei.isdir, nil }
func (ei *event) Sys() interface{}     { return &ei.fse }
