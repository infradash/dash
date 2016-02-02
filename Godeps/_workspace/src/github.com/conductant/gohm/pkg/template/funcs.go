package template

import (
	"golang.org/x/net/context"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
)

var (
	NullTemplate string = ""
	NullContent  string = ""

	lock      sync.Mutex
	userFuncs = map[string]func(context.Context) interface{}{}
)

func RegisterFunc(name string, generator func(context.Context) interface{}) {
	lock.Lock()
	defer lock.Unlock()
	userFuncs[name] = generator
}

func DefaultFuncMap(ctx context.Context) template.FuncMap {
	fm := template.FuncMap{}
	for k, v := range userFuncs {
		fm[k] = v(ctx)
	}
	return fm
}

func MergeFuncMaps(a, b template.FuncMap) template.FuncMap {
	merged := template.FuncMap{}
	for k, v := range a {
		merged[k] = v
	}
	for k, v := range b {
		merged[k] = v
	}
	return merged
}

func init() {
	RegisterFunc("host", ParseHost)
	RegisterFunc("port", ParsePort)
	RegisterFunc("inline", ContentInline)
	RegisterFunc("file", ContentToFile)
	RegisterFunc("sh", ExecuteShell)
}

func ParseHost(ctx context.Context) interface{} {
	return func(hostport string) (string, error) {
		host, _, err := net.SplitHostPort(hostport)
		return host, err
	}
}

func ParsePort(ctx context.Context) interface{} {
	return func(hostport string) (int, error) {
		_, port, err := net.SplitHostPort(hostport)
		if err != nil {
			return 0, err
		}
		return strconv.Atoi(port)
	}
}

// Fetch the url and write content inline
// ex) {{ inline "http://file/here" }}
func ContentInline(ctx context.Context) interface{} {
	return func(uri string) (string, error) {
		data := ContextGetTemplateData(ctx)
		applied, err := Apply([]byte(uri), data)
		if err != nil {
			return NullTemplate, err
		}
		url := string(applied)
		content, err := Source(ctx, url)
		if err != nil {
			return NullTemplate, err
		}
		return string(content), nil
	}
}

func ContentToFile(ctx context.Context) interface{} {
	return func(uri string, opts ...interface{}) (string, error) {
		data := ContextGetTemplateData(ctx)
		applied, err := Apply([]byte(uri), data)
		if err != nil {
			return NullTemplate, err
		}
		url := string(applied)
		content, err := Source(ctx, url)
		if err != nil {
			return NullTemplate, err
		}

		destination := os.TempDir()
		fileMode := os.FileMode(0644)

		// The optional param ordering not important. We check by type.
		// String -> destination path
		// Int -> file mode
		for _, opt := range opts {
			switch opt.(type) {
			case int:
				fileMode = os.FileMode(opt.(int))
			case string:
				if applied, err = Apply([]byte(opt.(string)), data); err != nil {
					return NullContent, err
				} else {
					destination = string(applied)
				}
				// Also expands shell path variables
				switch {
				case strings.Index(destination, "~") > -1:
					// expand tilda
					destination = strings.Replace(destination, "~", os.Getenv("HOME"), 1)
				case strings.Index(destination, "./") > -1:
					// expand tilda
					destination = strings.Replace(destination, "./", os.Getenv("PWD")+"/", 1)
				}
			}
		}

		parent := filepath.Dir(destination)
		fi, err := os.Stat(parent)
		if err != nil {
			switch {
			case os.IsNotExist(err):
				err = os.MkdirAll(parent, fileMode)
				if err != nil {
					return NullContent, err
				}
			default:
				return NullContent, err
			}
		}
		// read again after we created the directories
		fi, err = os.Stat(destination)
		if err == nil && fi.IsDir() {
			// build the name because we provided only a directory path
			destination = filepath.Join(destination, filepath.Base(string(url)))
		}

		err = ioutil.WriteFile(destination, []byte(content), fileMode)
		if err != nil {
			return NullTemplate, err
		}
		return destination, nil
	}
}
