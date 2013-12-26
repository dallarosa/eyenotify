package main

import (
  "flag"
	"time"
  "log"
  "strings"
  "os/exec"
	"io/ioutil"
  "os"
)

var (
	path      string                //path to be watched
	command   string                //command to be run
	ext       string                //file extension to be watched. right now only supporting one.
	pid       int                   //pid of the process being run
	polling   bool                  // should we poll or not
	pollList  map[string]polledFile //list of files to poll
	ignoreDir map[string]bool       //list of directories to ignore. not really working now
)

func init() {
	flag.StringVar(&path, "watch", ".", "path to be watched")
	flag.StringVar(&command, "command", "echo", "process to be run")
	flag.StringVar(&ext, "ext", "go", "extension to be watched")
	flag.BoolVar(&polling, "polling", false, "use polling")
	flag.BoolVar(&polling, "p", false, "use polling")
	flag.Parse()
	ignoreDir := make(map[string]bool, 256)
	ignoreDir[".git"] = true
}

//struct to hold the information on files being polled.
//for now we keep the minimum information necessary for the job.
type polledFile struct {
	path    string
	modTime time.Time
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
