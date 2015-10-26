package terraform

import (
	"bytes"
	"github.com/golang/glog"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/template"
	"io/ioutil"
	"net/http"
	"strings"
	gotemplate "text/template"
	"time"
)

func (this *Config) Validate() error {
	if this.Template.String() != "" {
		if e := check_url(this.Template.String()); e != nil {
			glog.Warningln("Bad url", this.Template)
			return e
		}
	}

	if this.Endpoint.String() == "" {
		glog.Warningln("Missing endpoint")
		return ErrBadConfig
	}

	if e := check_url(this.Endpoint.String()); e != nil {
		return e
	}
	return nil
}

func (this *Config) Execute(authToken string, context interface{}, funcs gotemplate.FuncMap) error {
	config, err := this.execute_template(authToken, context, funcs)
	if err != nil {
		return err
	}

	err = this.apply_config(authToken, config)
	if err != nil {
		glog.Warningln("Error applying config:", err)
		ticker := time.Tick(2 * time.Second)
		for {
			select {
			case <-ticker:
				glog.Infoln("Applying config:", this)
				err := this.apply_config(authToken, config)
				if err == nil {
					break
				}
			case <-this.Stop:
				break
			}
		}
	}
	return nil
}

func (this *Config) execute_template(authToken string, context interface{}, funcs gotemplate.FuncMap) ([]byte, error) {
	return template.ExecuteTemplateUrl(nil, this.Template.String(), authToken, context, funcs)
}

func (this *Config) apply_config(authToken string, config []byte) error {
	// now apply the config, based on the url of the destination
	parts := strings.Split(this.Endpoint.String(), "://")
	if len(parts) == 1 {
		return ErrBadUrl
	}
	switch parts[0] {
	case "http", "https":
		return do_post(this.Endpoint.String(), config, authToken)
	case "file":
		return do_save(parts[1], config)
	default:
		return ErrNotSupportedProtocol
	}
	return nil
}

func do_post(url string, body []byte, authToken string) error {
	client := &http.Client{}
	post, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	post.Header.Add("Authorization", "Bearer "+authToken)
	post.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(post)
	if err != nil {
		return err
	}
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		return ErrPostFailed
	}
}

func do_save(path string, body []byte) error {
	return ioutil.WriteFile(path, []byte(body), 0777)
}
