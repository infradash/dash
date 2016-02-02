package template

import (
	"github.com/conductant/gohm/pkg/auth"
	"github.com/conductant/gohm/pkg/server"
	"github.com/conductant/gohm/pkg/testutil"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
	"net/http"
	"testing"
	"time"
)

func TestTemplate(t *testing.T) { TestingT(t) }

type TestSuiteTemplate struct {
	port     int
	template string // template content to serve
	scope    string
	stop     chan<- int
	stopped  <-chan error
}

var _ = Suite(&TestSuiteTemplate{port: 7983})

func (suite *TestSuiteTemplate) SetUpSuite(c *C) {
	suite.stop, suite.stopped = server.NewService().
		ListenPort(suite.port).
		WithAuth(server.Auth{VerifyKeyFunc: testutil.PublicKeyFunc}.Init()).
		Route(server.ServiceMethod{UrlRoute: "/secure", HttpMethod: server.GET, AuthScope: server.AuthScope("secure")}).
		To(func(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
		resp.Write([]byte(suite.template))
	}).Start()
}

func (suite *TestSuiteTemplate) TearDownSuite(c *C) {
	suite.stop <- 1
	<-suite.stopped
}

func (suite *TestSuiteTemplate) TestTemplateToFile(c *C) {
	token := auth.NewToken(1*time.Hour).Add("secure", 1)
	header := http.Header{}
	token.SetHeader(header, testutil.PrivateKeyFunc)

	suite.template = "My name is {{.Name}} and I am {{.Age}} years old."

	// The url is a template too.
	url := "http://localhost:{{.port}}/secure"

	data := map[string]interface{}{
		"Name": "test",
		"Age":  20,
		"port": suite.port,
	}
	ctx := ContextPutTemplateData(ContextPutHttpHeader(context.Background(), header), data)

	text, err := Execute(ctx, url)
	c.Assert(err, IsNil)
	c.Log(string(text))
	c.Assert(string(text), Equals, "My name is test and I am 20 years old.")
}
