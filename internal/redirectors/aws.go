package redirectors

import (
	"log"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/nickzer0/GoBoxer/internal/models"
)

// CreateCloudfrontDomain takes a URL and creates a Cloudfront domain redirector in AWS.
func (m *Repository) CreateCloudfrontDomain(redirector models.Redirector) (models.Redirector, error) {
	awsAccount, err := m.DB.GetSecret("awsaccount")
	if err != nil {
		log.Printf("Failed to get AWS account secret: %v", err)
		return models.Redirector{}, err
	}

	awsSecret, err := m.DB.GetSecret("awssecret")
	if err != nil {
		log.Printf("Failed to get AWS secret: %v", err)
		return models.Redirector{}, err
	}

	creds := credentials.NewStaticCredentials(awsAccount, awsSecret, "")

	session, err := session.NewSession(&aws.Config{
		Credentials: creds,
		MaxRetries:  aws.Int(3),
	})

	if err != nil {
		return redirector, err
	}

	svc := cloudfront.New(session)

	input := &cloudfront.CreateDistributionInput{
		DistributionConfig: &cloudfront.DistributionConfig{
			CallerReference: aws.String(strconv.FormatInt(time.Now().UnixNano(), 10)),
			Comment:         aws.String(""),
			DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
				AllowedMethods: &cloudfront.AllowedMethods{
					Items:    aws.StringSlice([]string{"GET", "HEAD", "POST", "PUT", "PATCH", "OPTIONS", "DELETE"}),
					Quantity: aws.Int64(7),
				},
				MinTTL:               aws.Int64(0),
				TargetOriginId:       aws.String(redirector.Domain),
				ViewerProtocolPolicy: aws.String("allow-all"),
				ForwardedValues: &cloudfront.ForwardedValues{
					Cookies: &cloudfront.CookiePreference{
						Forward: aws.String("all"),
					},
					Headers: &cloudfront.Headers{
						Quantity: aws.Int64(0),
					},
					QueryString: aws.Bool(true),
					QueryStringCacheKeys: &cloudfront.QueryStringCacheKeys{
						Quantity: aws.Int64(0),
					},
				},
				TrustedSigners: &cloudfront.TrustedSigners{
					Enabled:  aws.Bool(false),
					Quantity: aws.Int64(0),
				},
			},
			Enabled: aws.Bool(true),
			Origins: &cloudfront.Origins{
				Quantity: aws.Int64(1),
				Items: []*cloudfront.Origin{
					{
						Id:         aws.String(redirector.Domain),
						DomainName: aws.String(redirector.Domain),
						OriginPath: aws.String(""),
						CustomHeaders: &cloudfront.CustomHeaders{
							Quantity: aws.Int64(0),
						},
						CustomOriginConfig: &cloudfront.CustomOriginConfig{
							HTTPPort:             aws.Int64(80),
							HTTPSPort:            aws.Int64(443),
							OriginProtocolPolicy: aws.String("match-viewer"),
							OriginSslProtocols: &cloudfront.OriginSslProtocols{
								Items:    aws.StringSlice([]string{"TLSv1", "TLSv1.1", "TLSv1.2"}),
								Quantity: aws.Int64(3),
							},
						},
					},
				},
			},
			OriginGroups: &cloudfront.OriginGroups{
				Quantity: aws.Int64(0),
			},
		},
	}

	result, err := svc.CreateDistribution(input)
	if err != nil {
		return redirector, err
	}

	newRedirector := models.Redirector{
		ProviderID: *result.Distribution.Id,
		Provider:   "AWS",
		URL:        *result.Distribution.DomainName,
		Domain:     redirector.Domain,
		Status:     "Creating",
		Project:    redirector.Project,
		CreatedAt:  time.Now(),
	}

	getDistributionInput := &cloudfront.GetDistributionInput{
		Id: &newRedirector.ProviderID,
	}

	// TODO: Change this to pass by reference
	returnedRedirector, err := m.DB.AddDomainRedirector(newRedirector)
	if err != nil {
		return newRedirector, err
	}

	go m.WaitForCloudfrontDeployedRoutine(svc, getDistributionInput, returnedRedirector)

	return returnedRedirector, nil
}

