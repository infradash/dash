package agent

import (
	"bytes"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/qorio/omni/rest"
	"github.com/qorio/omni/version"
	"io/ioutil"
	"net/http"
	"net/url"
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
	}

	// Docker Remote API proxy
	dockerApiHandler := agent.createDockerApiHandler(agent.DockerPort)
	ep.engine.Handle("/dockerapi/{docker:.*}", http.StripPrefix("/dockerapi", dockerApiHandler))

	ep.engine.Bind(
		rest.SetHandler(Methods[GetInfo], ep.GetInfo),
		rest.SetHandler(Methods[ListContainers], ep.ListContainers),
		rest.SetHandler(Methods[WatchContainer], ep.WatchContainer),
		rest.SetHandler(Methods[ConfigureDomain], ep.ConfigureDomain),
		rest.SetHandler(Methods[ForwardMessage], ep.ForwardMessage),
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
		Version: *version.BuildInfo(),
		Now:     time.Now(),
		Agent:   this.agent,
	}

	err := this.engine.MarshalJSON(req, info, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ListContainers(resp http.ResponseWriter, req *http.Request) {
	domain := this.engine.GetUrlParameter(req, "domain")
	service := this.engine.GetUrlParameter(req, "service")
	list, err := this.agent.ListContainers(domain, service)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, cc := range list {
		cc.Inspect()
	}
	err = this.engine.MarshalJSON(req, list, resp)
	if err != nil {
		this.engine.HandleError(resp, req, "malformed", http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) WatchContainer(resp http.ResponseWriter, req *http.Request) {
	domain := this.engine.GetUrlParameter(req, "domain")
	service := this.engine.GetUrlParameter(req, "service")
	spec := Methods[WatchContainer].RequestBody(req).(*WatchContainerSpec)
	err := this.engine.UnmarshalJSON(req, spec)
	if err != nil {
		glog.Warningln("Error", err)
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	err = this.agent.WatchContainer(domain, service, spec)
	glog.Infoln("Start watching container for service:", service, "spec=", spec)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ConfigureDomain(resp http.ResponseWriter, req *http.Request) {
	config := Methods[ConfigureDomain].RequestBody(req).(*DomainConfig)
	err := this.engine.UnmarshalJSON(req, config)
	if err != nil {
		glog.Warningln("Error", err)
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = this.agent.ConfigureDomain(config)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (this *EndPoint) ForwardMessage(resp http.ResponseWriter, req *http.Request) {
	payload := Methods[ForwardMessage].RequestBody(req).(*struct {
		Method  string                  `json:"method"`
		Url     string                  `json:"url"`
		Message *map[string]interface{} `json:"message"`
	})

	if err := this.engine.UnmarshalJSON(req, payload); err != nil {
		glog.Warningln("Error", err)
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	url, err := url.Parse(payload.Url)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
		return
	}

	// Http client and forward
	client := &http.Client{}
	remote_req := &http.Request{
		Method: payload.Method,
		URL:    url,
	}

	if payload.Message != nil {
		message, err := json.Marshal(payload.Message)
		if err != nil {
			this.engine.HandleError(resp, req, err.Error(), http.StatusBadRequest)
			return
		}
		remote_req.Body = ioutil.NopCloser(bytes.NewBuffer(message))
		remote_req.ContentLength = int64(len(message))
	}

	remote_resp, err := client.Do(remote_req)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}

	buff, err := ioutil.ReadAll(remote_resp.Body)
	if err != nil {
		this.engine.HandleError(resp, req, err.Error(), http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", remote_resp.Header.Get("Content-Type"))
	resp.WriteHeader(remote_resp.StatusCode)
	resp.Write(buff)
}
