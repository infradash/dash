package terraform

import (
	"bytes"
	. "github.com/infradash/dash/pkg/dash"
	"github.com/qorio/maestro/pkg/template"
	"io/ioutil"
	"net/http"
	"strings"
	gotemplate "text/template"
)

func (this *Config) Validate() error {
	switch {

	case this.Template.String() == "":
		return ErrBadConfig
	case this.Endpoint.String() == "":
		return ErrBadConfig
	}

	if e := check_url(this.Template.String()); e != nil {
		return e
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

	return this.apply_config(authToken, config)
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
