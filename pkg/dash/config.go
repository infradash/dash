package dash

import (
	"bytes"
	"encoding/json"
	"github.com/golang/glog"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
)

type ConfigLoader struct {
	SourceUrl string      `json:"config_source_url"`
	Context   interface{} `json:"-"`
}

func (this *ConfigLoader) Load(prototype interface{}) (err error) {
	if this.SourceUrl == "" {
		return nil
	}

	var body string
	if strings.Index(this.SourceUrl, "file://") == 0 {
		file := this.SourceUrl[len("file://"):]
		glog.Infoln("Loading from file", file)
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		if buff, err := ioutil.ReadAll(f); err != nil {
			return err
		} else {
			body = string(buff)
		}
	} else {
		if body, _, err = FetchUrl(this.SourceUrl); err != nil {
			return err
		}
	}
	if applied, err := this.applyTemplate(body); err != nil {
		return err
	} else {
		glog.Infoln("Parsing configuration:", applied)
		return json.Unmarshal([]byte(applied), prototype)
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
