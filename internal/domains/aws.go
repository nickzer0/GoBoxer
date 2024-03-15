package domains

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/nickzer0/GoBoxer/internal/models"
)

// CreateAWSHostedZoneForDomain creates a hosted zone in AWS Route53, then updates
// the NS on the domain provider to point to the hosted zone to allow us
// to easily and quickly update DNS records
func (m *Repository) CreateAWSHostedZoneForDomain(domain models.Domains) {
	awsAccount, err := m.DB.GetSecret("awsaccount")
	if err != nil {
		log.Println("Failed to get AWS account:", err)
		return
	}
	awsSecret, err := m.DB.GetSecret("awssecret")
	if err != nil {
		log.Println("Failed to get AWS secret:", err)
		return
	}

	session, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(awsAccount, awsSecret, ""),
		MaxRetries:  aws.Int(3),
	})
	if err != nil {
		log.Println("Failed to create AWS session:", err)
		return
	}

	svc := route53.New(session)
	input := &route53.CreateHostedZoneInput{
		CallerReference: aws.String(strconv.FormatInt(time.Now().UnixNano(), 10)),
		Name:            aws.String(domain.Name),
	}

	output, err := svc.CreateHostedZone(input)
	if err != nil {
		log.Println("Failed to create hosted zone:", err)
		return
	}

	nameServers := output.DelegationSet.NameServers
	if domain.Provider == "godaddy" {
		m.GoDaddyUpdateNameServer(domain.Name, nameServers)
	}

	for _, nameServer := range nameServers {
		record := models.DNS{
			ProviderID: *output.HostedZone.Id,
			Domain:     domain.Name,
			Data:       *nameServer,
			Name:       "AWS",
			Type:       "NS",
		}
		if err := m.DB.AddDnsRecord(record); err != nil {
			log.Println("Failed to add DNS record:", err)
		}
	}
}

// GetDNSRecordsFromAWS retrieves a list of all the DNS records for the domain
// from Route53 on AWS and returns them as a
func (m *Repository) GetDNSRecordsFromAWS(domain models.Domains) ([]models.DNS, error) {
	awsAccount, err := m.DB.GetSecret("awsaccount")
	if err != nil {
		log.Println("Failed to get AWS account:", err)
		return nil, err
	}
	awsSecret, err := m.DB.GetSecret("awssecret")
	if err != nil {
		log.Println("Failed to get AWS secret:", err)
		return nil, err
	}

	session, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(awsAccount, awsSecret, ""),
		MaxRetries:  aws.Int(3),
	})
	if err != nil {
		log.Println("Failed to create AWS session:", err)
		return nil, err
	}

	zoneID, err := m.DB.GetAWSHostedZone(domain.Name)
	if err != nil {
		log.Println("Failed to get hosted zone ID:", err)
		return nil, err
	}

	svc := route53.New(session)
	input := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}

	var records []models.DNS
	err = svc.ListResourceRecordSetsPages(input, func(page *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
		for _, rec := range page.ResourceRecordSets {
			if *rec.Type == "SOA" || *rec.Type == "NS" {
				continue // Skip SOA and NS records
			}

			if len(rec.ResourceRecords) > 0 {
				record := models.DNS{
					ProviderID: zoneID,
					Domain:     domain.Name,
					Data:       *rec.ResourceRecords[0].Value,
					Name:       *rec.Name,
					Ttl:        int(*rec.TTL),
					Type:       *rec.Type,
				}
				records = append(records, record)
				if err := m.DB.AddOrIgnoreDnsRecord(record); err != nil {
					log.Printf("Failed to add or ignore DNS record: %v", err)
				}
			}
		}
		return !lastPage
	})

	if err != nil {
		log.Printf("Failed to list DNS records: %v", err)
		return nil, fmt.Errorf("failed to list DNS records: %w", err)
	}

	return records, nil
}

func (m *Repository) AddDNSRecordToAWS(domain models.Domains, record models.DNS) error {
	awsaccount, _ := m.DB.GetSecret("awsaccount")
	awssecret, _ := m.DB.GetSecret("awssecret")

	creds := credentials.NewStaticCredentials(awsaccount, awssecret, "")

	session, err := session.NewSession(&aws.Config{
		Credentials: creds,
		MaxRetries:  aws.Int(3),
	})

	if err != nil {
		log.Println(err)
		return err
	}

	svc := route53.New(session)
	zoneID, err := m.DB.GetAWSHostedZone(domain.Name)
	if err != nil {
		log.Println(err)
		return err
	}

	err = m.RemoveAllDNSRecords(zoneID)
	if err != nil {
		log.Println(err)
	}

	var recordName string
	if strings.Contains(record.Name, domain.Name) {
		recordName = record.Name
	} else {
		recordName = record.Name + "." + domain.Name
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(recordName),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: aws.String(record.Data),
							},
						},
						TTL:           aws.Int64(int64(record.Ttl)),
						Type:          aws.String(record.Type),
						Weight:        aws.Int64(int64(record.Weight)),
						SetIdentifier: aws.String(strconv.FormatInt(time.Now().UnixNano(), 10)),
					},
				},
			},
		},
		HostedZoneId: aws.String(zoneID),
	}

	_, err = svc.ChangeResourceRecordSets(input)
	if err != nil {
		log.Println(err)
		return err
	}

	err = m.DB.AddDnsRecord(record)
	if err != nil {
		log.Println(err)
		return err
	}

	log.Printf("DNS record %s added to AWS", record.Name)

	return nil
}

// RemoveAllDNSRecords removes all the DNS entries from the AWS hosted zone with
// the given ID.
func (m *Repository) RemoveAllDNSRecords(zoneID string) error {
	log.Println("=== Deleting All DNS Records for Zone", zoneID)
	awsaccount, _ := m.DB.GetSecret("awsaccount")
	awssecret, _ := m.DB.GetSecret("awssecret")

	creds := credentials.NewStaticCredentials(awsaccount, awssecret, "")

	session, err := session.NewSession(&aws.Config{
		Credentials: creds,
		MaxRetries:  aws.Int(3),
	})

	if err != nil {
		log.Println(err)
		return err
	}

	svc := route53.New(session)

	// Get the list of DNS records in the hosted zone
	listParams := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}
	listResp, err := svc.ListResourceRecordSets(listParams)
	if err != nil {
		return err
	}

	// Create a change batch to remove the DNS records
	changeBatch := &route53.ChangeBatch{
		Changes: make([]*route53.Change, 0, len(listResp.ResourceRecordSets)),
	}
	for _, record := range listResp.ResourceRecordSets {
		log.Println(&record)
		// Create a change to remove the DNS record
		change := &route53.Change{
			Action:            aws.String(route53.ChangeActionDelete),
			ResourceRecordSet: record,
		}
		changeBatch.Changes = append(changeBatch.Changes, change)
	}

	// Submit the change batch to remove the DNS records
	changeParams := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  changeBatch,
		HostedZoneId: aws.String(zoneID),
	}
	_, err = svc.ChangeResourceRecordSets(changeParams)
	return err
}
