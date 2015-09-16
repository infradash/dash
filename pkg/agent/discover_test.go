package agent

import (
	_docker "github.com/fsouza/go-dockerclient"
	. "github.com/infradash/dash/pkg/dash"
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

func test_container(running bool, image, domain, name string, port int64, labels map[string]string, envs ...string) *docker.Container {
	c := &docker.Container{
		Image: image,
		DockerData: &_docker.Container{
			Config: &_docker.Config{
				Env: []string{
					"DASH_DOMAIN=" + domain,
				},
			},
			State: _docker.State{Running: running},
		},
	}
	if running {
		c.Ports = []docker.Port{docker.Port{ContainerPort: port}}
	}
	if len(labels) > 0 {
		c.DockerData.Config.Labels = labels
	}
	if len(name) > 0 {
		c.Name = name
	}
	for _, env := range envs {
		c.DockerData.Config.Env = append(c.DockerData.Config.Env, env)
	}
	return c
}

func test_match(c *C, m *DiscoveryContainerMatcher, dc *docker.Container, match bool) {
	isMatch, rule := m.Match(dc)
	c.Assert(isMatch, Equals, match)
	if isMatch {
		c.Log("Rule=", rule)
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
	m.C("test.com", ServiceKey("sidekiq"), &MatchContainerRule{
		Image: docker.Image{Repository: "infradash/infradash"},
		MatchFirst: []ContainerMatchRulesUnion{
			ContainerMatchRulesUnion{
				ByContainerName: &sidekiq,
			},
		},
	})
	m.C("test.com", ServiceKey("infradash"), &MatchContainerRule{
		Image:              docker.Image{Repository: "infradash/infradash"},
		MatchContainerPort: &infradash_port,
		MatchFirst: []ContainerMatchRulesUnion{
			ContainerMatchRulesUnion{
				ByContainerLabels: map[string]string{"DASH_SERVICE": "infradash"},
			},
		},
	})

	m.C("test.com", ServiceKey("mqtt"), &MatchContainerRule{
		Image:              docker.Image{Repository: "infradash/mqtt"},
		MatchContainerPort: &mqtt_port,
	})
	m.C("test.com", ServiceKey("nginx"), &MatchContainerRule{
		Image:              docker.Image{Repository: "infradash/nginx"},
		MatchContainerPort: &nginx_https,
		MatchFirst: []ContainerMatchRulesUnion{
			ContainerMatchRulesUnion{
				ByContainerEnvironment: []string{"DASH_SERVICE=proxy"},
			},
		},
	})
	m.C("test.com", ServiceKey("nginx-http"), &MatchContainerRule{
		Image:              docker.Image{Repository: "infradash/nginx"},
		MatchContainerPort: &nginx_https,
		MatchFirst: []ContainerMatchRulesUnion{
			ContainerMatchRulesUnion{
				ByContainerEnvironment: []string{"DASH_SERVICE=https"},
			},
		},
	})
	m.C("test.com", ServiceKey("nginx-http2"), &MatchContainerRule{
		Image:              docker.Image{Repository: "infradash/nginx"},
		MatchContainerPort: &nginx_http,
		MatchFirst: []ContainerMatchRulesUnion{
			ContainerMatchRulesUnion{
				ByContainerEnvironment: []string{"DASH_SERVICE=http2"},
			},
		},
	})

	// match when no port (container not running)
	var dc *docker.Container

	dc = test_container(false, "infradash/infradash:v1.0.0-1234.5678", "test.com", "sidekiq", 0,
		map[string]string{"DASH_SERVICE": "sidekiq"})
	test_match(c, m, dc, true)

	dc = test_container(false, "infradash/infradash:v1.0.0-1234.5678", "test.com", "infradash", 3000,
		map[string]string{"DASH_SERVICE": "infradash"})
	test_match(c, m, dc, true)

	test_match(c, m, test_container(true, "infradash/infradash:v1.0.0-1234.5678", "test.com", "infradash", 3000,
		map[string]string{"DASH_SERVICE": "infradash"}), true)

	test_match(c, m, test_container(true, "infradash/infradash:v1.0.0-1234.5678", "test.com", "sidekiq", 0, nil), true)
	test_match(c, m, test_container(true, "infradash/preview:v1.0.0-1234.5678", "test.com", "p", 3000, nil), false)
	test_match(c, m, test_container(true, "infradash/infradash:v1.0.0-1234.5678", "prod.infradash.com", "z", 3000, nil), false)
	test_match(c, m, test_container(true, "infradash/infradash:v1.0.0-1234.5678", "", "", 3000, nil), false)

	test_match(c, m, test_container(true, "infradash/mqtt:latest", "test.com", "x", 1883, nil), true)

	test_match(c, m, test_container(true, "infradash/nginx", "test.com", "y", 443, nil, "DASH_SERVICE=https"), true)
	test_match(c, m, test_container(true, "infradash/nginx", "test.com", "y", 80, nil, "DASH_SERVICE=http2"), true)
	test_match(c, m, test_container(true, "infradash/nginx", "test.com", "y", 443, nil, "DASH_SERVICE=ssl2"), false)
	test_match(c, m, test_container(true, "infradash/nginx", "test.com", "y", 443, nil), false)

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
