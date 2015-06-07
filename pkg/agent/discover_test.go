package agent

import (
	. "github.com/infradash/dash/pkg/dash"
	_docker "github.com/fsouza/go-dockerclient"
	"github.com/qorio/maestro/pkg/docker"
	. "gopkg.in/check.v1"
	"testing"
)

func TestDiscover(t *testing.T) { TestingT(t) }

type TestSuiteDiscover struct {
}

var _ = Suite(&TestSuiteDiscover{})

func (suite *TestSuiteDiscover) SetUpSuite(c *C) {
}

func (suite *TestSuiteDiscover) TearDownSuite(c *C) {
}

func test_container(image, domain, name string, port int64) *docker.Container {
	c := &docker.Container{
		Image: image,
		DockerData: &_docker.Container{
			Config: &_docker.Config{
				Env: []string{
					domain,
				},
			},
		},
		Ports: []docker.Port{
			docker.Port{
				ContainerPort: port,
			},
		},
	}
	if len(name) > 0 {
		c.Name = name
	}
	return c
}

func test_match(c *C, m *DiscoveryContainerMatcher, dc *docker.Container, match bool) {
	isMatch, rule := m.Match(dc)
	c.Assert(isMatch, Equals, match)
	if isMatch {
		c.Assert(rule, Not(Equals), nil)
	}
}

func (suite *TestSuiteDiscover) TestContainerMatch(c *C) {

	sidekiq := "sidekiq"
	infradash_port := 3000
	mqtt_port := 1883
	nginx_https := 443
	nginx_http := 80

	m := new(DiscoveryContainerMatcher).Init()
	m.C("test.infradash.com", ServiceKey("sidekiq"), &WatchContainerSpec{
		Image:              docker.Image{Repository: "infradash/infradash"},
		MatchContainerName: &sidekiq,
	})
	m.C("test.infradash.com", ServiceKey("infradash"), &WatchContainerSpec{
		Image:              docker.Image{Repository: "infradash/infradash"},
		MatchContainerPort: &infradash_port,
	})
	m.C("test.infradash.com", ServiceKey("mqtt"), &WatchContainerSpec{
		Image:              docker.Image{Repository: "infradash/mqtt"},
		MatchContainerPort: &mqtt_port,
	})
	m.C("test.infradash.com", ServiceKey("nginx"), &WatchContainerSpec{
		Image:              docker.Image{Repository: "infradash/nginx"},
		MatchContainerPort: &nginx_https,
	})
	m.C("test.infradash.com", ServiceKey("nginx-http"), &WatchContainerSpec{
		Image:              docker.Image{Repository: "infradash/nginx"},
		MatchContainerPort: &nginx_http,
	})

	test_match(c, m, test_container("infradash/infradash:v1.0.0-1234.5678", "DASH_DOMAIN=test.infradash.com", "a", 3000), true)
	test_match(c, m, test_container("infradash/infradash:v1.0.0-1234.5678", "DASH_DOMAIN=test.infradash.com", "sidekiq", 0), true)
	test_match(c, m, test_container("infradash/preview:v1.0.0-1234.5678", "DASH_DOMAIN=test.infradash.com", "p", 3000), false)
	test_match(c, m, test_container("infradash/infradash:v1.0.0-1234.5678", "DASH_DOMAIN=prod.infradash.com", "z", 3000), false)
	test_match(c, m, test_container("infradash/infradash:v1.0.0-1234.5678", "", "", 3000), false)
	test_match(c, m, test_container("infradash/mqtt:latest", "DASH_DOMAIN=test.infradash.com", "x", 1883), true)
	test_match(c, m, test_container("infradash/nginx", "DASH_DOMAIN=test.infradash.com", "y", 443), true)
}

func (suite *TestSuiteDiscover) TestImageMatch(c *C) {
	image := "infradash/infradash:v1.0.0-1234.567"
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/infradash",
	}), Equals, true)
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/bonker",
	}), Equals, false)
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/infradash",
		Tag:        "v1.0.1",
	}), Equals, false)
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/infradash",
		Tag:        "v1.0.0-1234.567",
	}), Equals, true)

	image = "infradash/infradash"
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/infradash",
	}), Equals, true)
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/bonker",
	}), Equals, false)
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/infradash",
		Tag:        "v1.0.1",
	}), Equals, false)
	c.Assert(ImageMatch(image, &docker.Image{
		Repository: "infradash/infradash",
		Tag:        "v1.0.0-1234.567",
	}), Equals, false)
}
