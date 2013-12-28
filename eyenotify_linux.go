package main

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"
)

const (
	EVENT_SIZE = 16
)

//syscall has its own inotify event struct
//but I'm keeping this one just for the sake of having that string there.
//However I guess this could be eliminated later
type inotifyEvent struct {
	wd     int32
	mask   int32
	cookie int32
	length int32
	name   string
}

var (
	lastEvent *inotifyEvent //keeping track of the last event. this is only useful for the vim problem
)

func intFromByte(byteSlice []byte, data interface{}) {
	err := binary.Read(bytes.NewBuffer(byteSlice), binary.LittleEndian, data)
	if err != nil {
		log.Fatal("binary.read failed: ", err)
	}
}

//Process the buffer from an inotify event.
func processBuffer(n int, buffer []byte) {
	event := new(inotifyEvent)
	var i int32

	for i < int32(n) {
		intFromByte(buffer[i:i+4], &event.wd)
		intFromByte(buffer[i+4:i+8], &event.mask)
		intFromByte(buffer[i+8:i+12], &event.cookie)
		intFromByte(buffer[i+12:i+16], &event.length)
		event.name = string(buffer[i+16 : i+16+event.length])
		event.name = strings.TrimRight(event.name, "\x00")
		i += EVENT_SIZE + event.length

		if len(strings.Split(event.name, ".")) > 1 {
			eventExt := strings.Split(event.name, ".")[1]
			log.Print(ext, " - ", eventExt)
			if ext == eventExt {
				//TODO
				//vim test: This should be done only if some "vim" flag is specified.
				//Some background:
				//=================
				//Editors like Vim instead of saving the updated contents to the existing
				//file, it creates a temp file (normally named "4093"), removes the existing
				//file and renames the temp file to be the new file. This creates a bunch
				//of unnecessary events that get the file tracking crazy.
				//This check guarantees that we won't restart the process twice for the vim case
				if lastEvent != nil && lastEvent.name == event.name && lastEvent.mask == syscall.IN_DELETE && event.mask == syscall.IN_CLOSE_WRITE {
					break
				}
				lastEvent = event
				restartProc()
				break
			}
		}
	}
}

//starts inotify tracking
func runInotify() {
	fd, err := syscall.InotifyInit()
	if err != nil {
		log.Fatal("error initializing Inotify: ", err)
		return
	}
	addFilesToInotify(fd, path)

	var buffer []byte = make([]byte, 1024*EVENT_SIZE)

	for {
		n, err := syscall.Read(fd, buffer)
		if err != nil {
			log.Fatal("Read failed: ", err)
			return
		}
		processBuffer(n, buffer)
	}
}

//Add directories recursively to the tracking list
func addFilesToInotify(fd int, dirPath string) {
	dir, err := os.Stat(dirPath)
	if err != nil {
		log.Fatal("error getting info on dir: ", err)
		return
	}
	if dir.IsDir() && dir.Name() != ".git" {
		log.Print("adding: ", dirPath)
		_, err = syscall.InotifyAddWatch(fd, dirPath, syscall.IN_CLOSE_WRITE|syscall.IN_DELETE)
		if err != nil {
			log.Fatal("error adding watch: ", err)
			return
		}

		fileList, err := ioutil.ReadDir(dirPath)
		if err != nil {
			log.Fatal("error reading dir: ", err)
			return
		}
		for _, file := range fileList {
			newPath := dirPath + "/" + file.Name()
			if file.IsDir() && file.Name() != ".git" {
				addFilesToInotify(fd, newPath)
			}
		}
	}
}

func main() {
	startProc()
	if polling {
		runPolling()
	} else {
		runInotify()
	}
}
