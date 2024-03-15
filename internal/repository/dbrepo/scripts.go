package dbrepo

import (
	"time"

	"github.com/nickzer0/GoBoxer/internal/models"
)

// AddScript inserts a new script into the database and returns it with its ID updated.
func (m *sqliteDBRepo) AddScript(script models.Script) (models.Script, error) {
	query := "INSERT INTO scripts (name, icon, description, created_by, created_at) VALUES (?, ?, ?, ?, ?)"
	result, err := m.DB.Exec(query, script.Name, script.Icon, script.Description, script.CreatedBy, time.Now())
	if err != nil {
		return script, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return script, err
	}
	script.ID = int(id)
	return script, nil
}

// RemoveScript removes a script from the database based on its name.
func (m *sqliteDBRepo) RemoveScript(script string) error {
	_, err := m.DB.Exec("DELETE FROM scripts WHERE name = ?", script)
	return err
}

// AssignScriptToServer links a script to a server by their IDs in the database.
func (m *sqliteDBRepo) AssignScriptToServer(scriptID, serverID int) error {
	var exists int
	err := m.DB.QueryRow("SELECT COUNT(*) FROM scripts_servers WHERE script_id = ? AND server_id = ?", scriptID, serverID).Scan(&exists)
	if err != nil {
		return err
	}

	if exists == 0 {
		_, err = m.DB.Exec("INSERT INTO scripts_servers (script_id, server_id) VALUES (?, ?)", scriptID, serverID)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetScriptsForServer retrieves script names executed on a specific server.
func (m *sqliteDBRepo) GetScriptsForServer(serverID int) ([]string, error) {
	var scripts []string
	query := "SELECT name FROM scripts INNER JOIN scripts_servers ON scripts_servers.script_id = scripts.id WHERE scripts_servers.server_id = ?"
	rows, err := m.DB.Query(query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var name string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		scripts = append(scripts, name)
	}
	return scripts, rows.Err()
}

// ListAllScripts fetches all scripts from the database.
func (m *sqliteDBRepo) ListAllScripts() ([]models.Script, error) {
	var scripts []models.Script
	query := "SELECT id, name, icon, description, created_by, created_at FROM scripts"
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var script models.Script
		if err := rows.Scan(&script.ID, &script.Name, &script.Icon, &script.Description, &script.CreatedBy, &script.CreatedAt); err != nil {
			return nil, err
		}
		scripts = append(scripts, script)
	}
	return scripts, rows.Err()
}

// GetScriptByID fetches a single script by its ID.
func (m *sqliteDBRepo) GetScriptByID(id int) (models.Script, error) {
	var script models.Script
	query := "SELECT id, name, icon, description, created_by, created_at FROM scripts WHERE id = ?"
	err := m.DB.QueryRow(query, id).Scan(&script.ID, &script.Name, &script.Icon, &script.Description, &script.CreatedBy, &script.CreatedAt)
	return script, err
}

// UpdateScript updates a script's details in the database.
func (m *sqliteDBRepo) UpdateScript(script models.Script) error {
	_, err := m.DB.Exec("UPDATE scripts SET description = ? WHERE id = ?", script.Description, script.ID)
	return err
}
