package templates

import(
	"html/template"
)

var tpl *template.Template

func Setup() *template.Template {
	tpl = template.Must(template.ParseGlob("/web/internal/templates/*.html"))
	return template.Must(template.ParseGlob("/web/templates/*.html"))
}
