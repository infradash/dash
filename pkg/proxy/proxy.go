package proxy

import (
	"github.com/conductant/gohm/pkg/resource"
	"github.com/conductant/gohm/pkg/server"
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

	glog.Infoln("Starting proxy with config", this)

	ctx := context.Background()

	_, stopped := server.NewService().
		WithAuth(this.getAuth(ctx)).ListenPort(this.Port).
		Route(
		server.Endpoint{
			UrlRoute: "/{host_port}/{url:.*}",
			HttpMethods: []server.HttpMethod{
				server.GET,
				server.POST,
				server.PUT,
				server.PATCH,
				server.DELETE,
			},
			AuthScope: server.AuthScope(this.AuthScope),
		}).To(
		func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
			hostPort := server.GetUrlParameter(req, "host_port")
			url := server.GetUrlParameter(req, "url")
			glog.V(100).Infoln(req.Method, req.URL, "HostPort=", hostPort, "forward=", url)
			server.NewReverseProxy().SetForwardHostPort(hostPort).Strip("/"+hostPort).ServeHTTP(resp, req)
		}).
		Start()

	err = <-stopped
	return err
}

func (this *Proxy) getAuth(ctx context.Context) server.AuthManager {
	if this.PublicKeyUrl != "" {
		glog.Infoln("Using public key for token auth:", this.PublicKeyUrl)
		return server.Auth{
			VerifyKeyFunc: func() []byte {
				bytes, err := resource.Fetch(ctx, this.PublicKeyUrl)
				mustNot(err)
				return bytes
			},
		}.Init()
	}
	glog.Infoln("Disabled token auth")
	return server.DisableAuth()
}
