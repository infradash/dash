package executor

import (
	"bufio"
	"github.com/golang/glog"
	"github.com/howeyc/fsnotify"
	"os"
)

type Tailer struct {
	Path string `json:"path"`
}

func (this *Tailer) Start(out chan<- interface{}, stop <-chan bool) error {
	file, err := os.Open(this.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer watcher.Close()

	err = watcher.Watch(this.Path)
	if err != nil {
		return err
	}

	glog.Infoln("Tail file", this.Path, "starts.")

	file.Seek(0, 2)
	for {
		select {
		case term := <-stop:
			if term {
				glog.Infoln("Stopping tail of file", this.Path)
				break
			}
		case event := <-watcher.Event:

			switch {

			case event.IsCreate():
			case event.IsDelete():
				return nil

			case event.IsRename():
				glog.Infoln("Tailing file", this.Path, "renamed", event.Name)
				file, err = os.Open(event.Name)
				if err != nil {
					return err
				}
				reader = bufio.NewReader(file)

			case event.IsModify() && !event.IsAttrib():
				line, _, err := reader.ReadLine()
				if err != nil {
					glog.Warningln("Error tailing file:", err)
					return err
				}
				out <- line
			}

		case err := <-watcher.Error:
			glog.Warningln("Error watching", this.Path, "Err=", err)
			return err
		}
	}
	return nil
}
