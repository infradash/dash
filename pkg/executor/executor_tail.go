package executor

import (
	"fmt"
	"github.com/golang/glog"
	_ "github.com/qorio/maestro/pkg/mqtt"
	"github.com/qorio/maestro/pkg/pubsub"
	"io"
	"io/ioutil"
	"os"
	"time"
)

// This executes asynchronously
func (this *Executor) HandleTailFile(req *TailFile) {
	tail := *req
	go func() {
		var out io.Writer = ioutil.Discard // goes to /dev/null

		if tail.Stdout {
			glog.Infoln(tail.Path, "==> stdout")
			out = io.MultiWriter(out, os.Stdout)
		}

		if tail.Stderr {
			glog.Infoln(tail.Path, "==> stderr")
			out = io.MultiWriter(out, os.Stderr)
		}

		if len(tail.Topic) > 0 {
			glog.Infoln(tail.Path, "==>", tail.Topic)
			broker, err := tail.Topic.Broker().PubSub(this.Id)
			if err != nil {
				glog.Warningln("Cannot connect to pubsub.  Not publishing to", tail.Topic)
			} else {
				out = io.MultiWriter(out, pubsub.GetWriter(tail.Topic, broker))
			}
		}
		this.TailFile(tail.Path, out)
	}()
}

func (this *Executor) TailFile(path string, outstream io.Writer) error {
	glog.Infoln("Tailing file", path, outstream)

	tailer := &Tailer{
		Path: path,
	}

	// results channel
	output := make(chan interface{})
	stop := make(chan bool)
	go func() {
		for {
			select {
			case line := <-output:
				fmt.Fprintf(outstream, "%s", line)
				glog.V(100).Infoln(path, "=>", fmt.Sprintf("%s", line))
			case term := <-stop:
				if term {
					return
				}
			}
		}
	}()

	// Start tailing
	stop_tail := make(chan bool)
	go func() {

		tries := 0
		for {
			err := tailer.Start(output, stop_tail)
			if err != nil {

				// This can go on indefinitely.  We want this behavior because some files
				// don't get written until requests come in.  So a file can be missing for a while.
				if this.TailFileOpenRetries > 0 && tries >= this.TailFileOpenRetries {
					glog.Warningln("Stopping trying to tail", path, "Attempts:", tries)
					break
				}
				glog.Warningln("Error while tailing", path, "Err:", err, "Attempts:", tries)
				time.Sleep(time.Duration(this.TailFileRetryWaitTime))
				tries++

			} else {
				glog.Infoln("Starting to tail", path)
				break // no error
			}
		}
		stop <- true // stop reading
	}()

	return nil
}
