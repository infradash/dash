package dash

import (
	"bytes"
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"text/template"
)

type ConfigLoader struct {
	SourceUrl string      `json:"config_source_url"`
	Context   interface{} `json:"-"`
}

func (this *ConfigLoader) Load(prototype interface{}) (loaded bool, err error) {
	if this.SourceUrl == "" {
		glog.Infoln("No config URL. Skip.")
		return false, nil
	}

	// parse the url
	_, err = url.Parse(this.SourceUrl)
	if err != nil {
		glog.Infoln("Config url is not valid:", this.SourceUrl)
		return false, err
	}

	var body string
	if strings.Index(this.SourceUrl, "file://") == 0 {
		file := this.SourceUrl[len("file://"):]
		glog.Infoln("Loading from file", file)
		f, err := os.Open(file)
		if err != nil {
			return false, err
		}
		if buff, err := ioutil.ReadAll(f); err != nil {
			return false, err
		} else {
			body = string(buff)
		}
	} else {
		if body, _, err = FetchUrl(this.SourceUrl); err != nil {
			return false, err
		}
	}
	if applied, err := this.applyTemplate(body); err != nil {
		return false, err
	} else {
		glog.Infoln("Parsing configuration:", applied)
		err2 := json.Unmarshal([]byte(applied), prototype)
		if err2 != nil {
			return false, err2
		} else {
			return true, nil
		}
	}
}

func (this *ConfigLoader) applyTemplate(body string) (string, error) {
	if this.Context == nil {
		return body, nil
	}

	t, err := template.New(body).Parse(body)
	if err != nil {
		return "", err
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, this.Context); err != nil {
		return "", err
	} else {
		return buff.String(), nil
	}
}
