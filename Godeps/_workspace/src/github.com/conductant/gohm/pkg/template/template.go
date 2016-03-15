package template

import (
	"bytes"
	"github.com/conductant/gohm/pkg/resource"
	"golang.org/x/net/context"
	"hash/fnv"
	"strconv"
	"strings"
	"text/template"
)

type templateDataContextKey int

var (
	TemplateDataContextKey templateDataContextKey = 1
)

func ContextPutTemplateData(ctx context.Context, data interface{}) context.Context {
	return context.WithValue(ctx, TemplateDataContextKey, data)
}
func ContextGetTemplateData(ctx context.Context) interface{} {
	return ctx.Value(TemplateDataContextKey)
}

func GetKeyForTemplate(tmpl []byte) string {
	hash := fnv.New64a()
	hash.Write(tmpl)
	return strconv.FormatUint(hash.Sum64(), 16)
}

// Generic Apply template.  This is simple convenince wrapper that generates a hash key
// for the template name based on the template content itself.
func Apply(tmpl []byte, data interface{}, funcs ...template.FuncMap) ([]byte, error) {
	fm := template.FuncMap{}
	for _, opt := range funcs {
		fm = MergeFuncMaps(fm, opt)
	}
	t := template.New(GetKeyForTemplate(tmpl)).Funcs(fm)
	t, err := t.Parse(string(tmpl))
	if err != nil {
		return nil, err
	}
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	return buff.Bytes(), err
}

func Execute(ctx context.Context, uri string, funcs ...template.FuncMap) ([]byte, error) {
	data := ContextGetTemplateData(ctx)
	fm := DefaultFuncMap(ctx)
	for _, opt := range funcs {
		fm = MergeFuncMaps(fm, opt)
	}

	url := uri
	if applied, err := Apply([]byte(uri), data, fm); err != nil {
		return nil, err
	} else {
		url = string(applied)
	}

	var body []byte
	var err error
	switch {
	case strings.Index(url, "func://") == 0:
		if f, has := fm[url[len("func://"):]]; !has {
			return nil, ErrMissingTemplateFunc
		} else {
			switch f.(type) {
			case func() []byte:
				body = f.(func() []byte)()
			case func() string:
				body = []byte(f.(func() string)())
			default:
				return nil, ErrBadTemplateFunc
			}
		}
	default:
		body, err = resource.Fetch(ctx, url)
		if err != nil {
			return nil, err
		}
	}
	return Apply(body, data, fm)
}
