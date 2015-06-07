package dash

import (
	"bytes"
	"encoding/json"
	"github.com/golang/glog"
	"github.com/qorio/maestro/pkg/zk"
	"strconv"
	"strings"
	"text/template"
)

// If value begins with env:// then automatically resolve the pointer recursively.
// Returns key, value, error
func Resolve(zc zk.ZK, key, value string) (string, string, error) {
	// de-reference the pointer...
	if strings.Index(value, "env://") == 0 {
		p := value[len("env://"):]
		n, err := zc.Get(p)
		switch {
		case err == zk.ErrNotExist:
			return key, "", nil
		case err != nil:
			return key, "", err
		}
		glog.Infoln("Resolving", key, "=", value, "==>", n.GetValueString())
		return Resolve(zc, key, n.GetValueString())
	} else {
		return key, value, nil
	}
}

func Zget(zc zk.ZK, key string) *string {
	n, err := zc.Get(key)
	switch {
	case err == zk.ErrNotExist:
		return nil
	case err != nil:
		return nil
	}
	v := n.GetValueString()
	if v == "" {
		return nil
	}
	return &v
}

func create_or_set(zc zk.ZK, key, value string) error {
	n, err := zc.Get(key)
	switch {
	case err == zk.ErrNotExist:
		n, err = zc.Create(key, []byte(value))
		if err != nil {
			return err
		}
	case err != nil:
		return err
	}
	err = n.Set([]byte(value))
	if err != nil {
		return err
	}
	return nil
}

func increment(zc zk.ZK, key string, increment int) error {
	n, err := zc.Get(key)
	switch {
	case err == zk.ErrNotExist:
		n, err = zc.Create(key, []byte(strconv.Itoa(0)))
		if err != nil {
			return err
		}
	case err != nil:
		return err
	}
	count, err := strconv.Atoi(n.GetValueString())
	if err != nil {
		count = 0
	}
	err = n.Set([]byte(strconv.Itoa(count + 1)))
	if err != nil {
		return err
	}
	return nil
}

func EscapeVars(escaped ...string) map[string]interface{} {
	m := map[string]interface{}{}
	for _, k := range escaped {
		m[k] = "{{." + k + "}}"
	}
	return m
}

func EscapeVar(k string) string {
	return "{{." + k + "}}"
}

func MergeMaps(m ...map[string]interface{}) map[string]interface{} {
	merged := map[string]interface{}{}
	for _, mm := range m {
		for k, v := range mm {
			merged[k] = v
		}
	}
	return merged
}

// Takes an original value/ struct that has its fields with {{.Template}} values and apply
// substitutions and returns a transformed value.  This allows multiple passes of applying templates.
func ApplyVarSubs(original, applied interface{}, context interface{}) (err error) {
	// first marshal into json
	json_buff, err := json.Marshal(original)
	if err != nil {
		return err
	}
	// now apply the entire json as if it were a template
	tpl, err := template.New(string(json_buff)).Parse(string(json_buff))
	if err != nil {
		return err
	}
	var buff bytes.Buffer
	err = tpl.Execute(&buff, context)
	if err != nil {
		return err
	}
	// now turn it back into a object
	return json.Unmarshal(buff.Bytes(), applied)
}
