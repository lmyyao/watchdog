package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/fsnotify/fsnotify"
)

var (
	dirname      string
	help         bool
	writescript  string
	removescript string
)

func init() {
	flag.BoolVar(&help, "h", false, "this help")
	flag.StringVar(&dirname, "f", "", "watch dir name")
	flag.StringVar(&writescript, "w", "", "write script")
	flag.StringVar(&removescript, "d", "", "delete script")

}

type Event fsnotify.Op

// type EventOp func(fsnotify.Event)

type EventOps struct {
	Name   string
	Events map[Event]string
}

func (ops *EventOps) RegistEvent(e Event, op string) {
	ops.Events[e] = op
}

func (ops *EventOps) DeleteEvevnt(e Event) {
	delete(ops.Events, e)
}

type WatchDog struct {
	watcher *fsnotify.Watcher
	data    map[string]*EventOps
	running bool
}

func DefaultNotRegisterDir(ev fsnotify.Event) {
	log.Printf("%s not exist\n", ev.Name)
}

func DefaultNotResiterMethod(ev fsnotify.Event) {
	log.Printf("%s %v not register\n", ev.Name, ev.Op)
}

func DefaultOp(ev fsnotify.Event) {
	log.Printf("%s %v xxxx\n", ev.Name, ev.Op)
}

func DefaultError(e error) {
	log.Println("error:", e)
}

func (dog *WatchDog) Register(ops *EventOps) {
	//TODO
	dog.data[ops.Name] = ops
	dog.watcher.Add(ops.Name)
}

func (dog *WatchDog) Find(name string) (*EventOps, bool) {
	for k, v := range dog.data {
		if strings.HasPrefix(name, k) {
			return v, true
		}
	}
	return nil, false
}
func (dog *WatchDog) Run() {
	if dog.running {
		return
	}
	dog.running = true

	for {
		select {
		case event, ok := <-dog.watcher.Events:
			if !ok {
				return
			}
			if ops, ok := dog.Find(event.Name); !ok {
				DefaultNotRegisterDir(event)
			} else {
				if op, ok := ops.Events[Event(event.Op)]; !ok {
					DefaultNotResiterMethod(event)
				} else {
					go RunScript(op, event.Name)
				}
			}
		case err, ok := <-dog.watcher.Errors:
			if !ok {
				return
			}
			DefaultError(err)
		}
	}
}

func RunScript(fname string, name string) {
	cmd := exec.Command("/bin/sh", fname, name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

func (dog *WatchDog) Close() {
	dog.watcher.Close()
}

func CreateWatchDog() (*WatchDog, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &WatchDog{watcher, make(map[string]*EventOps), false}, nil
}

func CheckFileExist(fname string) {
	if _, err := os.Stat(fname); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func CheckFileExec(fname string) bool {
	info, _ := os.Stat(fname)
	return info.Mode()&0111 != 0

}
func main() {
	flag.Parse()
	if help {
		flag.Usage()
		os.Exit(0)
	}
	if dirname == "" {
		fmt.Fprintln(os.Stderr, "dirname must be set")
		os.Exit(1)
	} else {
		CheckFileExist(dirname)
	}

	if writescript == "" && removescript == "" {
		fmt.Fprintln(os.Stderr, "writescript or removescript  must be set")
		os.Exit(1)
	} else {
		if len(writescript) > 0 {
			CheckFileExist(writescript)
			if !CheckFileExec(writescript) {
				fmt.Fprintf(os.Stderr, "writescript: %s should be excusive\n", writescript)
				os.Exit(1)
			}
		}
		if len(removescript) > 0 {
			CheckFileExist(removescript)
			if !CheckFileExec(removescript) {
				fmt.Fprintf(os.Stderr, "writescript: %s should be excusive\n", removescript)
				os.Exit(1)
			}
		}
	}
	dog, err := CreateWatchDog()
	if err != nil {
		log.Fatal(err)
	}
	var event = &EventOps{dirname, make(map[Event]string)}
	if len(writescript) > 0 {
		event.RegistEvent(Event(fsnotify.Write), writescript)
	}
	if len(removescript) > 0 {
		event.RegistEvent(Event(fsnotify.Remove), removescript)
	}

	dog.Register(event)
	go dog.Run()
	defer dog.Close()

	done := make(chan bool)
	<-done
}
