package guidance

import (
	"bytes"
	"embed"
	"strconv"
	"text/template"
)

//go:embed files/*.md files/*.tmpl files/*.yaml
var fs embed.FS

func Read(name string) (string, error) {
	raw, err := fs.ReadFile("files/" + name)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func MustRead(name string) string {
	text, err := Read(name)
	if err != nil {
		panic(err)
	}
	return text
}

func RenderTemplate(name string, data any) (string, error) {
	raw, err := Read(name)
	if err != nil {
		return "", err
	}
	tpl, err := template.New(name).Funcs(template.FuncMap{
		"quote": strconv.Quote,
	}).Parse(raw)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := tpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}
