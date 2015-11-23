package executor

import (
	"fmt"
	"github.com/golang/glog"
	ps "github.com/mitchellh/go-ps"
	"github.com/qorio/omni/rest"
	"github.com/qorio/omni/version"
	"net/http"
	"os"
	"time"
)

type EndPoint struct {
	executor *Executor
	start    time.Time
	engine   rest.Engine
}

var ServiceId = "executor"

func NewApiEndPoint(executor *Executor) (ep *EndPoint, err error) {
	ep = &EndPoint{
		executor: executor,
		engine:   rest.NewEngine(&Methods, nil, nil),
	}

	ep.engine.Bind(
		rest.SetHandler(Methods[ApiGetInfo], ep.GetInfo),
		rest.SetHandler(Methods[ApiQuitQuitQuit], ep.QuitQuitQuit),
	)
	return ep, nil
}

func (this *EndPoint) Stop() error {
	return nil
}

func (this *EndPoint) ServeHTTP(resp http.ResponseWriter, request *http.Request) {
	this.engine.ServeHTTP(resp, request)
}

func (this *EndPoint) GetInfo(resp http.ResponseWriter, req *http.Request) {
	info := &Info{
		Version:  *version.BuildInfo(),
		Now:      time.Now(),
		Executor: this.executor,
	}

	err := this.engine.MarshalJSON(req, info, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) QuitQuitQuit(resp http.ResponseWriter, req *http.Request) {
	wait_duration := 5 * time.Second
	if queries, err := this.engine.GetUrlQueries(req, Methods[ApiQuitQuitQuit].UrlQueries); err == nil {
		if parsed, err := time.ParseDuration(queries["wait"].(string)); err == nil {
			wait_duration = parsed
		}
	}

	message := fmt.Sprintf("Executor stopping in %v", wait_duration)
	this.engine.HandleError(resp, req, message, http.StatusServiceUnavailable)
	go func() {

		myPid := os.Getpid()

		glog.Infoln("PID=", myPid, "Show processes:")
		// TODO - go through all the child processes and stop them one by one for clean stop
		pss, err := ps.Processes()
		if err == nil {
			for _, p := range pss {
				glog.Infoln("PPID=", p.PPid(), "PID=", p.Pid(), "CMD=", p.Executable())
				if p.PPid() == myPid {
					glog.Infoln("Child process ==>", p.Pid(), "cmd=", p.Executable())
				}
			}

		} else {
			glog.Infoln("Failed to get ps:", err)
		}
		glog.Infoln("Executor going down!!!!!!!!!")
		time.Sleep(wait_duration)
		glog.Infoln("Bye")
		os.Exit(0)
	}()
}
