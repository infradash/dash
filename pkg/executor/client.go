package executor

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"net/http"
	"strings"
)

type Client struct {
	protocol string
	host     string
	port     int
	proxyUrl string
}

func NewClient() *Client {
	return &Client{
		protocol: "http://",
		host:     "localhost",
		port:     25658,
	}
}

func (this *Client) SetProxyUrl(u string) *Client {
	this.proxyUrl = u
	return this
}

func (this *Client) SetHost(h string) *Client {
	this.host = h
	return this
}

func (this *Client) SetPort(p int) *Client {
	this.port = p
	return this
}

func (this *Client) Https() *Client {
	this.protocol = "https://"
	return this
}

func (this *Client) GetUrl(p ...string) string {
	url := fmt.Sprintf("%s%s:%d", this.protocol, this.host, this.port)
	if len(this.proxyUrl) > 0 {
		url = fmt.Sprintf("%s/%s:%d", this.proxyUrl, this.host, this.port)
	}
	return url + strings.Join(p, "/")
}

func (this *Client) GetInfo() (*Info, error) {
	url := this.GetUrl("/v1/info")
	glog.Infoln("Calling", url)
	c := new(http.Client)

	info := Info{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&info)
	if err != nil {
		return nil, err
	}
	return &info, err
}

func (this *Client) GetPs() ([]Process, error) {
	url := this.GetUrl("/v1/ps")
	glog.Infoln("Calling", url)
	c := new(http.Client)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	result := []Process{}
	err = decoder.Decode(&result)
	if err != nil {
		return nil, err
	}

	return result, err
}

func (this *Client) RemoteKill() error {
	url := this.GetUrl("/v1/quitquitquit")
	glog.Infoln("Calling", url)
	c := new(http.Client)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	_, err = c.Do(req)
	if err != nil {
		return err
	}
	return nil
}
