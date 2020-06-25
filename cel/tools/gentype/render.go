package main

import (
	"bytes"
	"html/template"
)

// render renders the template using the definition.
func render(def definition) (string, error) {

	var buf bytes.Buffer
	tt := template.Must(template.New("type").Parse(src))
	err := tt.Execute(&buf, def)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
