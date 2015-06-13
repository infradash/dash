package executor

import (
	"fmt"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/mqtt"
	"github.com/qorio/maestro/pkg/pubsub"
	"github.com/qorio/maestro/pkg/registry"
	"github.com/qorio/maestro/pkg/zk"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// This executes asynchronously
func (this *Executor) HandleTailRequest(req *TailRequest) {
	tail := *req // copy
	go func() {
		var out io.Writer = os.Stdout
		switch {
		case tail.Output == "stderr":
			glog.Infoln("Tailing", tail.Path, "sending to stderr")
			out = os.Stderr
		case tail.Output == "mqtt":
			id := fmt.Sprintf("/%s/%s/%s", this.Domain, filepath.Base(tail.Path), this.Host)
			topic := id
			if tail.MQTTTopic != "" {
				topic = tail.MQTTTopic
			}

			glog.Infoln("Tailing", tail.Path, "sending to mqtt as", topic)
			var wait time.Duration
			var addr *string
			for {
				addr = zk.GetString(this.zk, registry.Path(tail.RegistryPath))
				if addr != nil || wait >= this.MQTTConnectionTimeout {
					break
				} else {
					glog.Infoln("Waiting for MQTT broker to become available -- path=", tail.RegistryPath)
					time.Sleep(this.MQTTConnectionRetryWaitTime)
					wait += this.MQTTConnectionRetryWaitTime
				}
			}
			if addr == nil {
				glog.Warningln("Cannot locate MQTT broker. Give up. Path=", tail.RegistryPath)
				return
			}
			mqtt, err := mqtt.Connect(id, *addr)
			if err != nil {
				glog.Warningln("Error starting mqtt client. Err=", err,
					"Broker path=", tail.RegistryPath, "host:port=", *addr, "topic=", topic)
				return
			}
			out = pubsub.GetWriter(pubsub.Topic(topic), mqtt)
			glog.Infoln("MQTT client for", tail.Path, "topic=", topic, "ready", out)

		default:
			// other cases include a url for websocket connection to push the output
			glog.Infoln("Tailing", tail.Path, "sending to stdout")
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

func (this *Executor) hostport_list_from_zk(containers_path string, service_port *int) (interface{}, error) {
	n, err := this.zk.Get(containers_path)
	if err != nil {
		return nil, err
	}
	all, err := n.VisitChildrenRecursive(func(z *zk.Node) bool {
		_, port := ParseHostPort(z.GetBasename())
		return z.IsLeaf() || (service_port != nil && port == strconv.Itoa(*service_port) && z.IsLeaf())
	})
	if err != nil {
		return nil, err
	}

	list := make([]interface{}, 0)
	for _, c := range all {
		host, port := ParseHostPort(c.GetValueString())
		list = append(list, struct {
			Host string
			Port string
		}{
			Host: host,
			Port: port,
		})
	}
	return struct {
		HostPortList []interface{}
	}{
		HostPortList: list,
	}, nil
}
