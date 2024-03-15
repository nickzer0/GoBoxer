package deploy

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/apenella/go-ansible/pkg/options"
	"github.com/apenella/go-ansible/pkg/playbook"
	"github.com/nickzer0/GoBoxer/internal/models"
)

func RunPlayBook(server models.Server) error {
	rootPassword, err := Repo.GetSecretFromDatabase("root_password")
	if err != nil {
		return err
	}

	ansiblePlaybookConnectionOptions := &options.AnsibleConnectionOptions{
		PrivateKey:    "id_rsa",
		SSHCommonArgs: "-o StrictHostKeyChecking=no",
	}
	extraVar := map[string]interface{}{
		"ansible_user":      "root",
		"ansible_password":  rootPassword,
		"host_key_checking": "False",
	}

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		Inventory: server.IP + ",",
		ExtraVars: extraVar,
	}

	// Wait for SSH to become available with retry limit
	const maxRetries = 10
	var conn net.Conn
	for attempt := 0; attempt < maxRetries; attempt++ {
		conn, err = net.DialTimeout("tcp", net.JoinHostPort(server.IP, "22"), 3*time.Second)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("SSH not available after %d attempts: %v", maxRetries, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	for _, scriptName := range server.Roles {

		playbook := &playbook.AnsiblePlaybookCmd{
			Playbooks:         []string{fmt.Sprintf("scripts/added/%s.yml", scriptName)},
			ConnectionOptions: ansiblePlaybookConnectionOptions,
			Options:           ansiblePlaybookOptions,
			StdoutCallback:    app.AnsibleDebug,
		}

		err := playbook.Run(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// AddSSHUser adds SSH user to servers
func AddSSHUser(server models.Server, user models.User) error {
	rootPassword, err := Repo.GetSecretFromDatabase("root_password")
	if err != nil {
		return err
	}

	ansiblePlaybookConnectionOptions := &options.AnsibleConnectionOptions{
		PrivateKey:    "id_rsa",
		SSHCommonArgs: "-o StrictHostKeyChecking=no",
	}

	extraVar := map[string]interface{}{
		"ansible_user":      "root",
		"host_key_checking": "False",
		"ssh_key":           user.SSHKey,
		"root_password":     rootPassword,
	}

	ansiblePlaybookOptions := &playbook.AnsiblePlaybookOptions{
		Inventory: server.IP + ",",
		ExtraVars: extraVar,
	}

	// Wait for SSH to become available with retry limit
	const maxRetries = 10
	retryCount := 0
	for retryCount < maxRetries {
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(server.IP, "22"), 3*time.Second)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(3 * time.Second)
		retryCount++
	}
	if retryCount == maxRetries {
		return fmt.Errorf("SSH not available after %d retries", maxRetries)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	playbook := &playbook.AnsiblePlaybookCmd{
		Playbooks:         []string{"scripts/default/Access.yml"},
		ConnectionOptions: ansiblePlaybookConnectionOptions,
		Options:           ansiblePlaybookOptions,
		StdoutCallback:    app.AnsibleDebug,
	}

	err = playbook.Run(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil

}

// GetSecretFromDatabase helper function to return a secret from the database
func (m *Repository) GetSecretFromDatabase(secret string) (string, error) {
	return m.DB.GetSecret(secret)
}
