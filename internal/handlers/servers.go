package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/CloudyKit/jet/v6"
	"github.com/nickzer0/GoBoxer/internal/deploy"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
	"github.com/nickzer0/GoBoxer/internal/server"
)

// Servers displays the list of servers associated with the current user.
func (m *Repository) Servers(w http.ResponseWriter, r *http.Request) {
	currentUser := m.App.Session.Get(r.Context(), "user").(models.User)

	servers, err := m.DB.ListAllServersForUser(currentUser.Username)
	if err != nil {
		log.Printf("Error listing servers for user %s: %v", currentUser.Username, err)
		m.App.Session.Put(r.Context(), "error", "Failed to retrieve servers.")
		http.Redirect(w, r, "/", http.StatusSeeOther) 
		return
	}

	vars := make(jet.VarMap)
	vars.Set("servers", servers)

	if err := helpers.RenderPage(w, r, "servers", vars, nil); err != nil {
		log.Printf("Error rendering servers page: %v", err)
		printTemplateError(w, err)
	}
}

// ServersAdd displays the form for adding a new server, including lists of projects, scripts, and providers.
func (m *Repository) ServersAdd(w http.ResponseWriter, r *http.Request) {
	currentUser := m.App.Session.Get(r.Context(), "user").(models.User)

	projects, err := m.DB.GetProjectsForUser(currentUser.Username)
	if err != nil {
		log.Printf("Error getting projects for user %s: %v", currentUser.Username, err)
		printErrorPage(w, err)
		return
	}

	scripts, err := GetAnsibleScriptsNames()
	if err != nil {
		log.Printf("Error getting Ansible script names: %v", err)
		printErrorPage(w, err)
		return
	}

	secrets, err := m.DB.GetAllSecrets()
	if err != nil {
		log.Printf("Error getting all secrets: %v", err)
		printTemplateError(w, err)
		return
	}

	var providers []string
	for name := range secrets {
		if secrets[name] != "" {
			providers = append(providers, name)
		}
	}

	vars := make(jet.VarMap)
	vars.Set("projects", projects)
	vars.Set("scripts", scripts)
	vars.Set("providers", providers)

	if err := helpers.RenderPage(w, r, "servers-add", vars, nil); err != nil {
		log.Printf("Error rendering servers-add page: %v", err)
		printTemplateError(w, err)
	}
}

// ServersAddPost processes the form submission for adding a new server and initiates its deployment.
func (m *Repository) ServersAddPost(w http.ResponseWriter, r *http.Request) {
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))

	if m.App.Session.Get(r.Context(), "ssh_key") == "0" {
		m.App.Session.Put(r.Context(), "error", "No SSH Key set for user!")
		http.Redirect(w, r, "/app/servers/add", http.StatusSeeOther)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		m.SendError(userID, "Failed to process form data.")
		http.Redirect(w, r, "/app/servers/add", http.StatusSeeOther)
		return
	}

	provider := r.Form.Get("provider")
	project := r.Form.Get("assign_project")
	hostname := r.Form.Get("hostname")
	scripts := r.PostForm["scripts"]

	projectInt, err := strconv.Atoi(project)
	if err != nil {
		log.Printf("Error converting project ID: %v", err)
		m.SendError(userID, "Invalid project selection.")
		http.Redirect(w, r, "/app/servers/add", http.StatusSeeOther)
		return
	}

	newServer := models.Server{
		OS:        "ubuntu-2204",
		IP:        "Pending",
		Provider:  provider,
		Name:      hostname,
		Status:    "Deploying",
		Roles:     scripts,
		Project:   projectInt,
		Creator:   m.App.Session.Get(r.Context(), "username").(string),
		CreatedAt: time.Now(),
	}

	databaseServer, err := m.DB.AddServerToDatabase(newServer)
	if err != nil {
		log.Printf("Error adding server to database: %v", err)
		m.SendError(userID, "Failed to add server to database.")
		http.Redirect(w, r, "/app/servers/add", http.StatusSeeOther)
		return
	}

	// Use a goroutine to handle server creation asynchronously
	go m.CreateServerRoutine(databaseServer, userID)

	m.App.Session.Put(r.Context(), "flash", "Deploying server...")
	http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
}

