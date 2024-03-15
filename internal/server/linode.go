package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/linode/linodego"
	"github.com/nickzer0/GoBoxer/internal/models"
	"golang.org/x/oauth2"
)

// LinodeCreateServer creates a VPS in Linode based on model
func (m *Repository) LinodeCreateServer(server models.Server) (models.Server, error) {
	var sshKeys []string

	apiKey, err := m.DB.GetSecret("linode")
	if err != nil {
		return server, err
	}

	sshKey, err := m.DB.GetSecret("publicSSHKey")
	if err != nil {
		return server, err
	}

	sshKeys = append(sshKeys, sshKey)

	rootPassword, err := m.DB.GetSecret("root_password")
	if err != nil {
		return server, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiKey})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	linodeClient := linodego.NewClient(oauth2Client)

	poweredOn := true
	instanceOptions := linodego.InstanceCreateOptions{
		Region:         "eu-central",
		Type:           "g6-nanode-1",
		AuthorizedKeys: sshKeys,
		RootPass:       rootPassword,
		Tags:           Tags,
		Booted:         &poweredOn,
	}

	if server.OS == "ubuntu-2204" {
		instanceOptions.Image = "linode/ubuntu22.04"
	}

	returnedServer, err := linodeClient.CreateInstance(ctx, instanceOptions)
	if err != nil {
		log.Println(err)
		return server, err
	}

	for {
		vps, err := linodeClient.GetInstance(ctx, returnedServer.ID)
		if err != nil {
			return server, err
		}
		if vps.Status == "running" {
			server.ProviderID = vps.ID
			server.IP = vps.IPv4[0].String()

			break
		} else {
			time.Sleep(5 * time.Second)
		}
	}

	return server, nil

}

// LinodeDeleteServer destroys a VPS based on serverID
func (m *Repository) LinodeDeleteServer(serverID int) error {
	apiKey, err := m.DB.GetSecret("linode")
	if err != nil {
		return err
	}

	if apiKey == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiKey})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	linodeClient := linodego.NewClient(oauth2Client)

	listOptions := linodego.ListOptions{
		PageOptions: &linodego.PageOptions{},
		PageSize:    500,
		Filter:      "",
	}
	instances, err := linodeClient.ListInstances(ctx, &listOptions)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		for _, tag := range instance.Tags {
			if tag == Tags[0] {
				err := linodeClient.DeleteInstance(ctx, serverID)
				log.Println("Deleting Linode Instance ID:", instance.ID)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// LinodeDeleteAll deletes all VPS hosted on Linode with specified tag
func (m *Repository) LinodeDeleteAll() error {
	apiKey, err := m.DB.GetSecret("linode")
	if err != nil {
		return err
	}

	if apiKey == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiKey})

	oauth2Client := &http.Client{
		Transport: &oauth2.Transport{
			Source: tokenSource,
		},
	}

	linodeClient := linodego.NewClient(oauth2Client)

	listOptions := linodego.ListOptions{
		PageOptions: &linodego.PageOptions{},
		PageSize:    500,
		Filter:      "",
	}
	instances, err := linodeClient.ListInstances(ctx, &listOptions)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		for _, tag := range instance.Tags {
			if tag == Tags[0] {
				err := linodeClient.DeleteInstance(ctx, instance.ID)
				log.Println("Deleting Linode Instance ID:", instance.ID)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
