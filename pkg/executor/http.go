package executor

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/qorio/omni/rest"
	"net/http"
	"os"
	"sync"
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
		rest.SetHandler(Methods[ApiProcessList], ep.ProcessList),
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
	info := this.executor.GetInfo()
	info.NowUnix = time.Now().Unix()
	info.UptimeSeconds = time.Unix(info.NowUnix, 0).Sub(time.Unix(this.executor.StartTimeUnix, 0)).Seconds()

	err := this.engine.MarshalJSON(req, info, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ProcessList(resp http.ResponseWriter, req *http.Request) {
	result, err := children_processes()
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	err = this.engine.MarshalJSON(req, result, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) QuitQuitQuit(resp http.ResponseWriter, req *http.Request) {
	wait_duration := 5 * time.Second
	if form, err := this.engine.GetPostForm(req, Methods[ApiQuitQuitQuit].FormParams); err == nil {
		if parsed, err := time.ParseDuration(form["wait"].(string)); err == nil {
			wait_duration = parsed
		}
	}

	message := fmt.Sprintf("Executor stopping in %v", wait_duration)
	this.engine.HandleError(resp, req, message, http.StatusServiceUnavailable)
	go func() {

		glog.Infoln("Shutdown in", wait_duration, "!!!!!!!!!!!!!")
		time.Sleep(wait_duration)

		var wg sync.WaitGroup

		result, err := children_processes()
		if err == nil {

			for _, p := range result {
				go func() {
					defer wg.Done()
					state, err := p.Process.Wait()
					glog.Infoln("state=", state, "err=", err)
				}()
				wg.Add(1)
				err := p.Process.Kill()
				glog.Infoln("Killing process", p, err)
			}
		}

		wg.Wait()

		glog.Infoln("Executor going down!!!!!!!!!")
		glog.Infoln("Bye")
		os.Exit(0)
	}()
}