func (m *Repository) CreateServerRoutine(newServer models.Server, userID string) {
	// Initialize deployment status and data
	status := "Configuring"
	data := map[string]string{
		"server_id": strconv.Itoa(newServer.ID),
		"project":   strconv.Itoa(newServer.Project),
		"provider":  newServer.Provider,
		"hostname":  newServer.Name,
		"os":        newServer.OS,
		"status":    status,
	}

	// Deploy server based on its provider
	var deployedServer models.Server
	var err error
	switch newServer.Provider {
	case "digitalocean":
		deployedServer, err = server.Repo.DigitalOceanCreateServer(newServer)
	case "linode":
		deployedServer, err = server.Repo.LinodeCreateServer(newServer)
	default:
		log.Printf("Unknown provider: %s", newServer.Provider)
		m.SendMessage(userID, "Unknown provider specified for deployment.")
		return
	}

	// Update data with IP address of the deployed server
	data["ip_address"] = deployedServer.IP

	// Handle errors from server deployment
	if err != nil {
		_ = m.DB.DeleteServerFromDatabase(newServer.ID)
		log.Printf("Error deploying server: %v", err)
		m.SendMessage(userID, "Error deploying server.")
		return
	}

	// Update server details in the database
	if err = m.DB.UpdateServer(deployedServer); err != nil {
		log.Printf("Error updating server in database: %v", err)
	}

	m.SendMessage(userID, fmt.Sprintf("Server %s deployed, configuring...", deployedServer.Name))
	m.Broadcast("public-channel", "server-changed", data)

	// Convert userID to int for database query
	userIDint, err := strconv.Atoi(userID)
	if err != nil {
		log.Printf("Error converting userID to int: %v", err)
		return
	}

	// Retrieve user details from the database
	user, err := m.DB.GetUserFromID(userIDint)
	if err != nil {
		log.Printf("Error retrieving user from database: %v", err)
		return
	}

	// Add SSH key for the user to the deployed server
	if err = deploy.AddSSHUser(deployedServer, user); err != nil {
		log.Printf("Error enabling access on server: %v", err)
	}

	// If server has roles, initiate provisioning routine
	if len(newServer.Roles) > 0 {
		go m.ProvisionServerRoutine(deployedServer, userID)
		return
	}

	// Finalize server status to 'Ready' if no roles are specified
	status = "Ready"
	data["status"] = status
	deployedServer.Status = status

	if err = m.DB.UpdateServer(deployedServer); err != nil {
		log.Printf("Error updating server status to ready: %v", err)
	}

	m.SendMessage(userID, fmt.Sprintf("Server %s ready", deployedServer.Name))
	m.Broadcast("public-channel", "server-changed", data)
}

// ProvisionServerRoutine manages the provisioning process for a newly deployed server.
func (m *Repository) ProvisionServerRoutine(newServer models.Server, userID string) {
	// Initial provisioning status update
	data := map[string]string{
		"server_id":  strconv.Itoa(newServer.ID),
		"project":    strconv.Itoa(newServer.Project),
		"provider":   newServer.Provider,
		"hostname":   newServer.Name,
		"os":         newServer.OS,
		"ip_address": newServer.IP,
		"status":     "Provisioning",
	}
	time.Sleep(1 * time.Second)
	m.Broadcast("public-channel", "server-changed", data)

	// Update server status in the database
	newServer.Status = "Provisioning"
	if err := m.DB.UpdateServer(newServer); err != nil {
		log.Printf("Error updating server status to provisioning: %v", err)
		m.SendMessage(userID, "Error updating server in database!")
	}

	time.Sleep(5 * time.Second)
	m.SendMessage(userID, fmt.Sprintf("Server %s is being provisioned", newServer.Name))

	// Execute provisioning playbook
	if err := deploy.RunPlayBook(newServer); err != nil {
		log.Printf("Error during server provisioning: %v", err)
		data["status"] = "ERROR"
		newServer.Status = "ERROR"
		m.Broadcast("public-channel", "server-changed", data)
		m.SendMessage(userID, fmt.Sprintf("Error provisioning server %s: %v", newServer.Name, err))

		// Update server status to ERROR in database
		if dbErr := m.DB.UpdateServer(newServer); dbErr != nil {
			log.Printf("Error updating server status to ERROR: %v", dbErr)
		}
		return
	}

	data["status"] = "Ready"
	newServer.Status = "Ready"
	if err := m.DB.UpdateServer(newServer); err != nil {
		log.Printf("Error updating server status to ready: %v", err)
		m.SendMessage(userID, "Error updating server in database!")

	}

	m.Broadcast("public-channel", "server-changed", data)
	m.SendMessage(userID, fmt.Sprintf("Server %s ready", newServer.Name))
}

