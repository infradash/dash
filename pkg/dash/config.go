package dash

import (
	"encoding/json"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/template"
	"github.com/qorio/maestro/pkg/zk"
	"net/url"
	gotemplate "text/template"
)

type ConfigLoader struct {
	ConfigUrl string      `json:"config_url"`
	Context   interface{} `json:"-"`
}

func (this *ConfigLoader) Load(prototype interface{}, auth string, zc zk.ZK, funcs ...gotemplate.FuncMap) (loaded bool, err error) {
	if this.ConfigUrl == "" {
		glog.Infoln("No config URL. Skip.")
		return false, nil
	}

	// parse the url
	_, err = url.Parse(this.ConfigUrl)
	if err != nil {
		glog.Infoln("Config url is not valid:", this.ConfigUrl)
		return false, err
	}

	headers := map[string]string{
		"Authorization": "Bearer " + auth,
	}

	body, _, err := template.FetchUrl(this.ConfigUrl, headers)
	if err != nil {
		return false, err
	}

	// Treat the entire body as a template
	applied, err := this.applyTemplate(body, funcs...)
	if err != nil {
		return false, err
	}

	glog.Infoln("Parsing configuration:", applied)
	err = json.Unmarshal([]byte(applied), prototype)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (this *ConfigLoader) applyTemplate(body string, funcs ...gotemplate.FuncMap) (string, error) {
	if this.Context == nil {
		return body, nil
	}
	return template.ApplyTemplate(body, this.Context, funcs...)
}
