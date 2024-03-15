package models

// User struct for application users and settings
type User struct {
	ID             int
	Username       string
	FirstName      string
	LastName       string
	HashedPassword string
	AccessLevel    int
	SSHKey         string
	Preferences    map[string]string
	Projects       []Project
}