package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
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

//struct to hold the information on files being polled.
//for now we keep the minimum information necessary for the job.
type polledFile struct {
	path    string
	modTime time.Time
}

var (
	path      string                //path to be watched
	command   string                //command to be run
	ext       string                //file extension to be watched. right now only supporting one.
	pid       int                   //pid of the process being run
	polling   bool                  // should we poll or not
	lastEvent *inotifyEvent         //keeping track of the last event. this is only useful for the vim problem
	pollList  map[string]polledFile //list of files to poll
	ignoreDir map[string]bool       //list of directories to ignore. not really working now
)

func init() {
	flag.StringVar(&path, "watch", ".", "path to be watched")
	flag.StringVar(&command, "command", "echo", "path to be watched")
	flag.StringVar(&ext, "ext", "go", "extension to be watched")
	flag.BoolVar(&polling, "polling", false, "use polling")
	flag.BoolVar(&polling, "p", false, "use polling")
	flag.Parse()
	ignoreDir := make(map[string]bool, 256)
	ignoreDir[".git"] = true
}

func intFromByte(byteSlice []byte, data interface{}) {
	err := binary.Read(bytes.NewBuffer(byteSlice), binary.LittleEndian, data)
	if err != nil {
		log.Fatal("binary.read failed: ", err)
	}
}

//Starts the process specified in the command line
//keeps track of the process pid for restart
func startProc() {
	log.Print("Starting Process...")
	commandArray := strings.Split(command, " ")
	paramArray := commandArray[1:]
	cmd := exec.Command(commandArray[0], paramArray...)
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Process Started Successfuly: ", cmd.Process.Pid)
	pid = cmd.Process.Pid
}

//Restart the process with pid value in the global variable pid
//If it cannot find the process to kill assume the process is
//already dead and start a new instance
func restartProc() {
	log.Print("Killing Process:  ", pid)
	if proc, err := os.FindProcess(pid); err != nil {
		log.Print("error: ", err)
		startProc()
	} else {
		err := proc.Kill()
		if err != nil {
			log.Print("error: ", err)
		}
		_, err = proc.Wait()
		if err != nil {
			log.Print("error: ", err)
		}
		startProc()
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

//Add files and directories to the polling list
func addFilesToPoll(filePath string) {
	fileList, err := ioutil.ReadDir(filePath)
	if err != nil {
		log.Fatal("ReadDir failed: ", err)
	}
	for _, file := range fileList {
		newPath := filePath + "/" + file.Name()
		if file.IsDir() && file.Name() != ".git" {
			pollList[newPath] = polledFile{path: newPath, modTime: file.ModTime()}
			addFilesToPoll(newPath)
		} else {
			fileName := file.Name()
			if len(strings.Split(fileName, ".")) > 1 {
				fileExt := strings.Split(fileName, ".")[1]
				if fileExt == ext {
					pollList[newPath] = polledFile{path: newPath, modTime: file.ModTime()}
					log.Print(fileName, " - ", file.ModTime())
				}
			}
		}
	}
}

//starts poll-based tracking
func runPolling() {
	pollList = make(map[string]polledFile)
	addFilesToPoll(path)
	for {
		for path, pollFile := range pollList {
			fileInfo, err := os.Stat(path)
			if err != nil {
				log.Fatal("Stat error: ", err)
			}
			if pollFile.modTime.Before(fileInfo.ModTime()) {
				restartProc()
			}
			pollList[path] = polledFile{path: path, modTime: fileInfo.ModTime()}
		}
		time.Sleep(200 * time.Millisecond)
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
