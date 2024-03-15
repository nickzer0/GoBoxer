package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
)

// Projects renders the projects page, listing projects associated with the current user.
func (m *Repository) Projects(w http.ResponseWriter, r *http.Request) {
	currentUser := m.App.Session.Get(r.Context(), "username").(string)

	projects, err := m.DB.GetProjectsForUser(currentUser)
	if err != nil {
		log.Printf("Error getting projects for user %s: %v", currentUser, err)
		m.App.Session.Put(r.Context(), "error", "Error getting projects for current user!")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("projects", projects)

	if err := helpers.RenderPage(w, r, "projects", vars, nil); err != nil {
		log.Printf("Error rendering projects page: %v", err)
		printTemplateError(w, err)
	}
}

// ViewProject displays details for a specific project, including associated servers and redirectors.
func (m *Repository) ViewProject(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 4 {
		m.App.Session.Put(r.Context(), "error", "Invalid project ID")
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(exploded[3])
	if err != nil {
		log.Printf("Error converting project ID: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error getting project!")
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	project, err := m.DB.GetProjectByNumber(id)
	if err != nil {
		log.Printf("Error fetching project by number: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error getting project!")
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	userList, err := m.DB.GetUsers()
	if err != nil {
		log.Printf("Error fetching user list: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error getting list of users!")
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	var users []string
	for _, person := range userList {
		users = append(users, person.Username)
	}

	servers, err := m.DB.ListAllServersForProject(exploded[3])
	if err != nil {
		log.Printf("Error listing all servers for project: %v", err)
		printTemplateError(w, err)
		return
	}

	redirectors, err := m.DB.GetAllDomainRedirectors()
	if err != nil {
		log.Printf("Error fetching all domain redirectors: %v", err)
		printTemplateError(w, err)
		return
	}

	var redirectorsForProjects []models.Redirector
	for _, redirector := range redirectors {
		if project.ProjectNumber == redirector.Project {
			redirectorsForProjects = append(redirectorsForProjects, redirector)
		}
	}

	vars := make(jet.VarMap)
	vars.Set("project", project)
	vars.Set("users", users)
	vars.Set("servers", servers)
	vars.Set("redirectors", redirectorsForProjects)

	if err := helpers.RenderPage(w, r, "projects-view", vars, nil); err != nil {
		log.Printf("Error rendering project view page: %v", err)
		printTemplateError(w, err)
	}
}

// AddProjects renders the page for adding new projects, including user and project lists.
func (m *Repository) AddProjects(w http.ResponseWriter, r *http.Request) {
	userList, err := m.DB.GetUsers()
	if err != nil {
		log.Printf("Error getting list of users: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error getting list of users!")
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	currentUser := m.App.Session.Get(r.Context(), "username").(string)
	projects, err := m.DB.GetProjectNamesForUsername(currentUser)
	if err != nil {
		log.Printf("Error getting projects for user %s: %v", currentUser, err)
		m.App.Session.Put(r.Context(), "error", "Error getting projects for current user!")
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	var users []string
	for _, user := range userList {
		users = append(users, user.Username)
	}

	vars := make(jet.VarMap)
	vars.Set("users", users)
	vars.Set("projects", projects)

	if err := helpers.RenderPage(w, r, "projects-add", vars, nil); err != nil {
		log.Printf("Error rendering projects-add page: %v", err)
		printTemplateError(w, err)
	}
}

// AddProjectsPost handles the submission of a new project form and adds the project to the database.
func (m *Repository) AddProjectsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printErrorPage(w, err)
		return
	}

	projectID, err := strconv.Atoi(r.Form.Get("project_id"))
	if err != nil {
		m.App.Session.Put(r.Context(), "error", "Invalid project ID format.")
		http.Redirect(w, r, "/app/projects/add", http.StatusSeeOther)
		return
	}

	assignedUsers := r.PostForm["assign_users"]

	project := models.Project{
		ProjectNumber: projectID,
		ProjectName:   r.Form.Get("project_name"),
		CreatedBy:     m.App.Session.Get(r.Context(), "username").(string),
		Notes:         r.Form.Get("project_notes"),
	}

	exists, err := m.DB.CheckProjectByNumber(projectID)
	if err != nil {
		log.Printf("Error checking project by number: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error looking up project ID!")
		http.Redirect(w, r, "/app/projects/add", http.StatusSeeOther)
		return
	}

	if exists {
		m.App.Session.Put(r.Context(), "error", "Project ID already exists!")
		http.Redirect(w, r, "/app/projects/add", http.StatusSeeOther)
		return
	}

	if err := m.DB.AddProject(project, assignedUsers); err != nil {
		log.Printf("Error adding project: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error adding project!")
		http.Redirect(w, r, "/app/projects/add", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Project successfully added!")
	http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
}

// UpdateProjectsPost handles the submission of the project update form.
func (m *Repository) UpdateProjectsPost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printErrorPage(w, err)
		return
	}

	projectID, err := strconv.Atoi(r.Form.Get("project_id"))
	if err != nil {
		log.Printf("Invalid project ID: %v", err)
		m.App.Session.Put(r.Context(), "error", "Invalid project ID.")
		http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
		return
	}

	projectNumber, err := strconv.Atoi(r.Form.Get("project_number"))
	if err != nil {
		log.Printf("Invalid project number: %v", err)
		m.App.Session.Put(r.Context(), "error", "Invalid project number.")
		http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
		return
	}

	assignedUsers := r.PostForm["assign_users"]
	project := models.Project{
		ID:            projectID,
		ProjectNumber: projectNumber,
		ProjectName:   r.Form.Get("project_name"),
		CreatedBy:     m.App.Session.Get(r.Context(), "username").(string),
		Notes:         r.Form.Get("project_notes"),
	}

	if err := m.DB.UpdateProject(project, assignedUsers); err != nil {
		log.Printf("Error updating project: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error updating project.")
		http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Project updated successfully.")
	http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
}

// RemoveProject handles the request to delete a project from the database.
func (m *Repository) RemoveProject(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 5 {
		log.Printf("Invalid project ID in URL")
		m.App.Session.Put(r.Context(), "error", "Invalid request.")
		http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting project ID: %v", err)
		m.App.Session.Put(r.Context(), "error", "Invalid project ID.")
		http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
		return
	}

	if err := m.DB.DeleteProjectByNumber(id); err != nil {
		log.Printf("Error removing project: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error removing project.")
		http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Project successfully removed.")
	http.Redirect(w, r, "/app/projects", http.StatusSeeOther)
}
