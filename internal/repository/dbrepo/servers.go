package dbrepo

import (
	"fmt"

	"github.com/nickzer0/GoBoxer/internal/models"
)

// AddServerToDatabase adds a new server entry to the database and assigns scripts to it.
func (m *sqliteDBRepo) AddServerToDatabase(server models.Server) (models.Server, error) {
	tx, err := m.DB.Begin()
	if err != nil {
		return server, err
	}

	query := `INSERT INTO servers (provider, server_name, provider_id, server_status, server_ip, server_os, server_project, created_by) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`
	var serverID int
	err = tx.QueryRow(query, server.Provider, server.Name, server.ProviderID, server.Status, server.IP, server.OS, server.Project, server.Creator).Scan(&serverID)
	if err != nil {
		tx.Rollback()
		return server, err
	}
	server.ID = serverID

	for _, scriptName := range server.Roles {
		var scriptID int
		scriptQuery := `SELECT id FROM scripts WHERE name = ?`
		err := tx.QueryRow(scriptQuery, scriptName).Scan(&scriptID)
		if err != nil {
			tx.Rollback()
			return server, fmt.Errorf("script %s not found in script table, try re-adding script", scriptName)
		}

		_, err = tx.Exec("INSERT OR IGNORE INTO scripts_servers (script_id, server_id) VALUES (?, ?)", scriptID, serverID)
		if err != nil {
			tx.Rollback()
			return server, err
		}

	}

	if err := tx.Commit(); err != nil {
		return server, err
	}

	return server, nil
}

// GetServer retrieves a server by its ID from the database, including the scripts assigned to it.
func (m *sqliteDBRepo) GetServer(id int) (models.Server, error) {
	var server models.Server
	query := `SELECT provider, server_name, provider_id, server_status, server_ip, server_os, server_project, created_by, created_at FROM servers WHERE id = ?`
	err := m.DB.QueryRow(query, id).Scan(&server.Provider, &server.Name, &server.ProviderID, &server.Status, &server.IP, &server.OS, &server.Project, &server.Creator, &server.CreatedAt)
	if err != nil {
		return server, err
	}
	server.ID = id

	scripts, err := m.GetScriptsForServer(id)
	if err != nil {
		return server, err
	}
	server.Roles = scripts
	return server, nil
}

// ListAllServers retrieves all servers stored in the database.
func (m *sqliteDBRepo) ListAllServers() ([]models.Server, error) {
	query := "SELECT id, provider, server_name, provider_id, server_status, server_ip, server_os, server_project FROM servers"
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var server models.Server
		if err := rows.Scan(&server.ID, &server.Provider, &server.Name, &server.ProviderID, &server.Status, &server.IP, &server.OS, &server.Project); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, rows.Err()
}

// ListAllServersForUser fetches all servers associated with the user's projects.
func (m *sqliteDBRepo) ListAllServersForUser(user string) ([]models.Server, error) {
	var servers []models.Server
	var server models.Server

	projects, err := m.GetProjectsForUser(user)
	if err != nil {
		return servers, err
	}

	query := `
		SELECT
			id, provider, server_name, provider_id, server_status, server_ip, server_os, server_project
		FROM
			servers
		WHERE
			server_project = ?
		`
	for _, project := range projects {
		rows, err := m.DB.Query(query, project.ProjectNumber)
		if err != nil {
			return servers, err
		}
		defer rows.Close()

		for rows.Next() {

			err = rows.Scan(
				&server.ID,
				&server.Provider,
				&server.Name,
				&server.ProviderID,
				&server.Status,
				&server.IP,
				&server.OS,
				&server.Project,
			)
			if err != nil {
				return servers, err
			}
			servers = append(servers, server)
		}
		err = rows.Err()
		if err != nil {
			return servers, err
		}
	}

	return servers, nil
}

// getProjectIDsForUser fetches all project IDs associated with a given username.
func (m *sqliteDBRepo) getProjectIDsForUser(username string) ([]int, error) {
	var projectIDs []int
	projects, err := m.GetProjectsForUser(username)
	if err != nil {
		return nil, err
	}
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
	}
	return projectIDs, nil
}

// ListAllServersForProject fetches all servers associated with a specific project number.
func (m *sqliteDBRepo) ListAllServersForProject(projectNumber string) ([]models.Server, error) {
	query := `
    SELECT
        id, provider, server_name, provider_id, server_status, server_ip, server_os, server_project
    FROM
        servers
    WHERE
        server_project = ?
    `
	rows, err := m.DB.Query(query, projectNumber)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []models.Server
	for rows.Next() {
		var server models.Server
		if err := rows.Scan(&server.ID, &server.Provider, &server.Name, &server.ProviderID, &server.Status, &server.IP, &server.OS, &server.Project); err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, rows.Err()
}

// UpdateServer updates server details in the database.
func (m *sqliteDBRepo) UpdateServer(server models.Server) error {
	_, err := m.DB.Exec(`
        UPDATE servers
        SET server_ip = ?, server_status = ?, provider_id = ?
        WHERE id = ?`,
		server.IP, server.Status, server.ProviderID, server.ID,
	)
	if err != nil {
		return err
	}

	for _, scriptName := range server.Roles {
		scriptID, err := m.getScriptIDByName(scriptName)
		if err != nil {
			return err
		}
		if err := m.AssignScriptToServer(scriptID, server.ID); err != nil {
			return err
		}
	}
	return nil
}

// getScriptIDByName returns the ID of a script given its name.
func (m *sqliteDBRepo) getScriptIDByName(scriptName string) (int, error) {
	var scriptID int
	query := "SELECT id FROM scripts WHERE name = ?"
	err := m.DB.QueryRow(query, scriptName).Scan(&scriptID)
	if err != nil {
		return 0, err
	}
	return scriptID, nil
}

// DeleteServerFromDatabase removes a server entry from the database.
func (m *sqliteDBRepo) DeleteServerFromDatabase(serverID int) error {
	_, err := m.DB.Exec("DELETE FROM servers WHERE id = ?", serverID)
	return err
}

// GetServiceDetails is used to return the number of services for each provider
// This data is used on the homepage/dashboard to populate the badge fields
func (m *sqliteDBRepo) GetServiceDetails() (models.Services, error) {
	var services models.Services
	query := `
	select 
		(select count(id) from servers where provider = 'digitalocean') as digitalocean,
		(select count(id) from servers where provider = 'linode') as linode,
		(select count(id) from redirectors where provider = 'AWS') as aws
	`

	row := m.DB.QueryRow(query)
	err := row.Scan(
		&services.DigitalOcean,
		&services.Linode,
		&services.AWS,
	)
	if err != nil {
		return services, err
	}

	return services, nil
}
