package agent

import (
	"github.com/qorio/omni/rest"
	"net/http"
	"time"
)

type EndPoint struct {
	agent  *Agent
	start  time.Time
	engine rest.Engine
}

var ServiceId = "agent"

func NewApiEndPoint(agent *Agent) (ep *EndPoint, err error) {
	ep = &EndPoint{
		agent:  agent,
		engine: rest.NewEngine(&Methods, nil, nil),
		start:  time.Now(),
	}

	// Docker Remote API proxy
	dockerApiHandler := agent.createDockerApiHandler(agent.DockerPort)
	ep.engine.Handle("/dockerapi/{docker:.*}", http.StripPrefix("/dockerapi", dockerApiHandler))

	ep.engine.Bind(
		rest.SetHandler(Methods[GetInfo], ep.GetInfo),
		rest.SetHandler(Methods[HealthCheck], ep.HealthCheck),
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
	err := this.engine.MarshalJSON(req, this.agent.GetInfo(), resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) HealthCheck(resp http.ResponseWriter, req *http.Request) {
	health := Types.Health(req).(*Health)
	health.Status = "ok"
	health.UptimeSeconds = time.Now().Sub(this.start).Seconds()
	err := this.engine.MarshalJSON(req, health, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}
