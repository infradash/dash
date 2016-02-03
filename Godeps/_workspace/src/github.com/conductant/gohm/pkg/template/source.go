package template

import (
	"crypto/tls"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	net "net/url"
	"os"
	"strings"
)

const (
	SourceHttp   = "http"
	SourceHttps  = "https"
	SourceString = "string"
	SourceFile   = "file"
)

type SourceFunc func(context.Context, string) ([]byte, error)

var (
	protocols = map[string]SourceFunc{}
)

func init() {
	Register(SourceHttp, HttpSource)
	Register(SourceHttps, HttpSource)
	Register(SourceString, StringSource)
	Register(SourceFile, FileSource)
}

// Packages providing different backend support for this should call this
// in its `init()` function.
func Register(protocol string, source SourceFunc) {
	lock.Lock()
	defer lock.Unlock()
	protocols[protocol] = source
}

// This is the function most user will use.  Using the protocol/scheme provided
// with the url, different implementations will be invoked.  There are standard
// implementations in this package (see constants).
// Other packages using different backends can call the `Register` function to
// register different implementations.
func Source(ctx context.Context, url string) ([]byte, error) {
	parsed, err := net.Parse(url)
	if err != nil {
		return nil, err
	}
	if source, exists := protocols[parsed.Scheme]; exists {
		return source(ctx, url)
	}
	return nil, &NotSupported{parsed.Scheme}
}

type httpHeaderContextKey int

var HttpHeaderContextKey httpHeaderContextKey = 1

func ContextPutHttpHeader(ctx context.Context, header http.Header) context.Context {
	return context.WithValue(ctx, HttpHeaderContextKey, header)
}

func ContextGetHttpHeader(ctx context.Context) http.Header {
	if v, ok := (ctx.Value(HttpHeaderContextKey)).(http.Header); ok {
		return v
	}
	return nil
}

func HttpSource(ctx context.Context, url string) ([]byte, error) {
	// Don't check certificate for https
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", url, nil)

	for h, vv := range ContextGetHttpHeader(ctx) {
		for _, v := range vv {
			req.Header.Add(h, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	return content, err
}

func StringSource(ctx context.Context, url string) ([]byte, error) {
	content := url[len("string://"):]
	return []byte(content), nil
}

func FileSource(ctx context.Context, url string) ([]byte, error) {
	file := url[len("file://"):]
	switch {
	case strings.Index(file, "~") > -1:
		// expand tilda
		file = strings.Replace(file, "~", os.Getenv("HOME"), 1)
	case strings.Index(file, "./") > -1:
		// expand tilda
		if pwd, err := os.Getwd(); err == nil {
			file = strings.Replace(file, "./", pwd+"/", 1)
		} else {
			file = strings.Replace(file, "./", os.Getenv("PWD")+"/", 1)
		}
	}
	if f, err := os.Open(file); err == nil {
		defer f.Close()
		return ioutil.ReadAll(f)
	} else {
		return nil, err
	}
}
