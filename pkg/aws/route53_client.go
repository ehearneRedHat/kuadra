package aws

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
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

// Meaning of potential return options...
// false, nil --> no root domain exists for given subdomain.
// true, nil --> root domain exists for given subdomain.
// false, err --> root domain was given as parameter, unsuccessful query for hosted zones in AWS.
func (wrapper route53Wrapper) IsExistingRootDomain(ctx context.Context, subDomain string) (bool, error) {
	// Check to see if subdomain given is not a root domain
	parts := strings.Split(subDomain, ".")

	len_parts := len(parts)

	if len_parts <= 2 {
		log.Printf("Given subdomain %v is a root domain. A subdomain looks like something.example.com, and a root domain looks like example.com.", subDomain)
		return false, fmt.Errorf("root domain given")
	}

	// If passes check, proceed on...

	// Creates a root domain using the last two parts of the split string.
	rootDomain := parts[len_parts-2] + "." + parts[len_parts-1]

	// Create an array of hosted zones from route53 query.

	hostedZones, err := wrapper.Route53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})

	// Check to see if request was successful
	if err != nil {
		log.Printf("Request was unsuccessful. %v", err)
		return false, err
	}

	// If passes check, proceed on...

	// zoom in on list of hosted zones.
	hostedZonesList := hostedZones.HostedZones

	// Iterate through list of hosted zones and compare domain names for match with root domain
	for _, hostedZone := range hostedZonesList {
		if hostedZone.Name == &rootDomain {
			return true, nil
		}
	}

	// No existing root domain found for given subdomain
	return false, nil
}

func (wrapper route53Wrapper) CreateHostedZone(ctx context.Context, name string, isPublicHostedZone bool) (types.DelegationSet, error) {
	// Check to see if subdomain given is not a root domain
	parts := strings.Split(name, ".")

	len_parts := len(parts)

	// Create hosted zone config from domain.
	hostedZoneConfig := types.HostedZoneConfig{
		PrivateZone: !isPublicHostedZone, // must negate the value as struct was set up with public hosted zone in mind.
	}
	hostedZoneInput := route53.CreateHostedZoneInput{
		Name:             &name,
		HostedZoneConfig: &hostedZoneConfig,
	}

	subdomain := name

	// if root domain...
	if len_parts <= 2 {
		// Create subdomain to check for existing root domain since root domain.
		subdomain = "example" + "." + name
	}

	rootDomainExists, err := wrapper.IsExistingRootDomain(ctx, subdomain)

	if rootDomainExists || err != nil {
		log.Printf("Given domain %v exists in route 53 already. %v", name, err)
		return types.DelegationSet{}, err
	}

	log.Printf("Given subdomain %v is a root domain. Creating Hosted Zone.", name)

	// if sub domain...

	route53Output, err := wrapper.Route53Client.CreateHostedZone(ctx, &hostedZoneInput)

	if err != nil {
		log.Printf("Failed to create hosted zone %v. Here's why: %v", name, err)
	}

	return *route53Output.DelegationSet, nil
}

func AddNameserversToRootDomain(ctx context.Context, nameservers []string) error {
	return nil
}

func DeleteHostedZone(ctx context.Context, name string) error {
	return nil
}

func DeleteNameserversFromRootDomain(ctx context.Context, name string) error {
	return nil
}
