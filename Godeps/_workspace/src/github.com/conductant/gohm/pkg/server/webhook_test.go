package server

import (
	. "gopkg.in/check.v1"
	"testing"
)

func TestWebhook(t *testing.T) { TestingT(t) }

type TestSuiteWebhook struct {
}

var _ = Suite(&TestSuiteWebhook{})

func (suite *TestSuiteWebhook) SetUpSuite(c *C) {
}

func (suite *TestSuiteWebhook) TearDownSuite(c *C) {
}

var hooks = WebhookMap{
	"service1": EventKeyUrlMap{
		"event1": Webhook{
			Url: "http://foo.com/bar/callback1",
		},
		"event2": Webhook{
			Url: "http://foo.com/bar/callback2",
		},
	},
	"service2": EventKeyUrlMap{
		"event1": Webhook{
			Url: "http://bar.com/bar/callback1",
		},
		"event2": Webhook{
			Url: "http://bar.com/bar/callback2",
		},
	},
}

type impl int

func (this *impl) Load() *WebhookMap {
	return &hooks
}

func (suite *TestSuiteWebhook) TestWebhookSerialization(c *C) {

	bytes := hooks.ToJSON()
	c.Assert(bytes, Not(Equals), nil)
	c.Assert(len(bytes), Not(Equals), 0)

	hooks2 := WebhookMap{}

	hooks2.FromJSON(bytes)

	c.Assert(hooks, DeepEquals, hooks2)
	c.Log("json", string(hooks2.ToJSON()))
}
