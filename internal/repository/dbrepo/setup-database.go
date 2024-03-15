package dbrepo

import (
	_ "github.com/mattn/go-sqlite3"
)

// SetupDatabase is used at first run to create the relevant database structure
func (m *sqliteDBRepo) SetupDatabase() error {
	createTableUsers := `CREATE TABLE IF NOT EXISTS users (
		id				INTEGER PRIMARY KEY AUTOINCREMENT,
		username		TEXT,
		first_name		TEXT,
		last_name		TEXT,
		password		TEXT,
		access_level	INTEGER,
		ssh_key			TEXT,
		UNIQUE(username)
	)`

	_, err := m.DB.Exec(createTableUsers)
	if err != nil {
		return err
	}

	createTableSecrets := `CREATE TABLE IF NOT EXISTS secrets (
		id				INTEGER PRIMARY KEY AUTOINCREMENT,
		name			TEXT,
		value			TEXT,
		last_change		TEXT,
		UNIQUE(name)
	)`

	_, err = m.DB.Exec(createTableSecrets)
	if err != nil {
		return err
	}

	createTablePreferences := `CREATE TABLE IF NOT EXISTS preferences (
		id			INTEGER PRIMARY KEY,
		name 		TEXT,
		preference 	TEXT
	)`

	_, err = m.DB.Exec(createTablePreferences)
	if err != nil {
		return err
	}

	createTableProjects := `CREATE TABLE IF NOT EXISTS projects (
		id				INTEGER PRIMARY KEY,
		project_number	INTEGER,
		project_name	TEXT,
		created_by		TEXT,
		notes			TEXT,
		created_at		timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = m.DB.Exec(createTableProjects)
	if err != nil {
		return err
	}

	createTableSessions := `CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		data BLOB NOT NULL,
		expiry REAL NOT NULL
	)`

	_, err = m.DB.Exec(createTableSessions)
	if err != nil {
		return err
	}

	createTableSessionsIndex := `CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions(expiry)`

	_, err = m.DB.Exec(createTableSessionsIndex)
	if err != nil {
		return err
	}

	createTableProjectsUsers := `CREATE TABLE IF NOT EXISTS projects_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		projects_id INTEGER,
		users_id INTEGER,
		FOREIGN KEY (projects_id) REFERENCES projects (id) ON DELETE CASCADE,
		FOREIGN KEY (users_id) REFERENCES users (id) ON DELETE CASCADE
	)`

	_, err = m.DB.Exec(createTableProjectsUsers)
	if err != nil {
		return err
	}

	createTableServers := `CREATE TABLE IF NOT EXISTS servers (
		id 				INTEGER PRIMARY KEY AUTOINCREMENT,
		provider		TEXT,
		server_os		TEXT,
		server_name		TEXT,
		provider_id		INT,
		server_status	TEXT,
		server_ip		TEXT,
		server_project	INT,
		created_by		TEXT,
		created_at		timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP

	)`

	_, err = m.DB.Exec(createTableServers)
	if err != nil {
		return err
	}

	createTableScripts := `CREATE TABLE IF NOT EXISTS scripts (
		id 			INTEGER PRIMARY KEY AUTOINCREMENT,
		name		TEXT,
		icon		TEXT,
		description	TEXT,
		created_by	TEXT,
		created_at	timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = m.DB.Exec(createTableScripts)
	if err != nil {
		return err
	}

	createTableScriptsServers := `CREATE TABLE IF NOT EXISTS scripts_servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id INTEGER,
		server_id INTEGER,
		FOREIGN KEY (script_id) REFERENCES scripts (id) ON DELETE CASCADE,
		FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
	)`

	_, err = m.DB.Exec(createTableScriptsServers)
	if err != nil {
		return err
	}

	createTableDomains := `CREATE TABLE IF NOT EXISTS domains (
		id 			INTEGER PRIMARY KEY AUTOINCREMENT,
		provider	TEXT,
		provider_id TEXT,
		name		TEXT,
		status		TEXT,
		project		INT,
		created_by	TEXT,
		created_at	timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = m.DB.Exec(createTableDomains)
	if err != nil {
		return err
	}

	createTableDns := `CREATE TABLE IF NOT EXISTS dns (
		id 			INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id	TEXT,
		domain		TEXT,
		data		TEXT,
		name		TEXT,
		ttl 		INTEGER,
		type 		TEXT,
		priority 	INTEGER,
		weight 		INTEGER, 
		created_at	timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = m.DB.Exec(createTableDns)
	if err != nil {
		return err
	}

	createTableRedirectors := `CREATE TABLE IF NOT EXISTS redirectors (
		id 					INTEGER PRIMARY KEY AUTOINCREMENT,
		provider_id			TEXT,
		provider			TEXT,
		domain				TEXT,
		url					TEXT,
		status				TEXT,
		project			 	INT,
		created_at			timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`

	_, err = m.DB.Exec(createTableRedirectors)
	if err != nil {
		return err
	}

	return nil

}
