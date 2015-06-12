package dash

import (
	"bytes"
	"encoding/json"
	"text/template"
)

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
