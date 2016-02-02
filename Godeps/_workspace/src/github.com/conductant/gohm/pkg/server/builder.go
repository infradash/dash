package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"reflect"
	"runtime"
	"sync"
	"time"
)

type routeBuilder struct {
	parent  *serviceBuilder
	binding *methodBinding
}

func (this *routeBuilder) To(h Handler) *serviceBuilder {
	this.binding.Handler = h
	return this.parent
}

func NewService() *serviceBuilder {
	return &serviceBuilder{
		routes:          make(map[string]*routeBuilder),
		port:            8080,
		shutdownTimeout: 10 * time.Second,
		onShutdownFunc:  func() error { return nil },
	}
}

type serviceBuilder struct {
	port            int
	shutdownTimeout time.Duration
	onShutdownFunc  func() error
	routes          map[string]*routeBuilder
	engine          *engine
	auth            AuthManager
	webhooks        WebhookManager
	lock            sync.Mutex
}

func (this *serviceBuilder) ListenPort(port int) *serviceBuilder {
	this.port = port
	return this
}

func (this *serviceBuilder) ShutdownTimeout(timeout time.Duration) *serviceBuilder {
	this.shutdownTimeout = timeout
	return this
}

func (this *serviceBuilder) OnShutdown(run func() error) *serviceBuilder {
	this.onShutdownFunc = run
	return this
}

func (this *serviceBuilder) WithAuth(auth AuthManager) *serviceBuilder {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.auth = auth
	if this.engine != nil {
		this.engine.auth = auth
	}
	return this
}

func (this *serviceBuilder) WithWebhooks(webhooks WebhookManager) *serviceBuilder {
	this.lock.Lock()
	defer this.lock.Unlock()

	this.webhooks = webhooks
	if this.engine != nil {
		this.engine.webhooks = webhooks
	}
	return this
}

func (this *serviceBuilder) Route(m ServiceMethod) *routeBuilder {
	route := &routeBuilder{
		parent: this,
		binding: &methodBinding{
			Api: m,
		}}
	this.routes[string(m.HttpMethod)+"/"+m.UrlRoute] = route
	return route
}

func (this *serviceBuilder) Start() (chan<- int, <-chan error) {
	return Start(this.port, this.Build(), this.onShutdownFunc, this.shutdownTimeout)
}

func (this *serviceBuilder) Build() Server {
	this.lock.Lock()
	defer this.lock.Unlock()

	if this.auth == nil {
		panic(fmt.Errorf("AuthManager is not set."))
	}

	if this.engine == nil {
		this.engine = &engine{
			renderError:   DefaultErrorRenderer,
			routes:        make(map[string]*methodBinding),
			functionNames: make(map[string]*methodBinding),
			router:        mux.NewRouter(),
			auth:          this.auth,
			event_chan:    make(chan *ServerEvent),
			done_chan:     make(chan bool),
			webhooks:      this.webhooks,
			sseChannels:   make(map[string]*sseChannel),
		}
	}

	for methodRoute, builder := range this.routes {
		binding := builder.binding
		this.engine.routes[methodRoute] = binding

		// Get the function name of the handler and use that to index bindings.
		fn := cleanFuncName(runtime.FuncForPC(reflect.ValueOf(binding.Handler).Pointer()).Name())
		this.engine.functionNames[fn] = binding

		if binding.Handler == nil {
			panic(fmt.Sprintf("No implementation for REST endpoint: %s", binding.Api))
		}

		h := this.engine.router.HandleFunc(binding.Api.UrlRoute, httpHandler(this.engine, binding, this.auth))
		if len(binding.Api.HttpMethods) > 0 {
			s := []string{}
			for _, m := range binding.Api.HttpMethods {
				s = append(s, string(m))
			}
			h.Methods(s...)
		}
		if binding.Api.HttpMethod != "" {
			h.Methods(string(binding.Api.HttpMethod))
		}

		// check the content type
		ct := string(binding.Api.ContentType)
		if _, has := marshalers[ct]; !has {
			panic(fmt.Sprintf("Bad content type: %s", ct))
		}
		if _, has := unmarshalers[ct]; !has {
			panic(fmt.Sprintf("Bad content type: %s", ct))
		}

	}
	return this.engine
}
