package aws

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/emicklei/go-restful/v3/log"
)

type route53Wrapper struct {
	Route53Client *route53.Client
}

func NewRoute53Wrapper() (*route53Wrapper, error) {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		return nil, err
	}

	route53Wrapper := route53Wrapper{
		Route53Client: route53.NewFromConfig(sdkConfig),
	}
	return &route53Wrapper, nil
}

// Checks to see if domain exists.
func (route53Wrapper route53Wrapper) IsExistingDomain(ctx context.Context, domain string) (bool, error) {
	// Grab list of existing hosted zones
	hostedZonesList, err := route53Wrapper.Route53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		log.Printf("Failed to grab hosted zones from AWS. Here's why: %v", err)
		return false, err
	}
	// Sort list of hosted zones to contain just the hosted zone name.
	hostedZoneNameList := hostedZonesList.HostedZones
	// Check to see if domain exists in list of hosted zones.
	for _, hostedZone := range hostedZoneNameList {
		if *hostedZone.Name == domain+"." {
			return true, nil
		}
	}
	// No match found
	return false, nil
}

// Creates a hosted zone.

func (route53Wrapper route53Wrapper) CreateHostedZone(ctx context.Context, name string, isPrivateHostedZone bool) error {
	// Check to see if hosted zone exists
	isExistingDomain, err := route53Wrapper.IsExistingDomain(ctx, name)
	if isExistingDomain {
		log.Printf("Hosted Zone %v already exists.", name)
		return nil
	}

	if err != nil {
		return err
	}

	// Continue if no errors...

	// Create unique string (time)

	timeNow := time.Now().Format("2006-01-02 15:04:05")

	// Create Hosted Zone
	_, err = route53Wrapper.Route53Client.CreateHostedZone(ctx, &route53.CreateHostedZoneInput{
		CallerReference: &timeNow,
		Name:            &name,
		HostedZoneConfig: &types.HostedZoneConfig{
			PrivateZone: isPrivateHostedZone,
		},
	})

	if err != nil {
		return err
	}

	// Everything went well... *STUPENDOUS* :0
	return nil

}

// Creates a hosted zone and attaches its nameserver records to a given root domain.
func (route53Wrapper route53Wrapper) CreateHostedZoneRootDomain(ctx context.Context, name string, rootDomain string, isPrivateHostedZone bool) error {
	// Check to see if hosted zone exists
	isExistingDomain, err := route53Wrapper.IsExistingDomain(ctx, name)
	if isExistingDomain {
		log.Printf("Hosted Zone %v already exists.", name)
		return nil
	}

	if err != nil {
		return err
	}

	// Check to see if root hosted zone exists
	isExistingDomain, err = route53Wrapper.IsExistingDomain(ctx, rootDomain)
	if !isExistingDomain {
		log.Printf("Hosted Zone %v does not exist. Creating root domain.", rootDomain)
		// Create unique string (time)
		timeNow := time.Now().Format("2006-01-02 15:04:05")
		// Create Hosted Zone
		_, err1 := route53Wrapper.Route53Client.CreateHostedZone(ctx, &route53.CreateHostedZoneInput{
			CallerReference: &timeNow,
			Name:            &rootDomain,
			HostedZoneConfig: &types.HostedZoneConfig{
				PrivateZone: isPrivateHostedZone,
			},
		})
		if err1 != nil {
			return err1
		}
	}

	if err != nil {
		return err
	}

	// Continue on if hosted zone does not exist and the root hosted zone exists

	// Create unique string (time)

	timeNow := time.Now().Format("2006-01-02 15:04:05")

	// Create Hosted Zone
	_, err = route53Wrapper.Route53Client.CreateHostedZone(ctx, &route53.CreateHostedZoneInput{
		CallerReference: &timeNow,
		Name:            &name,
		HostedZoneConfig: &types.HostedZoneConfig{
			PrivateZone: isPrivateHostedZone,
		},
	})

	if err != nil {
		return err
	}

	// Everything went well... *NOICE* :D
	return nil
}

// Adds the nameservers to the root domain if root domain exists for given subdomain.
func (route53Wrapper route53Wrapper) AddNameserverRecordsToDomain(ctx context.Context, domain string, nameservers []string) error {
	return nil
}

// Get Delegation Set for given hosted zone.
func (route53Wrapper route53Wrapper) GetDelegationSet(ctx context.Context, hostedZoneName string) (types.DelegationSet, error) {
	return types.DelegationSet{}, nil
}

// Lists the nameservers for a given hosted zone.
func (route53Wrapper route53Wrapper) ListNameservers(ctx context.Context, hostedZoneName string) ([]string, error) {
	return []string{}, nil
}

// Deletes the hosted zone by domain name.
func (route53Wrapper route53Wrapper) DeleteHostedZone(ctx context.Context, hostedZoneName string) error {
	return nil
}

// Deletes the nameserver record of the hosted zone.
func (route53Wrapper route53Wrapper) DeleteNameserverRecordFromHostedZone(ctx context.Context, hostedZoneName string, nameservers []string) error {
	return nil
}
