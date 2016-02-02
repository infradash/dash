package proxy

import (
	"github.com/conductant/gohm/pkg/server"
	"github.com/conductant/gohm/pkg/template"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"golang.org/x/net/context"
	"net/http"
	"time"
)

// More accurately this is a reverse proxy.
type Proxy struct {
	ProxyConfig

	Initializer *ConfigLoader `json:"-"`
}

func mustNot(err error) {
	if err != nil {
		panic(err)
	}
}

func (this *Proxy) Run() error {

	if this.Initializer == nil {
		return ErrNoConfig
	}

	// We don't want the application of template to wipe out Domain, Service, etc. variables
	// So escape them.
	this.Initializer.Context = EscapeVars(ConfigVariables...)
	this.ProxyConfig = DefaultProxyConfig

	loaded := false
	var err error
	for {
		loaded, err = this.Initializer.Load(this, "", nil)

		if !loaded || err != nil {
			glog.Infoln("Wait then retry:", err)
			time.Sleep(2 * time.Second)

		} else {
			break
		}
	}

	ctx := context.Background()

	_, stopped := server.NewService().
		WithAuth(this.getAuth(ctx)).ListenPort(this.Port).
		Route(
		server.ServiceMethod{
			UrlRoute:   "/{host_port}/{url:.*}",
			HttpMethod: server.GET,
			AuthScope:  server.AuthScope(this.AuthScopeGET),
		}).To(this.HandleGet).
		Route(
		server.ServiceMethod{
			UrlRoute:   "/{host_port}/{url:.*}",
			HttpMethod: server.POST,
			AuthScope:  server.AuthScope(this.AuthScopePOST),
		}).To(this.HandlePost).
		Start()

	err = <-stopped
	return err
}

func (this *Proxy) getAuth(ctx context.Context) server.AuthManager {
	if this.PublicKeyUrl != "" {
		glog.Infoln("Using public key for token auth:", this.PublicKeyUrl)
		return server.Auth{
			VerifyKeyFunc: func() []byte {
				bytes, err := template.Source(ctx, this.PublicKeyUrl)
				mustNot(err)
				return bytes
			},
		}.Init()
	}
	glog.Infoln("Disabled token auth")
	return server.DisableAuth()
}

func (this *Proxy) HandleGet(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	this.proxy(ctx, resp, req)
}
func (this *Proxy) HandlePost(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	this.proxy(ctx, resp, req)
}

func (this *Proxy) proxy(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	hostPort := server.GetUrlParameter(req, "host_port")
	url := server.GetUrlParameter(req, "url")
	glog.V(100).Infoln(req.Method, req.URL, "HostPort=", hostPort, "forward=", url)
	server.NewReverseProxy().SetForwardHostPort(hostPort).Strip("/"+hostPort).ServeHTTP(resp, req)
}
