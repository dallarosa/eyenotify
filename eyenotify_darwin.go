package main

import (
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"
	"time"
)

func processEvent() {
	restartProc()
}

func runKqueue() {
	fd, err := syscall.Kqueue()
	evTrackList := make([]syscall.Kevent_t, 1024)
	if err != nil {
		log.Fatal("error initializing Kqueue: ", err)
		return
	}

	addFilesToKqueue(evTrackList, path)

	// configure timeout
	timeout := syscall.Timespec{
		Sec:  0,
		Nsec: 0,
	}

	// wait for events
	for {
		// create kevent
		events := make([]syscall.Kevent_t, 10)
		_, err := syscall.Kevent(fd, evTrackList, events, &timeout)
		if err != nil {
			log.Println("Error creating kevent")
		}
		// check if there was an event and process it
		if len(events) > 0 && events[0].Ident > 0 {
			processEvent()
		}
	}
}

func addFilesToKqueue(evTrackList []syscall.Kevent_t, dirPath string) {
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
					fd, err := syscall.Open(newPath, syscall.O_RDONLY, 0)
					if err != nil {
						log.Fatal("Open failed: ", err)
					}
					// build kevent
					ev := syscall.Kevent_t{
						Ident:  uint64(fd),
						Filter: syscall.EVFILT_VNODE,
						Flags:  syscall.EV_ADD | syscall.EV_ENABLE | syscall.EV_ONESHOT,
						Fflags: syscall.NOTE_DELETE | syscall.NOTE_WRITE,
						Data:   0,
						Udata:  nil,
					}
					evTrackList = append(evTrackList, ev)
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
