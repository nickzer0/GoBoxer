package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
	"github.com/nickzer0/GoBoxer/internal/redirectors"
)

// Redirectors displays a list of domain redirectors associated with the user's projects.
func (m *Repository) Redirectors(w http.ResponseWriter, r *http.Request) {
	redirectors, err := m.DB.GetAllDomainRedirectors()
	if err != nil {
		log.Printf("Error fetching all domain redirectors: %v", err)
		printTemplateError(w, err)
		return
	}

	user := m.App.Session.Get(r.Context(), "username").(string)
	projects, err := m.DB.GetProjectsForUser(user)
	if err != nil {
		log.Printf("Error fetching projects for user %s: %v", user, err)
		printTemplateError(w, err)
		return
	}

	var redirectorList []models.Redirector
	projectMap := make(map[int]bool) // Create a map for efficient project number lookup

	for _, project := range projects {
		projectMap[project.ProjectNumber] = true
	}

	for _, redirector := range redirectors {
		if _, exists := projectMap[redirector.Project]; exists {
			redirectorList = append(redirectorList, redirector)
		}
	}

	vars := make(jet.VarMap)
	vars.Set("redirectors", redirectorList)

	if err := helpers.RenderPage(w, r, "redirectors", vars, nil); err != nil {
		log.Printf("Error rendering redirectors page: %v", err)
		printTemplateError(w, err)
	}
}

// RedirectorAdd renders the page for adding a new redirector, listing available providers and user-associated projects.
func (m *Repository) RedirectorAdd(w http.ResponseWriter, r *http.Request) {
	secrets, err := m.DB.GetAllSecrets()
	if err != nil {
		log.Printf("Error fetching secrets: %v", err)
		printTemplateError(w, err)
		return
	}

	user := m.App.Session.Get(r.Context(), "username").(string)
	projects, err := m.DB.GetProjectsForUser(user)
	if err != nil {
		log.Printf("Error fetching projects for user %s: %v", user, err)
		printTemplateError(w, err)
		return
	}

	var providers []string
	if _, ok := secrets["awsaccount"]; ok {
		providers = append(providers, "Cloudfront")
	}

	vars := make(jet.VarMap)
	vars.Set("providers", providers)
	vars.Set("projects", projects)

	if err := helpers.RenderPage(w, r, "redirector-add", vars, nil); err != nil {
		log.Printf("Error rendering redirector add page: %v", err)
		printTemplateError(w, err)
	}
}

// RedirectorAddPost handles the POST request for creating a new domain redirector.
func (m *Repository) RedirectorAddPost(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))
	m.SendMessage(userID, "Creating domain redirector...")

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		m.SendMessage(userID, "Error parsing form data.")
		http.Redirect(w, r, "/app/redirectors/add", http.StatusSeeOther)
		return
	}

	provider := r.Form.Get("provider")
	projectID, err := strconv.Atoi(r.Form.Get("project"))
	if err != nil {
		log.Printf("Invalid project ID: %v", err)
		m.SendMessage(userID, "Invalid project ID provided.")
		http.Redirect(w, r, "/app/redirectors/add", http.StatusSeeOther)
		return
	}

	redirector := models.Redirector{
		Domain:  r.Form.Get("domain"),
		Project: projectID,
	}

	var newRedirector models.Redirector
	if provider == "Cloudfront" {
		newRedirector, err = redirectors.Repo.CreateCloudfrontDomain(redirector)
		if err != nil {
			log.Printf("Error creating Cloudfront domain: %v", err)
			m.SendMessage(userID, "Failed to create Cloudfront domain.")
			http.Redirect(w, r, "/app/redirectors/add", http.StatusSeeOther)
			return
		}
	} else {
		log.Printf("Unsupported provider: %s", provider)
		m.SendMessage(userID, "Unsupported provider specified.")
		http.Redirect(w, r, "/app/redirectors/add", http.StatusSeeOther)
		return
	}

	go m.WaitUntilReadyRoutine(newRedirector)
	m.SendMessage(userID, "Domain redirector creation initiated.")
	http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
}

// WaitUntilReadyRoutine polls the database for a redirector's status and broadcasts a message once it's ready.
func (m *Repository) WaitUntilReadyRoutine(redirector models.Redirector) {
	for {
		returnedRedirector, err := m.DB.GetDomainRedirector(redirector.ID)
		if err != nil {
			log.Printf("Error fetching domain redirector %d: %v", redirector.ID, err)
			return
		}

		if returnedRedirector.Status == "Ready" {
			break
		}

		time.Sleep(5 * time.Second) // Sleep before checking again
	}

	data := map[string]string{
		"status":        "Ready",
		"redirector_id": strconv.Itoa(redirector.ID),
	}

	m.Broadcast("public-channel", "redirector-changed", data)
}

// RedirectorDelete handles the request to delete a domain redirector.
func (m *Repository) RedirectorDelete(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		m.SendMessage(userID, "Failed to process request.")
		http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(r.Form.Get("id"))
	if err != nil {
		log.Printf("Invalid redirector ID: %v", err)
		m.SendMessage(userID, "Invalid redirector ID provided.")
		http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
		return
	}

	redirector, err := m.DB.GetDomainRedirector(id)
	if err != nil {
		log.Printf("Error fetching redirector: %v", err)
		m.SendMessage(userID, "Redirector not found.")
		http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
		return
	}

	if err := redirectors.Repo.DeleteCloudfrontDistribution(redirector); err != nil {
		log.Printf("Error deleting Cloudfront domain: %v", err)
		m.SendMessage(userID, "Failed to delete Cloudfront domain.")
		http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Redirector deleted successfully!")
	http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
}

// RedirectorsSync initiates the resynchronization process for a specified redirector.
func (m *Repository) RedirectorsSync(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))
	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		m.SendMessage(userID, "Failed to process request.")
		http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
		return
	}

	id, err := strconv.Atoi(r.Form.Get("id"))
	if err != nil {
		log.Printf("Invalid redirector ID: %v", err)
		m.SendMessage(userID, "Invalid redirector ID provided.")
		http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
		return
	}

	redirector, err := m.DB.GetDomainRedirector(id)
	if err != nil {
		log.Printf("Error fetching redirector: %v", err)
		m.SendMessage(userID, "Redirector not found.")
		http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
		return
	}

	go redirectors.Repo.ResyncDistribution(redirector)

	m.SendMessage(userID, "Resync initiated.")
	http.Redirect(w, r, "/app/redirectors", http.StatusSeeOther)
}
