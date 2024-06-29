package main

import (
	_ "embed"
	"text/template"
)

var (
	//go:embed index.txt
	eIndexApi []byte

	//go:embed bs.css
	eBScss []byte

	//go:embed index.html
	eIndex []byte

	//go:embed jobstatus.template.html
	eStatus string

	tStatus *template.Template = func() *template.Template {
		templ, err := template.New("status").Parse(eStatus)
		if err != nil {
			panic(err)
		}

		return templ
	}()
)
