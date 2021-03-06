package resource

import (
	"fmt"
	"github.com/conductant/gohm/pkg/auth"
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

func TestResource(t *testing.T) { TestingT(t) }

type TestSuiteResource struct {
	port         int
	template     string
	stop         chan<- int
	stopped      <-chan error
	templateFile string
}

var templateFileContent = "this is some test template written to disk"

var _ = Suite(&TestSuiteResource{port: 7981})

func (suite *TestSuiteResource) SetUpSuite(c *C) {
	suite.stop, suite.stopped = server.NewService().
		ListenPort(suite.port).
		WithAuth(server.Auth{VerifyKeyFunc: testutil.PublicKeyFunc}.Init()).
		Route(server.Endpoint{UrlRoute: "/template", HttpMethod: server.GET, AuthScope: server.AuthScopeNone}).
		To(func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(suite.template))
	}).
		Route(server.Endpoint{UrlRoute: "/secure", HttpMethod: server.GET, AuthScope: server.AuthScope("secure")}).
		To(func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(suite.template))
	}).Start()

	suite.templateFile = os.TempDir() + "/test-template"
	err := ioutil.WriteFile(suite.templateFile, []byte(templateFileContent), 0644)
	c.Assert(err, IsNil)
}

func (suite *TestSuiteResource) TearDownSuite(c *C) {
	suite.stop <- 1
	<-suite.stopped
	os.Remove(suite.templateFile)
}

func (suite *TestSuiteResource) TestStringResource(c *C) {
	source := "string://{.FirstName}{.LastName}"
	ctx := context.Background()
	t, err := Fetch(ctx, source)
	c.Assert(err, IsNil)
	c.Assert(string(t), DeepEquals, "{.FirstName}{.LastName}")
}

func (suite *TestSuiteResource) TestFileResource(c *C) {
	source := "file://" + suite.templateFile
	ctx := context.Background()
	t, err := Fetch(ctx, source)
	c.Assert(err, IsNil)
	c.Assert(string(t), DeepEquals, templateFileContent)
}

func (suite *TestSuiteResource) TestHttpResource(c *C) {
	suite.template = "this-template"
	source := fmt.Sprintf("http://localhost:%d/template", suite.port)
	ctx := context.Background()
	t, err := Fetch(ctx, source)
	c.Assert(err, IsNil)
	c.Assert(string(t), DeepEquals, suite.template)
}

func (suite *TestSuiteResource) TestHttpResourceWithToken(c *C) {
	suite.template = "secure-template"
	source := fmt.Sprintf("http://localhost:%d/secure", suite.port)

	token := auth.NewToken(1*time.Hour).Add("secure", 1)
	ctx := context.Background()
	header := http.Header{}
	token.SetHeader(header, testutil.PrivateKeyFunc)
	ctx = ContextPutHttpHeader(ctx, header)

	t, err := Fetch(ctx, source)
	c.Assert(err, IsNil)
	c.Assert(string(t), DeepEquals, suite.template)
}
