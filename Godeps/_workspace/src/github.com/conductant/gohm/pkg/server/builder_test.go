package server

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	. "gopkg.in/check.v1"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func TestBuilder(t *testing.T) { TestingT(t) }

type TestSuiteBuilder struct {
}

var _ = Suite(&TestSuiteBuilder{})

func (suite *TestSuiteBuilder) SetUpSuite(c *C) {
}

func (suite *TestSuiteBuilder) TearDownSuite(c *C) {
}

var (
	test_method1 = ServiceMethod{
		UrlRoute:   "/method1",
		HttpMethod: GET,
	}
	test_method2 = ServiceMethod{
		UrlRoute:   "/method2",
		HttpMethod: GET,
	}
)

func testFunc1(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	glog.Infoln("testFunc1 called")
	sm := ApiForScope(ctx)
	glog.Infoln("Api=", sm, sm.Equals(test_method1))
	if !sm.Equals(test_method1) {
		panic(fmt.Errorf("Does not match test_method1"))
	}
}

func testFunc2(ctx context.Context, resp http.ResponseWriter, req *http.Request) {
	glog.Infoln("testFunc2 called")
	sm := ApiForScope(ctx)
	glog.Infoln("Api=", sm, test_method2.Equals(sm))
	if !sm.Equals(test_method2) {
		panic(fmt.Errorf("Does not match test_method2"))
	}
}

func (suite *TestSuiteBuilder) TestBuild(c *C) {
	engine := NewService().WithAuth(DisableAuth()).
		Route(test_method1).To(testFunc1).
		Route(test_method2).To(testFunc2).Build()
	c.Log(engine)

	engine2 := NewService().WithAuth(DisableAuth()).
		Route(test_method1).To(testFunc1).
		Route(test_method2).To(testFunc2).Build()
	c.Log(engine2)
}

func test_get(c *C, url string) {
	resp, err := http.Get(url)
	c.Assert(err, IsNil)
	body, err := ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)
	resp.Body.Close()
	c.Log("response=", string(body))
}

func (suite *TestSuiteBuilder) TestRun(c *C) {
	stop, stopped := NewService().WithAuth(DisableAuth()).
		Route(test_method1).To(testFunc1).
		Route(test_method2).To(testFunc2).
		Start()

	go func() {
		<-stopped
	}()

	time.Sleep(1 * time.Second)

	test_get(c, "http://localhost:8080/method1")
	test_get(c, "http://localhost:8080/method2")

	time.Sleep(1 * time.Second)
	stop <- 1
}
