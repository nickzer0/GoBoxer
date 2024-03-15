package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
)

// Home page renderer for home page
func (m *Repository) Home(w http.ResponseWriter, r *http.Request) {
	loggedIn := m.App.Session.Exists(r.Context(), "username")
	if !loggedIn {
		helpers.RenderPage(w, r, "login", nil, nil)
		return
	}

	vars := make(jet.VarMap)
	services, err := m.DB.GetServiceDetails()
	if err != nil {
		log.Printf("Error fetching service details: %v", err)
		printErrorPage(w, err)
		return
	}
	vars.Set("services", services)

	username := m.App.Session.Get(r.Context(), "username").(string)
	var projects []models.Project
	if helpers.IsAdmin(r) {
		projects, err = m.DB.GetAllProjects()
	} else {
		projects, err = m.DB.GetProjectsForUser(username)
	}
	if err != nil {
		log.Printf("Error fetching projects: %v", err)
		printErrorPage(w, err)
		return
	}

	vars.Set("projects", projects)
	if err := helpers.RenderPage(w, r, "home", vars, nil); err != nil {
		log.Printf("Error rendering home page: %v", err)
		printTemplateError(w, err)
	}
}

// printTemplateError reports a template execution error to the user.
func printTemplateError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError) // Set HTTP status code to 500
	fmt.Fprintf(w, `<html><small><span class='text-danger'><h1>Error executing template: %s</h1></span></small></html>`, err)
}

// printErrorPage displays a generic error message to the user.
func printErrorPage(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusInternalServerError) // Set HTTP status code to 500
	fmt.Fprintf(w, `<html><small><span class='text-danger'><h1>Error: %s</h1></span></small></html>`, err)
}
