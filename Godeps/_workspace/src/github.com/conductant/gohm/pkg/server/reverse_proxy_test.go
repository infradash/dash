package server

import (
	"github.com/conductant/gohm/pkg/testutil"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
	"net/http"
	"sync"
	"testing"
)

func TestReverseProxy(t *testing.T) { TestingT(t) }

type TestSuiteReverseProxy struct {
	proxyPort      int
	proxyStop      chan<- int
	proxyStopped   <-chan error
	backendPort    int
	backendStop    chan<- int
	backendStopped <-chan error

	proxyInvokes   int
	backendInvokes int

	lock sync.Mutex // in case tests are run in parallel that can mess up the counts
}

var _ = Suite(&TestSuiteReverseProxy{})

func (suite *TestSuiteReverseProxy) SetUpSuite(c *C) {
	// Set up a backend
	suite.backendPort = 7891
	suite.backendStop, suite.backendStopped = NewService().WithAuth(DisableAuth()).ListenPort(suite.backendPort).
		Route(ServiceMethod{
		UrlRoute:   "/test/get",
		HttpMethod: GET,
		AuthScope:  AuthScopeNone,
	}).To(func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
		c.Log("GET called")
		suite.backendInvokes++
	}).Start()

	suite.proxyPort = 7712
	suite.proxyStop, suite.proxyStopped = NewService().WithAuth(DisableAuth()).ListenPort(suite.proxyPort).
		Route(ServiceMethod{
		UrlRoute:   "/{host_port}/{url:.*}",
		HttpMethod: GET,
		AuthScope:  AuthScopeNone,
	}).To(func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
		suite.proxyInvokes++
		hostPort := GetUrlParameter(req, "host_port")
		NewReverseProxy().SetForwardHostPort(hostPort).Strip("/"+hostPort).ServeHTTP(resp, req)
	}).Start()
}

func (suite *TestSuiteReverseProxy) TearDownSuite(c *C) {
	suite.proxyStop <- 0
	<-suite.proxyStopped
	suite.backendStop <- 0
	<-suite.backendStopped
}

func (suite *TestSuiteReverseProxy) TestSetValues(c *C) {
	c.Assert(NewReverseProxy().SetForwardHostPort("10.20.5.128:8078").reverseProxyUrl(),
		Equals, "http://10.20.5.128:8078")
	c.Assert(NewReverseProxy().SetForwardHostPort(":8078").reverseProxyUrl(),
		Equals, "http://127.0.0.1:8078")
	c.Assert(NewReverseProxy().SetForwardHostPort(":8078").SetForwardPrefix("foo").reverseProxyUrl(),
		Equals, "http://127.0.0.1:8078/foo")
	c.Assert(NewReverseProxy().SetForwardHostPort(":8078").SetForwardPrefix("/foo").reverseProxyUrl(),
		Equals, "http://127.0.0.1:8078/foo")
	c.Assert(NewReverseProxy().SetForwardHostPort(":8078").SetForwardPrefix("/foo/").reverseProxyUrl(),
		Equals, "http://127.0.0.1:8078/foo/")
	c.Assert(NewReverseProxy().SetForwardHostPort(":8078").SetForwardPrefix("/foo/").
		SetForwardScheme("https").reverseProxyUrl(),
		Equals, "https://127.0.0.1:8078/foo/")
	c.Assert(NewReverseProxy().SetForwardHostPort("10.20.1.1").SetForwardPrefix("/foo/").
		SetForwardScheme("https").reverseProxyUrl(),
		Equals, "https://10.20.1.1/foo/")
	c.Assert(NewReverseProxy().SetForwardHostPort("10.20.1.1").SetForwardPort(8080).SetForwardPrefix("/foo/").
		SetForwardScheme("https").reverseProxyUrl(),
		Equals, "https://10.20.1.1:8080/foo/")
}

func (suite *TestSuiteReverseProxy) TestReverseProxy(c *C) {
	suite.lock.Lock()
	defer suite.lock.Unlock()

	suite.backendInvokes = 0
	suite.proxyInvokes = 0

	testutil.Get(c, "http://localhost:7712/:7891/test/get",
		func(resp *http.Response, body []byte) {
			c.Assert(resp.StatusCode, Equals, http.StatusOK)
		})
	c.Assert(suite.proxyInvokes, Equals, suite.backendInvokes)

	testutil.Get(c, "http://localhost:7712/127.0.0.1:7891/test/get",
		func(resp *http.Response, body []byte) {
			c.Assert(resp.StatusCode, Equals, http.StatusOK)
		})
	c.Assert(suite.proxyInvokes, Equals, suite.backendInvokes)

	testutil.Get(c, "http://localhost:7712/127.0.0.1:7891/test/bad/not/exist",
		func(resp *http.Response, body []byte) {
			c.Assert(resp.StatusCode, Equals, http.StatusNotFound)
		})
	c.Assert(suite.proxyInvokes, Equals, suite.backendInvokes+1)
}
