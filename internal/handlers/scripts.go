package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
)

// Scripts displays a list of available Ansible scripts.
func (m *Repository) Scripts(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))
	scripts, err := m.DB.ListAllScripts()
	if err != nil {
		m.SendMessage(userID, "Error getting Ansible scripts!")
		http.Redirect(w, r, "/app/dashboard", http.StatusSeeOther)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("scripts", scripts)

	if err := helpers.RenderPage(w, r, "scripts", vars, nil); err != nil {
		log.Printf("Error rendering scripts page: %v", err)
		printTemplateError(w, err)
	}
}

// ScriptView displays details and contents of a specific Ansible script by ID.
func (m *Repository) ScriptView(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 4 {
		log.Println("Invalid script ID in URL")
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	scriptID, err := strconv.Atoi(exploded[3])
	if err != nil {
		log.Printf("Error converting script ID: %v", err)
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	script, err := m.DB.GetScriptByID(scriptID)
	if err != nil {
		log.Printf("Error fetching script by ID: %v", err)
		printTemplateError(w, err)
		return
	}

	scriptFile := fmt.Sprintf("./scripts/added/%s.yml", script.Name)
	scriptContent, err := os.ReadFile(scriptFile)
	if err != nil {
		log.Printf("Error reading script file: %v", err)
		printTemplateError(w, err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("script", script)
	vars.Set("scriptContent", string(scriptContent))

	if err := helpers.RenderPage(w, r, "scripts-view", vars, nil); err != nil {
		log.Printf("Error rendering script view page: %v", err)
		printTemplateError(w, err)
	}
}

// ScriptsAdd displays the form for adding new Ansible scripts.
func (m *Repository) ScriptsAdd(w http.ResponseWriter, r *http.Request) {
	if err := helpers.RenderPage(w, r, "scripts-add", make(jet.VarMap), nil); err != nil {
		printTemplateError(w, err)
	}
}

// UploadScript handles the upload of new Ansible scripts and saves them to the server.
func (m *Repository) UploadScript(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))
	userName := m.App.Session.Get(r.Context(), "username").(string)

	if err := r.ParseForm(); err != nil {
		m.SendError(userID, "Error parsing form data.")
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		m.SendError(userID, "Error retrieving the file.")
		return
	}
	defer file.Close()

	fileName := handler.Filename
	scriptPath := fmt.Sprintf("./scripts/added/%s", fileName)

	// Prevent overwriting existing files and ensure correct file extension
	if filepath.Ext(fileName) != ".yml" {
		m.SendError(userID, "File needs to be .yml extension.")
		return
	}

	if _, err := os.Stat(scriptPath); err == nil {
		m.SendError(userID, "File already exists.")
		return
	}

	// Save the file
	dst, err := os.Create(scriptPath)
	if err != nil {
		m.SendError(userID, "Error saving the file.")
		return
	}
	defer dst.Close()

	if _, err = io.Copy(dst, file); err != nil {
		m.SendError(userID, "Error writing file to disk.")
		return
	}

	// Add script details to the database
	script := models.Script{
		Name:        strings.TrimSuffix(fileName, ".yml"),
		CreatedBy:   userName,
		Description: r.Form.Get("description"),
	}

	if _, err = m.DB.AddScript(script); err != nil {
		m.SendError(userID, "Error adding script to database.")
		return
	}

	m.SendMessage(userID, "File uploaded successfully.")
	http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
}

// UpdateScript handles the update of an existing Ansible script's metadata and content.
func (m *Repository) UpdateScript(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 5 {
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	scriptID, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting script ID: %v", err)
		printTemplateError(w, err)
		return
	}

	script, err := m.DB.GetScriptByID(scriptID)
	if err != nil {
		log.Printf("Error fetching script by ID: %v", err)
		printTemplateError(w, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printTemplateError(w, err)
		return
	}

	script.Description = r.Form.Get("description")
	if err := m.DB.UpdateScript(script); err != nil {
		log.Printf("Error updating script: %v", err)
		printTemplateError(w, err)
		return
	}

	scriptContent := r.Form.Get("script_content")
	scriptFile := fmt.Sprintf("./scripts/added/%s.yml", script.Name)
	if err := os.WriteFile(scriptFile, []byte(scriptContent), 0644); err != nil {
		log.Printf("Error writing script file: %v", err)
		printTemplateError(w, err)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Script updated successfully!")
	http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
}

// RemoveScript handles the deletion of a script both from the database and the file system.
func (m *Repository) RemoveScript(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 5 {
		m.App.Session.Put(r.Context(), "error", "Invalid script ID.")
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	scriptID, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting script ID to integer: %v", err)
		m.App.Session.Put(r.Context(), "error", "Invalid script ID format.")
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	script, err := m.DB.GetScriptByID(scriptID)
	if err != nil {
		log.Printf("Error retrieving script by ID: %v", err)
		m.App.Session.Put(r.Context(), "error", "Script not found.")
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	if err := m.DB.RemoveScript(script.Name); err != nil {
		log.Printf("Error removing script from database: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error removing script from database.")
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	fileToRemove := fmt.Sprintf("./scripts/added/%s.yml", script.Name)
	if err := os.Remove(fileToRemove); err != nil {
		log.Printf("Error removing script file: %v", err)
		m.App.Session.Put(r.Context(), "error", "Error removing script file.")
		http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
		return
	}

	m.App.Session.Put(r.Context(), "flash", "Script removed successfully.")
	http.Redirect(w, r, "/app/scripts", http.StatusSeeOther)
}

// GetAnsibleScriptsNames retrieves the names of all Ansible playbooks in the scripts directory, excluding the ".yml" extension.
func GetAnsibleScriptsNames() ([]string, error) {
	var ansibleScripts []string

	ansibleFiles, err := os.ReadDir("./scripts/added/")
	if err != nil {
		return nil, err
	}

	for _, file := range ansibleFiles {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".yml") {
			scriptName := strings.TrimSuffix(file.Name(), ".yml")
			ansibleScripts = append(ansibleScripts, scriptName)
		}
	}

	return ansibleScripts, nil
}
