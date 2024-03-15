package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log"
	random "math/rand"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh"

	"github.com/alexedwards/scs/sqlite3store"
	"github.com/alexedwards/scs/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nickzer0/GoBoxer/internal/config"
	"github.com/nickzer0/GoBoxer/internal/deploy"
	"github.com/nickzer0/GoBoxer/internal/domains"
	"github.com/nickzer0/GoBoxer/internal/driver"
	"github.com/nickzer0/GoBoxer/internal/handlers"
	"github.com/nickzer0/GoBoxer/internal/helpers"
	"github.com/nickzer0/GoBoxer/internal/models"
	"github.com/nickzer0/GoBoxer/internal/redirectors"
	"github.com/nickzer0/GoBoxer/internal/server"
	"github.com/spf13/viper"
	"golang.org/x/net/websocket"
)

var session *scs.SessionManager
var wsServer *models.WebsocketServer
var handlerRepo *handlers.Repository

func setup() (string, error) {
	viper.SetConfigName("config.yaml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	var httpPort, domain, dbDataFile, ansibleDebug string
	var inProduction bool

	// Check if httpPort exists and is a string
	if viper.IsSet("httpPort") {
		httpPort = viper.GetString("httpPort")
	} else {
		log.Fatalf("httpPort is not set in the configuration")
	}

	// Check if domain exists and is a string
	if viper.IsSet("domain") {
		domain = viper.GetString("domain")
	} else {
		log.Fatalf("domain is not set in the configuration")
	}

	// Check if inProduction exists and is a bool
	if viper.IsSet("inProduction") {
		inProduction = viper.GetBool("inProduction")
	} else {
		log.Fatalf("inProduction is not set in the configuration")
	}

	// Check if dbDataFile exists and is a string
	if viper.IsSet("dbDataFile") {
		dbDataFile = viper.GetString("dbDataFile")
	} else {
		log.Fatalf("dbDataFile is not set in the configuration")
	}

	// Check if inProduction exists and is a bool
	if viper.IsSet("ansibleDebug") {
		if viper.GetBool("ansibleDebug") {
			ansibleDebug = "default"
		} else {
			ansibleDebug = "null"
		}
	} else {
		log.Fatalf("ansibleDebug is not set in the configuration")
	}

	db, err := driver.ConnectSQL(dbDataFile)
	if err != nil {
		log.Fatal("could not initialize database:", err)
	}

	// session
	session = scs.New()
	session.Store = sqlite3store.New(db.SQL)
	session.Lifetime = time.Duration(48 * time.Hour)
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = false
	sqlite3store.NewWithCleanupInterval(db.SQL, 30*time.Minute)

	// Configure app config
	a := config.AppConfig{
		DB:           db,
		Session:      session,
		InProduction: inProduction,
		AnsibleDebug: ansibleDebug,
		Domain:       domain,
		Version:      goBoxerVersion,
	}

	app = a

	// Initialize websocket server
	wsServer = &models.WebsocketServer{
		Conns: make(map[string]*websocket.Conn),
	}

	// Handler repo
	handlerRepo = handlers.NewRepo(&app, db, wsServer)
	handlers.NewHandlers(handlerRepo, &app)

	// Server repo
	serversRepo := server.NewRepo(&app, db)
	server.NewServers(serversRepo, &app)

	// Domains repo
	domainsRepo := domains.NewRepo(&app, db)
	domains.NewDomains(domainsRepo, &app)

	// Redirector repo
	redirectorsRepo := redirectors.NewRepo(&app, db)
	redirectors.NewRedirectors(redirectorsRepo, &app)

	// Deploy repo
	deployRepo := deploy.NewRepo(&app, db)
	deploy.NewDeploy(deployRepo, &app)

	// Helpers repo
	helpers.NewHelpers(&app)

	// Set up database
	err = handlerRepo.DB.SetupDatabase()
	if err != nil {
		log.Fatal("Cannot set up database:", err)
	}

	if err = CheckSSHConfig(); err != nil {
		return httpPort, err
	}

	if err = CheckRootPassword(); err != nil {
		return httpPort, err
	}

	if err = CheckAdminUserExists(); err != nil {
		return httpPort, err
	}

	// Preference map
	preferenceMap = make(map[string]string)
	preferenceMap["version"] = goBoxerVersion
	app.PreferenceMap = preferenceMap

	return httpPort, err

}

// CheckSSHConfig checks if private and public key exist in secrets table, and generates them if not
func CheckSSHConfig() error {
	privateKey, err := handlerRepo.DB.GetSecret("private_key")
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to fetch private key: %v", err)
	}

	publicKey, err := handlerRepo.DB.GetSecret("sshkey")
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to fetch public key: %v", err)
	}

	if privateKey == "" || publicKey == "" {
		log.Println("No Root SSH Keys detected. Generating new ones.")
		log.Println("Make sure new keys are added to cloud providers in Settings after adding API keys!")

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return fmt.Errorf("failed to generate private key: %v", err)
		}

		privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
		privateKeyPEM := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		}
		privateKeyStr := string(pem.EncodeToMemory(privateKeyPEM))
		publicKey := &privateKey.PublicKey

		publicKeySSH, err := ssh.NewPublicKey(publicKey)
		if err != nil {
			return fmt.Errorf("error converting public key: %v", err)
		}

		publicKeyStr := string(ssh.MarshalAuthorizedKey(publicKeySSH))

		if err = os.WriteFile("id_rsa", []byte(privateKeyStr), 0400); err != nil {
			return fmt.Errorf("failed to write private key to disk: %v", err)
		}

		if err = handlerRepo.DB.UpdateSecret("private_key", privateKeyStr); err != nil {
			return err
		}
		if err = handlerRepo.DB.UpdateSecret("sshkey", publicKeyStr); err != nil {
			return err
		}
	}

	return nil
}

// CheckRootPassword generates and updates root password in config.yaml
func CheckRootPassword() error {
	rootPassword, err := handlerRepo.DB.GetSecret("root_password")
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to fetch root password: %v", err)
	}

	if rootPassword == "" {
		log.Println("No default Root password configured. Generating new one.")

		const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
		randomString := make([]byte, 16)
		// Generate random string based on charset
		for i := range randomString {
			randomString[i] = charset[random.Intn(len(charset))]
		}

		// Update root password in secrets table
		if err := handlerRepo.DB.UpdateSecret("root_password", string(randomString)); err != nil {
			return fmt.Errorf("failed to add root password: %v", err)
		}
	}
	return nil
}

// AddDefaultAdminUser adds default admin user to database
func CheckAdminUserExists() error {
	// Check if users table is empty
	users, err := handlerRepo.DB.GetUsers()
	if err != nil {
		return fmt.Errorf("error looking up users in database: %v", err)
	}

	if len(users) != 0 {
		return nil
	}

	log.Println("No default portal user configured, generating credentials.")
	username := "admin"

	// Generate random password
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	password := make([]byte, 16)
	for i := range password {
		password[i] = charset[random.Intn(len(charset))]
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return fmt.Errorf("failed to generate password: %v", err)
	}
	var user = models.User{
		Username:       username,
		FirstName:      username,
		HashedPassword: string(hashedPassword),
		AccessLevel:    10,
	}

	// Add user to DB
	if err = handlerRepo.DB.AddUser(user); err != nil {
		return fmt.Errorf("failed to add new admin user to DB: %v", err)
	}

	log.Println("MAKE NOTE OF THIS PASSWORD, IT WILL NOT BE DISPLAYED AGAIN!")
	log.Printf("Added new user:\n\tUsername: %s\n\tPassword: %s", username, password)

	return nil
}
