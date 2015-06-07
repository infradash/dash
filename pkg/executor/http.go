package executor

import (
	. "github.com/infradash/dash/pkg/dash"
	"github.com/golang/glog"
	"github.com/qorio/omni/rest"
	"github.com/qorio/omni/version"
	"net/http"
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
		rest.SetHandler(Methods[GetInfo], ep.GetInfo),
		rest.SetHandler(Methods[SaveWatchAction], ep.SaveWatchAction),
		rest.SetHandler(Methods[GetWatchAction], ep.GetWatchAction),
		rest.SetHandler(Methods[TailFile], ep.TailFile),
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

func (this *EndPoint) SaveWatchAction(resp http.ResponseWriter, req *http.Request) {
	watch := Methods[SaveWatchAction].RequestBody(req).(*RegistryWatch)
	err := this.engine.UnmarshalJSON(req, watch)
	if err != nil {
		glog.Warningln("Error", err)
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	watch.Domain = this.engine.GetUrlParameter(req, "domain")
	watch.Service = this.engine.GetUrlParameter(req, "service")

	err = this.executor.SaveWatchAction(watch)
	if err != nil {
		glog.Warningln("Error", err)
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	return
}

func (this *EndPoint) GetWatchAction(resp http.ResponseWriter, req *http.Request) {
	domain := this.engine.GetUrlParameter(req, "domain")
	service := this.engine.GetUrlParameter(req, "service")

	key, _, err := RegistryKeyValue(KLive, map[string]string{
		"Service": service,
		"Domain":  domain,
	})

	watch, err := this.executor.GetWatchAction(key)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	if watch == nil {
		this.engine.HandleError(resp, req, "not-found", http.StatusNotFound)
		return
	}
	err = this.engine.MarshalJSON(req, watch, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) TailFile(resp http.ResponseWriter, req *http.Request) {
	tail := Methods[TailFile].RequestBody(req).(*TailRequest)
	if err := this.engine.UnmarshalJSON(req, tail); err != nil {
		glog.Warningln("Error", err)
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}
	this.executor.HandleTailRequest(tail)
	return
}
