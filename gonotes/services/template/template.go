package template

import "html/template"

//This package provides template helpers

//LoadTemplate loads the template tmplName from tplDir
func LoadTemplate(tplDir, tmplName string) *template.Template {
	t, err := template.New(tmplName).ParseFiles(tplDir + "/" + tmplName)
	if err != nil {
		panic(err)
	}
	return t
}
