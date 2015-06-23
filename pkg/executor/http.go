package executor

import (
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
		rest.SetHandler(Methods[ApiGetInfo], ep.GetInfo),
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