// ProvisionServer initiates the provisioning of a server identified by its ID in the request URL.
// This function is for manual provisioning of a server after it has been deployed.
func (m *Repository) ProvisionServer(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 5 {
		log.Printf("Invalid server ID in URL")
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	serverID, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting server ID to integer: %v", err)
		m.App.Session.Put(r.Context(), "error", "Invalid server ID provided.")
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))

	server, err := m.DB.GetServer(serverID)
	if err != nil {
		log.Printf("Error retrieving server by ID: %v", err)
		m.App.Session.Put(r.Context(), "error", "Server not found.")
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	// Use a goroutine for asynchronous provisioning.
	go m.ProvisionServerRoutine(server, userID)

	m.App.Session.Put(r.Context(), "flash", "Server provisioning initiated.")
	http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
}

// ViewServer displays details for a specific server including its associated scripts.
func (m *Repository) ViewServer(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 4 {
		log.Println("Invalid server ID in URL")
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	serverID, err := strconv.Atoi(exploded[3])
	if err != nil {
		log.Printf("Error converting server ID: %v", err)
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	server, err := m.DB.GetServer(serverID)
	if err != nil {
		log.Printf("Error fetching server by ID: %v", err)
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	scripts, err := GetAnsibleScriptsNames()
	if err != nil {
		log.Printf("Error getting Ansible script names: %v", err)
		printErrorPage(w, err)
		return
	}

	vars := make(jet.VarMap)
	vars.Set("server", server)
	vars.Set("scripts", scripts)

	if err := helpers.RenderPage(w, r, "servers-view", vars, nil); err != nil {
		log.Printf("Error rendering server view page: %v", err)
		printTemplateError(w, err)
	}
}

// UpdateServer handles the update of server roles and initiates provisioning based on new roles.
func (m *Repository) UpdateServer(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 5 {
		log.Println("Invalid server ID in URL")
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	serverID, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting server ID to integer: %v", err)
		printErrorPage(w, err)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Error parsing form: %v", err)
		printErrorPage(w, err)
		return
	}

	scripts := r.PostForm["scripts"]
	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))

	server, err := m.DB.GetServer(serverID)
	if err != nil {
		log.Printf("Error fetching server by ID: %v", err)
		printErrorPage(w, err)
		return
	}

	// Check if role already exists for server
	for _, script := range scripts {
		found := false

		for _, existingScript := range server.Roles {
			if script == existingScript {
				found = true
				break
			}
		}

		if !found {
			server.Roles = append(server.Roles, script)
		}
	}

	// Update server in the database before provisioning
	if err := m.DB.UpdateServer(server); err != nil {
		log.Printf("Error updating server roles in database: %v", err)
		printErrorPage(w, err)
		return
	}

	// Use a goroutine to provision the server asynchronously
	go m.ProvisionServerRoutine(server, userID)

	m.App.Session.Put(r.Context(), "flash", "Server roles updated. Provisioning in progress...")
	http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
}

// ServersRemove initiates the removal of a specified server and redirects the user back to the server list page.
func (m *Repository) ServersRemove(w http.ResponseWriter, r *http.Request) {
	exploded := strings.Split(r.RequestURI, "/")
	if len(exploded) < 5 {
		log.Printf("Invalid path: missing server ID")
		m.App.Session.Put(r.Context(), "error", "Invalid request.")
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	serverID, err := strconv.Atoi(exploded[4])
	if err != nil {
		log.Printf("Error converting server ID to integer: %v", err)
		m.App.Session.Put(r.Context(), "error", "Invalid server ID provided.")
		http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
		return
	}

	userID := strconv.Itoa(m.App.Session.Get(r.Context(), "user_id").(int))

	// Use a goroutine to handle the server removal process asynchronously.
	go m.RemoveServersRoutine(serverID, userID)
	// Give the go routine time to delete the server before redirecting the user.
	// TODO: Create a waitgroup/channel for this incase there is a delay in deleting
	time.Sleep(2 * time.Second)
	http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
}

// RemoveServersRoutine handles the background removal of a server from both the database and the cloud provider.
func (m *Repository) RemoveServersRoutine(serverID int, userID string) {
	serverToRemove, err := m.DB.GetServer(serverID)
	if err != nil {
		log.Printf("Error fetching server %d: %v", serverID, err)
		m.SendMessage(userID, "Error fetching server details.")
		return
	}

	// Attempt to delete server from the cloud provider
	switch serverToRemove.Provider {
	case "digitalocean":
		if err := server.Repo.DigitalOceanDeleteServer(serverToRemove.ProviderID); err != nil {
			log.Printf("Error deleting DigitalOcean server %d: %v", serverToRemove.ProviderID, err)
			m.SendMessage(userID, fmt.Sprintf("Error removing server from DigitalOcean: %s", serverToRemove.Name))
			return
		}
	case "linode":
		if err := server.Repo.LinodeDeleteServer(serverToRemove.ProviderID); err != nil {
			log.Printf("Error deleting Linode server %d: %v", serverToRemove.ProviderID, err)
			m.SendMessage(userID, fmt.Sprintf("Error removing server from Linode: %s", serverToRemove.Name))
			return
		}
	default:
		log.Printf("Unknown provider for server %d: %s", serverID, serverToRemove.Provider)
		m.SendMessage(userID, fmt.Sprintf("Unknown provider for server: %s", serverToRemove.Name))
		return
	}

	if err := m.DB.DeleteServerFromDatabase(serverID); err != nil {
		log.Printf("Error deleting server %d from database: %v", serverID, err)
		m.SendMessage(userID, fmt.Sprintf("Error removing server from database: %s", serverToRemove.Name))
		return
	}

	m.SendMessage(userID, fmt.Sprintf("Server removed: %s", serverToRemove.Name))
}

// DeleteAllServers initiates an asynchronous operation to remove all servers.
func (m *Repository) DeleteAllServers(w http.ResponseWriter, r *http.Request) {
	// WARNING: Use with caution. This function should be restricted to debug environments.

	if app.InProduction {
		log.Println("Attempt to delete all servers in a non-debug environment")
		http.Redirect(w, r, "/app/servers", http.StatusForbidden)
		return
	}

	go m.DeleteAllServersRoutine()

	m.App.Session.Put(r.Context(), "flash", "All servers deleted!")
	http.Redirect(w, r, "/app/servers", http.StatusSeeOther)
}

func (m *Repository) DeleteAllServersRoutine() {
	servers, err := m.DB.ListAllServers()
	if err != nil {
		log.Printf("error fetching list of servers from database: %v", err)
		return
	}

	for _, server := range servers {
		if err := m.DB.DeleteServerFromDatabase(server.ID); err != nil {
			log.Printf("error deleting server (ID: %d) from database: %v", server.ID, err)
		}
	}

	// Asynchronously delete all servers from DigitalOcean
	go func() {
		if err := server.Repo.DigitalOceanDeleteAll(); err != nil {
			log.Printf("error deleting all servers from DigitalOcean: %v", err)
		}
	}()

	// Asynchronously delete all servers from Linode
	go func() {
		if err := server.Repo.LinodeDeleteAll(); err != nil {
			log.Printf("error deleting all servers from Linode: %v", err)
		}
	}()
}
