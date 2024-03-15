package helpers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/justinas/nosurf"
	"github.com/nickzer0/GoBoxer/internal/templates"
)

// AddDefaultData adds data for all the templates
func DefaultData(td templates.TemplateData, r *http.Request, w http.ResponseWriter) templates.TemplateData {
	td.Flash = app.Session.PopString(r.Context(), "flash")
	td.Error = app.Session.PopString(r.Context(), "error")
	td.Warning = app.Session.PopString(r.Context(), "warning")
	td.CSRFToken = nosurf.Token(r)
	td.PreferenceMap = app.PreferenceMap

	if app.Session.Get(r.Context(), "access_level") == 10 {
		td.IsAdmin = 1
	}

	if app.Session.Exists(r.Context(), "username") {
		td.IsAuthenticated = 1
		td.User.Username = app.Session.Get(r.Context(), "username").(string)
		td.User.ID = app.Session.Get(r.Context(), "user_id").(int)
	}

	return td
}

// Template renders templates using html/template
func RenderPage(w http.ResponseWriter, r *http.Request, templateName string, variables, data interface{}) error {
	var vars jet.VarMap

	if variables == nil {
		vars = make(jet.VarMap)
	} else {
		vars = variables.(jet.VarMap)
	}

	var td templates.TemplateData
	if data != nil {
		td = data.(templates.TemplateData)
	}

	td = DefaultData(td, r, w)
	addTemplateFunctions(r)

	t, err := views.GetTemplate(fmt.Sprintf("%s.jet", templateName))
	if err != nil {
		log.Println(err)
		return err
	}

	if err = t.Execute(w, vars, td); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

var views = jet.NewSet(
	jet.NewOSFileSystemLoader("./views"),
	jet.InDevelopmentMode(),
)
