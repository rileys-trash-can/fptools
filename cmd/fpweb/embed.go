package main

import (
	"embed"
	"html/template"
	"io"
)

var (
	//go:embed html
	embedFS embed.FS

	tStatus *template.Template = func() *template.Template {
		f, err := embedFS.Open("html/jobstatus.template.html")
		if err != nil {
			panic(err)
		}

		data, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}

		templ, err := template.New("status").Parse(string(data))
		if err != nil {
			panic(err)
		}

		return templ
	}()

	tList *template.Template = func() *template.Template {
		f, err := embedFS.Open("html/list.template.html")
		if err != nil {
			panic(err)
		}

		data, err := io.ReadAll(f)
		if err != nil {
			panic(err)
		}

		templ, err := template.New("status").Parse(string(data))
		if err != nil {
			panic(err)
		}

		return templ
	}()
)