// WaitForCloudfrontDeployedRoutine polls AWS CloudFront until the specified distribution is fully deployed,
// then updates the redirector's status in the database to "Ready".
func (m *Repository) WaitForCloudfrontDeployedRoutine(svc *cloudfront.CloudFront, input *cloudfront.GetDistributionInput, redirector models.Redirector) {
	if err := svc.WaitUntilDistributionDeployed(input); err != nil {
		log.Printf("Error waiting for CloudFront distribution to be deployed: %v", err)
		return
	}
	log.Printf("CloudFront distribution %s is ready.", *input.Id)

	redirector.Status = "Ready"
	if err := m.DB.UpdateDomainRedirector(redirector); err != nil {
		log.Printf("Error updating redirector status in database: %v", err)
	} else {
		log.Printf("Redirector status updated to 'Ready' in database for distribution %s.", *input.Id)
	}
}

// DeleteCloudfrontDistribution deletes the Cloudfront distribution from AWS and the database.
func (m *Repository) DeleteCloudfrontDistribution(redirector models.Redirector) error {
	awsAccount, err := m.DB.GetSecret("awsaccount")
	if err != nil {
		log.Printf("Failed to get AWS account secret: %v", err)
		return err
	}

	awsSecret, err := m.DB.GetSecret("awssecret")
	if err != nil {
		log.Printf("Failed to get AWS secret: %v", err)
		return err
	}

	creds := credentials.NewStaticCredentials(awsAccount, awsSecret, "")

	session, err := session.NewSession(&aws.Config{
		Credentials: creds,
		MaxRetries:  aws.Int(3),
	})
	if err != nil {
		return err
	}

	svc := cloudfront.New(session)
	getDistributionInput := &cloudfront.GetDistributionInput{
		Id: &redirector.ProviderID,
	}

	getDistributionOutput, err := svc.GetDistribution(getDistributionInput)
	if err != nil {
		log.Println(err)
	}

	getDistributionOutput.Distribution.DistributionConfig.Enabled = aws.Bool(false)
	updateInput := &cloudfront.UpdateDistributionInput{
		DistributionConfig: getDistributionOutput.Distribution.DistributionConfig,
		Id:                 &redirector.ProviderID,
		IfMatch:            getDistributionOutput.ETag,
	}

	updateDistributionOutput, err := svc.UpdateDistribution(updateInput)
	if err != nil {
		log.Println(err)
	}

	deleteInput := &cloudfront.DeleteDistributionInput{
		Id:      &redirector.ProviderID,
		IfMatch: updateDistributionOutput.ETag,
	}

	go DeleteDistributionRoutine(svc, deleteInput)

	if err := m.DB.RemoveDomainRedirector(redirector); err != nil {
		return err
	}

	return nil
}

// DeleteDistributionRoutine is to delete the distribution, it can take several minutes to disable the distribution on AWS.
// Once it has successfully been disabled, it is then deleted.
func DeleteDistributionRoutine(svc *cloudfront.CloudFront, deleteInput *cloudfront.DeleteDistributionInput) {
	for {
		_, err := svc.DeleteDistribution(deleteInput)
		if err != nil {
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}
}

func (m *Repository) ResyncDistribution(redirector models.Redirector) {
	awsAccount, err := m.DB.GetSecret("awsaccount")
	if err != nil {
		log.Printf("Failed to get AWS account secret: %v", err)
		return
	}

	awsSecret, err := m.DB.GetSecret("awssecret")
	if err != nil {
		log.Printf("Failed to get AWS secret: %v", err)
		return
	}

	creds := credentials.NewStaticCredentials(awsAccount, awsSecret, "")
	session, err := session.NewSession(&aws.Config{
		Credentials: creds,
		MaxRetries:  aws.Int(3),
	})
	if err != nil {
		log.Printf("Failed to create AWS session: %v", err)
		return
	}

	svc := cloudfront.New(session)
	getDistributionInput := &cloudfront.GetDistributionInput{
		Id: &redirector.ProviderID,
	}

	// Use a goroutine to asynchronously wait for the CloudFront distribution to be deployed
	go m.WaitForCloudfrontDeployedRoutine(svc, getDistributionInput, redirector)
}
