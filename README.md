eyenotify
=========

DISCLAIMER: This is a work in progress and very alpha! try it, but this nowhere
near production code. Contributions are welcome.

What is it?
-----------

A simple supervisor tool that keeps track of changes in the files, restarting
a given process when files are changed.

Why?
----

This all started when I started looking for an alternative to nodejs supervisor.

The implementation didn't seem to be making use of the OS provided tools to do
the file tracking efficiently (inotify on linux, kqueue on bsd-like).

Another reason was because I wanted to play around with polling algorithms.
"Why use polling when you can use event-based tracking?" some might ask.
One use case is keeping track of file changes when the files are being
shared through a network file system (think NFS, Samba or even Virtual Box
shared folders). Polling kind of sucks in the way it is done in most of the
tools I've seen around and I wanna see if I can come up with something better.

Last but not least, I wanted to make something significant in Go.
This is no 10k lines of code but it was interesting dealing with system calls,
handling binary data and system processes using Go.

Installing
----------

#### Using go get:
`go get -u github.com/dallarosa/eyenotify`

#### Building from source:

1. Clone this repository
2. build using `go build`

Usage
-----

     Usage of eyenotify:
      -command="echo": process to be run
      -ext="go": extension to be watched
      -p=false: use polling
      -polling=false: use polling
      -watch=".": path to be watched

Try

`eyenotify --help`

for the above very concise explanation on the possible options

Supported platforms
-------------------

Right now, the only supported platforms are Linux and Mac OSX (Darwin).
It's in my plans to support Linux, and BSD-like platforms using inotify
and Kqueue and whatever else with polling. Maybe there will be better
ideas in the future.


TODO
----

* Tracking file creation and deletion
  - Right now we're only tracking changes in existing files.
  - Hoping to do this during the holidays
* Test the Mac OSX version
  - I've implemented and compiled the OSX version using a cross-compiler
  - Gotta get this running on a Mac and do proper testing
  - Definitely doing this during the holidays 
* Write tests for the code
  - This one is also a must
* Intelligent Polling
  - Most tools poll all files equally at once when in practice just a few a in use.
  - I'll look more deeply into this one once I get the above worked out
* Refactor the code
  - I created an unnecessary inotifyevent when syscall provides me with one.
* Get the ignore list properly working
  - Right now I'm ignoring .git with hardcoded rules but that should be handled properly
