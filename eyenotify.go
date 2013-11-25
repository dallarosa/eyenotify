package main

import (
	"syscall"
	"bytes"
	"encoding/binary"
	"flag"
	"log"
	"os"
	"os/exec"
	"strings"
)

const (
	EVENT_SIZE = 16
)

type inotifyEvent struct {
	wd int32
	mask int32
	cookie int32
	length int32
	name string
}

var (
	path string
	command string
	ext string
	pid int
)

func init() {
	flag.StringVar(&path, "path", ".", "path to be watched")
	flag.StringVar(&command, "command", "echo", "path to be watched")
	flag.StringVar(&ext, "ext", "go", "extension to be watched")
	flag.Parse()
}

func intFromByte(byteSlice []byte, data interface{} ) {
	err := binary.Read(bytes.NewBuffer(byteSlice), binary.LittleEndian, data)
	if err != nil {
		log.Fatal("binary.read failed: ", err)
	}
}

func runApp() {
	log.Print("Starting Process...")
	commandArray := strings.Split(command, " ")
	paramArray := commandArray[1:]
	cmd := exec.Command(commandArray[0], paramArray...)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Process Started Successfuly: ",cmd.Process.Pid)
	pid = cmd.Process.Pid
}

func main() {
	runApp()
	fd, err := syscall.InotifyInit()
	if err != nil {
		log.Fatal("error initializing Inotify: ", err)
		return
	}
	_, err = syscall.InotifyAddWatch(fd, path, syscall.IN_CLOSE_WRITE)
	if err != nil {
		log.Fatal("error adding watch: ", err)
		return
	}

	var buffer []byte = make([]byte, 1024 * EVENT_SIZE)

	for {
		var  i int32
		event := new(inotifyEvent)
		n, err := syscall.Read(fd, buffer)
		if err != nil {
			log.Fatal("Read failed: ", err)
			return
		}
		for i < int32(n) {
			intFromByte(buffer[i: i + 4], &event.wd)
			intFromByte(buffer[i+ 4: i + 8], &event.mask)
			intFromByte(buffer[i + 8: i + 12], &event.cookie)
			intFromByte(buffer[i + 12: i + 16], &event.length)
			event.name =string(buffer[i + 16: i + 16 + event.length])
			event.name = strings.TrimRight(event.name, "\x00")
			i += EVENT_SIZE + event.length

			if(len(strings.Split(event.name,".")) > 1) {
				eventExt := strings.Split(event.name,".")[1]
				if(ext == eventExt){
					log.Print("Killing Process:  ",pid)
					if proc, err := os.FindProcess(pid); err != nil {
						log.Print("error: ",err)
						runApp()
					}else{
						err := proc.Kill()
						if err != nil {
							log.Print("error: ", err)
						}
						_, err = proc.Wait()
						if err != nil {
							log.Print("error: ", err)
						}
						runApp()
					}
					break
				}
			}
		}
	}
}
