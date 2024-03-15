package dbrepo

import (
	"errors"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/nickzer0/GoBoxer/internal/models"
)

// AddProject creates a new project in the database and assigns it to specified users.
func (m *sqliteDBRepo) AddProject(project models.Project, assignTo []string) error {
	exists, err := m.CheckProjectByNumber(project.ProjectNumber)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("project with that ID already exists")
	}

	// Use RETURNING id with QueryRow to get the newly inserted project ID.
	var projectID int
	insertProject := `INSERT INTO projects (project_number, project_name, created_by, notes) VALUES (?, ?, ?, ?) RETURNING id`
	err = m.DB.QueryRow(insertProject, project.ProjectNumber, project.ProjectName, project.CreatedBy, project.Notes).Scan(&projectID)
	if err != nil {
		return err
	}

	for _, username := range assignTo {
		userID, err := m.GetUserIDFromUsername(username)
		if err != nil {
			return err
		}

		// Directly execute without preparing statement as it's a simple insert.
		insertRelation := `INSERT INTO projects_users (projects_id, users_id) VALUES (?, ?)`
		_, err = m.DB.Exec(insertRelation, projectID, userID)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckProjectByNumber checks if a project exists in the database by its number.
func (m *sqliteDBRepo) CheckProjectByNumber(number int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM projects WHERE project_number = ?)"
	err := m.DB.QueryRow(query, number).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetProjectNamesForUsername fetches project names associated with a given username.
func (m *sqliteDBRepo) GetProjectNamesForUsername(username string) ([]string, error) {
	userID, err := m.GetUserIDFromUsername(username)
	if err != nil {
		return nil, err
	}

	var projectNames []string
	query := `
    SELECT p.project_name
    FROM projects p
    INNER JOIN projects_users pu ON pu.projects_id = p.id
    WHERE pu.users_id = ?
    `

	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var name string
	for rows.Next() {
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		projectNames = append(projectNames, name)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return projectNames, nil
}

// DeleteProjectByNumber removes a project identified by its project number from the database.
func (m *sqliteDBRepo) DeleteProjectByNumber(number int) error {
	_, err := m.DB.Exec("DELETE FROM projects WHERE project_number = ?", number)
	if err != nil {
		return err
	}
	return nil
}

// UpdateProject modifies an existing project's details and reassigns its associated users.
func (m *sqliteDBRepo) UpdateProject(project models.Project, assignTo []string) error {
	// Update project details
	_, err := m.DB.Exec(`UPDATE projects SET project_name = ?, notes = ? WHERE id = ?`, project.ProjectName, project.Notes, project.ID)
	if err != nil {
		log.Println("Update project error:", err)
		return err
	}

	// Clear existing user assignments for the project
	_, err = m.DB.Exec(`DELETE FROM projects_users WHERE projects_id = ?`, project.ID)
	if err != nil {
		log.Println("Delete projects_users error:", err)
		return err
	}

	// Reassign users to the project
	for _, user := range assignTo {
		userID, err := m.GetUserIDFromUsername(user)
		if err != nil {
			log.Println("GetUserIDFromUsername error:", err)
			return err
		}

		_, err = m.DB.Exec(`INSERT INTO projects_users (projects_id, users_id) VALUES (?, ?)`, project.ID, userID)
		if err != nil {
			log.Println("Insert into projects_users error:", err)
			return err
		}
	}

	return nil
}

// GetProjectByNumber fetches a project and its assigned users by project number.
func (m *sqliteDBRepo) GetProjectByNumber(number int) (models.Project, error) {
	var project models.Project

	// Query project details
	err := m.DB.QueryRow(`
        SELECT id, project_number, project_name, created_by, notes
        FROM projects
        WHERE project_number = ?
    `, number).Scan(&project.ID, &project.ProjectNumber, &project.ProjectName, &project.CreatedBy, &project.Notes)
	if err != nil {
		return project, err
	}

	// Query assigned users
	rows, err := m.DB.Query(`
        SELECT u.username
        FROM projects_users pu
        JOIN users u ON pu.users_id = u.id
        WHERE pu.projects_id = ?
    `, project.ID)
	if err != nil {
		return project, err
	}
	defer rows.Close()

	var assignedUsers []string
	for rows.Next() {
		var userTemp string
		if err := rows.Scan(&userTemp); err != nil {
			return project, err
		}
		assignedUsers = append(assignedUsers, userTemp)
	}

	project.AssignedTo = assignedUsers

	return project, nil
}

// GetProjectsForUser retrieves all projects assigned to a specified user.
func (m *sqliteDBRepo) GetProjectsForUser(username string) ([]models.Project, error) {
	userID, err := m.GetUserIDFromUsername(username)
	if err != nil {
		return nil, err
	}

	query := `
    SELECT p.id, p.project_number, p.project_name, p.created_by, p.notes, p.created_at
    FROM projects p
    JOIN projects_users pu ON p.id = pu.projects_id
    WHERE pu.users_id = ?
    ORDER BY p.id`

	rows, err := m.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		if err := rows.Scan(&project.ID, &project.ProjectNumber, &project.ProjectName, &project.CreatedBy, &project.Notes, &project.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// GetAllProjects retrieves all projects stored in the database.
func (m *sqliteDBRepo) GetAllProjects() ([]models.Project, error) {
	query := "SELECT id, project_number, project_name, created_by, notes, created_at FROM projects"
	rows, err := m.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		if err := rows.Scan(&project.ID, &project.ProjectNumber, &project.ProjectName, &project.CreatedBy, &project.Notes, &project.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return projects, nil
}
