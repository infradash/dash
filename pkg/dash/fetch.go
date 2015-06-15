package dash

import (
	"bytes"
	"crypto/tls"
	"errors"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/zk"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// // Wrapper to allow loading from zk
// func FetchUrl2(zc zk.ZK) func(string) (string, string, error) {
// 	return func(urlRef string) (string, string, error) {
// 		glog.Infoln("Fetching from", urlRef)

// 		if strings.Index(urlRef, "env://") == 0 {
// 		} else {
// 			return FetchUrl(urlRef)
// 		}
// 	}
// }

func FetchUrl(urlRef string, headers map[string]string, zc ...zk.ZK) (body string, mime string, err error) {
	switch {
	case strings.Index(urlRef, "http://") == 0, strings.Index(urlRef, "https://") == 0:
		url, err := url.Parse(urlRef)
		if err != nil {
			return "", "", err
		}

		// don't check certificate for https
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}
		req, err := http.NewRequest("GET", url.String(), nil)

		for h, v := range headers {
			req.Header.Add(h, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			return "", "", err
		}
		content, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return "", "", err
		}
		return string(content), resp.Header.Get("Content-Type"), nil

	case strings.Index(urlRef, "file://") == 0:
		file := urlRef[len("file://"):]
		f, err := os.Open(file)
		if err != nil {
			return "", "", err
		}
		defer f.Close()
		if buff, err := ioutil.ReadAll(f); err != nil {
			return "", "", err
		} else {
			return string(buff), "text/plain", nil
		}
	case strings.Index(urlRef, "env://") == 0:
		if len(zc) == 0 {
			return "", "", errors.New("no-zk-client")
		}
		path := urlRef[len("env://"):]
		n, err := zc[0].Get(path)
		glog.Infoln("Content from environment: Path=", urlRef, "Err=", err)
		if err != nil {
			return "", "", err
		} else {
			return n.GetValueString(), "text/plain", nil
		}
	}
	return "", "", ErrNotSupportedProtocol
}

func ExecuteTemplateUrl(zc zk.ZK, url string, authToken string, data interface{}) ([]byte, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + authToken,
	}

	config_template_text, _, err := FetchUrl(url, headers, zc)
	if err != nil {
		glog.Warningln("Error fetching template:", err)
		return nil, err
	}

	funcMap := template.FuncMap{
		"inline": func(url string) string {
			content, _, err := FetchUrl(url, headers, zc)
			if err != nil {
				return "err:" + err.Error()
			}
			return content
		},
		"file": func(url string, dir ...string) string {
			content, _, err := FetchUrl(url, headers, zc)
			if err != nil {
				return "err:" + err.Error()
			}
			// Write to local file and return the path
			parent := os.TempDir()
			if len(dir) > 0 {
				parent = dir[0]
			}
			path := filepath.Join(parent, filepath.Base(url))
			err = ioutil.WriteFile(path, []byte(content), 0777)
			glog.Infoln("Written", len([]byte(content)), " bytes to", path, "Err=", err)
			if err != nil {
				return "err:" + err.Error()
			}
			return path
		},
	}

	config_template, err := template.New(url).Funcs(funcMap).Parse(config_template_text)
	if err != nil {
		glog.Warningln("Error parsing template", url, err)
		return nil, err
	}

	var buff bytes.Buffer
	err = config_template.Execute(&buff, data)
	return buff.Bytes(), err
}
