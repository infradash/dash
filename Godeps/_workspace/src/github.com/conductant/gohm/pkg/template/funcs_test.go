package template

import (
	"fmt"
	"github.com/conductant/gohm/pkg/auth"
	"github.com/conductant/gohm/pkg/resource"
	"github.com/conductant/gohm/pkg/server"
	"github.com/conductant/gohm/pkg/testutil"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestFuncs(t *testing.T) { TestingT(t) }

type TestSuiteFuncs struct {
	port        int
	content     string
	stop        chan<- int
	stopped     <-chan error
	contentFile string
}

var contentFileContent = "this is some test content"
var _ = Suite(&TestSuiteFuncs{port: 7982})

func (suite *TestSuiteFuncs) SetUpSuite(c *C) {
	suite.stop, suite.stopped = server.NewService().
		ListenPort(suite.port).
		WithAuth(server.Auth{VerifyKeyFunc: testutil.PublicKeyFunc}.Init()).
		Route(server.Endpoint{UrlRoute: "/content", HttpMethod: server.GET, AuthScope: server.AuthScopeNone}).
		To(func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(suite.content))
	}).
		Route(server.Endpoint{UrlRoute: "/secure", HttpMethod: server.GET, AuthScope: server.AuthScope("secure")}).
		To(func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(suite.content))
	}).Start()

	suite.contentFile = os.TempDir() + "/test-content"
	err := ioutil.WriteFile(suite.contentFile, []byte(contentFileContent), 0644)
	c.Assert(err, IsNil)
}

func (suite *TestSuiteFuncs) TearDownSuite(c *C) {
	suite.stop <- 1
	<-suite.stopped
	os.Remove(suite.contentFile)
}

func (suite *TestSuiteFuncs) TestParseHost(c *C) {
	f := ParseHost(context.Background())
	parseHost, ok := f.(func(string) (string, error))
	c.Assert(ok, Equals, true)

	host, err := parseHost("localhost:5050")
	c.Assert(err, IsNil)
	c.Assert(host, Equals, "localhost")

	host, err = parseHost("google.com:5050")
	c.Assert(err, IsNil)
	c.Assert(host, Equals, "google.com")

	host, err = parseHost("10.30.0.23:5050")
	c.Assert(err, IsNil)
	c.Assert(host, Equals, "10.30.0.23")
}

func (suite *TestSuiteFuncs) TestParsePort(c *C) {
	f := ParsePort(context.Background())
	parsePort, ok := f.(func(string) (int, error))
	c.Assert(ok, Equals, true)

	port, err := parsePort("localhost:5050")
	c.Assert(err, IsNil)
	c.Assert(port, Equals, 5050)

	port, err = parsePort("google.com:5050")
	c.Assert(err, IsNil)
	c.Assert(port, Equals, 5050)

	port, err = parsePort("10.30.0.23:5050")
	c.Assert(err, IsNil)
	c.Assert(port, Equals, 5050)
}

func (suite *TestSuiteFuncs) TestContentInline(c *C) {
	f := ContentInline(context.Background())
	inline, ok := f.(func(string) (string, error))
	c.Assert(ok, Equals, true)

	in := "this is a test.\n"
	out, err := inline("string://" + in)
	c.Assert(err, IsNil)
	c.Assert(out, Equals, in)
}

func (suite *TestSuiteFuncs) TestContentToFileWithAuthToken(c *C) {
	token := auth.NewToken(1*time.Hour).Add("secure", 1)
	header := http.Header{}
	token.SetHeader(header, testutil.PrivateKeyFunc)
	ctx := resource.ContextPutHttpHeader(context.Background(), header)

	// set up the content
	suite.content = "this is a test"

	f := ContentToFile(ctx)
	tofile, ok := f.(func(string, ...interface{}) (string, error))
	c.Assert(ok, Equals, true)

	path, err := tofile(fmt.Sprintf("http://localhost:%d/secure", suite.port))
	c.Assert(err, IsNil)
	c.Log("path=", path)

	bytes, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, suite.content)
}

func (suite *TestSuiteFuncs) TestContentToFileWithNoToken(c *C) {

	ctx := context.Background()

	// set up the content
	suite.content = "this is a test without token to open endpoint."

	f := ContentToFile(ctx)
	tofile, ok := f.(func(string, ...interface{}) (string, error))
	c.Assert(ok, Equals, true)

	path, err := tofile(fmt.Sprintf("http://localhost:%d/content", suite.port))
	c.Assert(err, IsNil)
	c.Log("path=", path)

	bytes, err := ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, suite.content)

	// Test with filepath
	fn := "/tmp/mytest"
	os.Remove(fn)
	path, err = tofile(fmt.Sprintf("http://localhost:%d/content", suite.port), fn)
	c.Assert(err, IsNil)
	c.Log("path=", path)
	c.Assert(path, Equals, fn)
	bytes, err = ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, suite.content)

	// Test with permission
	fn = "/tmp/mytest2"
	os.Remove(fn)
	path, err = tofile(fmt.Sprintf("http://localhost:%d/content", suite.port), fn, 0600)
	c.Assert(err, IsNil)
	c.Log("path=", path)

	// permission
	df, err := os.Open(path)
	c.Assert(err, IsNil)
	s, err := df.Stat()
	c.Assert(err, IsNil)
	c.Assert(s.Mode(), Equals, os.FileMode(0600))
	// now read
	bytes, err = ioutil.ReadFile(path)
	c.Assert(err, IsNil)
	c.Assert(string(bytes), Equals, suite.content)

}
