package main

import (
	"io/ioutil"
	"log"
	"runtime/debug"
	"strings"
	"syscall"
)

func processEvent() {
	restartProc()
}

func runKqueue() {
	kq, err := syscall.Kqueue()
	if err != nil {
		log.Fatal("error initializing Kqueue: ", err)
		return
	}

	evTrackList := []syscall.Kevent_t{}

	addFilesToKqueue(&evTrackList, path)

	events := make([]syscall.Kevent_t, 10)

	// wait for events
	for {
		// create kevent
		log.Print("I should be blocking")
		log.Print("evTrackList", evTrackList)
		n, err := syscall.Kevent(kq, evTrackList, events, nil)
		log.Print("n: ", n)
		log.Print("events: ", events)
		if err != nil {
			debug.PrintStack()
			log.Print("Error creating kevent: ", err)
		}
		// check if there was an event and process it
		if len(events) > 0 && events[0].Ident > 0 {
			processEvent()
		}
	}
}

func addFilesToKqueue(evTrackList *[]syscall.Kevent_t, dirPath string) {
	fileList, err := ioutil.ReadDir(dirPath)
	if err != nil {
		log.Fatal("ReadDir failed: ", err)
	}
	for _, file := range fileList {
		newPath := dirPath + "/" + file.Name()
		if file.IsDir() && file.Name() != ".git" {
			addFilesToPoll(newPath)
		} else {
			fileName := file.Name()
			if len(strings.Split(fileName, ".")) > 1 {
				fileExt := strings.Split(fileName, ".")[1]
				if fileExt == ext {
					log.Print("newPath: ", newPath)
					log.Print("fileName: ", fileName)
					fd, err := syscall.Open(newPath, syscall.O_RDONLY, 0)
					if err != nil {
						log.Fatal("Open failed: ", err)
					}
					// build kevent
					ev := syscall.Kevent_t{
						Ident:  uint64(fd),
						Filter: syscall.EVFILT_VNODE,
						Flags:  syscall.EV_ADD | syscall.EV_ENABLE,
						Fflags: syscall.NOTE_EXTEND | syscall.NOTE_WRITE | syscall.NOTE_RENAME | syscall.NOTE_DELETE,
					}
					*evTrackList = append(*evTrackList, ev)
					log.Print(*evTrackList)
				}
			}
		}
	}
}

func main() {
	startProc()
	if polling {
		runPolling()
	} else {
		runKqueue()
	}
}
