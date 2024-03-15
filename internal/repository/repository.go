package repository

import "github.com/nickzer0/GoBoxer/internal/models"

type DatabaseRepo interface {
	//Secrets
	GetAllSecrets() (map[string]string, error)
	GetSecret(name string) (string, error)
	UpdateSecret(name, value string) error

	// Projects
	AddProject(project models.Project, assignTo []string) error
	GetProjectNamesForUsername(username string) ([]string, error)
	GetProjectsForUser(username string) ([]models.Project, error)
	CheckProjectByNumber(number int) (bool, error)
	DeleteProjectByNumber(number int) error
	GetProjectByNumber(number int) (models.Project, error)
	UpdateProject(project models.Project, assignTo []string) error
	GetAllProjects() ([]models.Project, error)

	// Users
	Authenticate(username, tesPass string) (models.User, error)
	GetUsers() ([]models.User, error)
	GetUserIDFromUsername(username string) (int, error)
	GetUserFromID(id int) (models.User, error)
	UpdateUser(user models.User) error
	AddUser(user models.User) error
	DeleteUser(userID int) error

	// Servers
	AddServerToDatabase(server models.Server) (models.Server, error)
	GetServer(id int) (models.Server, error)
	ListAllServers() ([]models.Server, error)
	ListAllServersForUser(user string) ([]models.Server, error)
	UpdateServer(server models.Server) error
	DeleteServerFromDatabase(serverID int) error
	ListAllServersForProject(projectName string) ([]models.Server, error)
	GetServiceDetails() (models.Services, error)

	// Scripts
	AddScript(script models.Script) (models.Script, error)
	RemoveScript(script string) error
	AssignScriptToServer(scriptID, serverID int) error
	GetScriptsForServer(serverID int) ([]string, error)
	ListAllScripts() ([]models.Script, error)
	GetScriptByID(id int) (models.Script, error)
	UpdateScript(script models.Script) error

	// Domains
	GetAllDomains() ([]models.Domains, error)
	AddDomain(domain models.Domains) error
	GetDomainFromDatabase(name string) (models.Domains, error)
	GetDomainById(id int) (models.Domains, error)

	// DNS
	AddDnsRecord(record models.DNS) error
	AddOrIgnoreDnsRecord(record models.DNS) error
	GetDnsRecordsForDomain(domain string) ([]models.DNS, error)
	GetDnsRecordByID(id int) (models.DNS, error)
	UpdateDnsRecord(record models.DNS) error
	DeleteDnsRecord(id int) error
	DeleteDnsRecordsForDomain(domain string) error
	GetAWSHostedZone(domain string) (string, error)

	// Redirectors
	AddDomainRedirector(models.Redirector) (models.Redirector, error)
	RemoveDomainRedirector(redirector models.Redirector) error
	GetDomainRedirector(id int) (models.Redirector, error)
	UpdateDomainRedirector(redirector models.Redirector) error
	GetAllDomainRedirectors() ([]models.Redirector, error)

	// Init
	SetupDatabase() error
}
