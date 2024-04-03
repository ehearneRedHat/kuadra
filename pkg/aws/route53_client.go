package aws

import (
	"context"
	"strings"
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

	// Check to see if subdomain contains root domain

	if !strings.Contains(name, rootDomain) {
		log.Printf("Unable to continue, as %v is not a subdomain of %v.", name, rootDomain)
		return ctx.Err()
	}

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
		timeNow := time.Now().Format("2006-01-02 15:04:05.000")
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

	timeNow := time.Now().Format("2006-01-02 15:04:05.000")

	// Create Hosted Zone
	domain, err := route53Wrapper.Route53Client.CreateHostedZone(ctx, &route53.CreateHostedZoneInput{
		CallerReference: &timeNow,
		Name:            &name,
		HostedZoneConfig: &types.HostedZoneConfig{
			PrivateZone: isPrivateHostedZone,
		},
	})

	if err != nil {
		return err
	}

	// Get nameservers for hosted zone.
	nameserversHostedZone := domain.DelegationSet.NameServers

	// Add nameserver record to root hosted zone.
	err = route53Wrapper.AddNameserverRecordsToDomain(ctx, rootDomain, name, nameserversHostedZone)

	// Check to see if err
	if err != nil {
		log.Printf("An error occurred while trying to add the nameserver records of %v to root hosted zone %v. Here's why: %v", rootDomain, name, err)
		return err
	}

	// Everything went well... *NOICE* :D
	return nil
}

// Adds the nameservers to the root domain if root domain exists for given subdomain.
func (route53Wrapper route53Wrapper) AddNameserverRecordsToDomain(ctx context.Context, domain string, recordName string, nameservers []string) error {
	// Check to see if hosted zone exists
	domainExists, err := route53Wrapper.IsExistingDomain(ctx, domain)

	if !domainExists {
		log.Printf("Hosted zone %v does not exist - unable to add nameserver records to domain %v.", domain)
		return nil
	}

	if err != nil {
		log.Printf("Unable to check if domain exists for adding nameserver - here's why: %v", err)
		return err
	}

	// If checks are good, proceed...

	// Get Id for hosted zone to add nameserver record to.

	id, err := route53Wrapper.GetHostedZoneId(ctx, domain)

	if err != nil {
		return err
	}

	// Create resource record
	resourceRecord := []types.ResourceRecord{}

	// Iterate through array of nameservers to add them to resource record list.
	for _, a := range nameservers {
		nameserver := a
		// Append resource record
		resourceRecord = append(resourceRecord, types.ResourceRecord{
			Value: &nameserver,
		})
	}

	// Also, create a list of the required changes (nameserver record to be added.)

	changes := []types.Change{}

	// create ttl
	ttl := int64(300)

	// Append change.
	changes = append(changes, types.Change{
		Action: "CREATE",
		ResourceRecordSet: &types.ResourceRecordSet{
			Name:            &recordName,
			Type:            "NS",
			ResourceRecords: resourceRecord,
			TTL:             &ttl,
		},
	})

	// Now, make a request to add nameserver record to hosted zone.
	_, err = route53Wrapper.Route53Client.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: changes,
		},
		HostedZoneId: &id,
	})

	if err != nil {
		log.Printf("Unable to change resource record sets for hosted zone %v. Here's why: %v", domain, err)
		return err
	}

	// Nameserver records have been modified successfully.
	return nil
}

// Get Delegation Set for given hosted zone.
func (route53Wrapper route53Wrapper) GetDelegationSet(ctx context.Context, hostedZoneName string) (types.DelegationSet, error) {
	// check to see if hosted zone exists
	hostedZoneExists, err := route53Wrapper.IsExistingDomain(ctx, hostedZoneName)

	if !hostedZoneExists {
		log.Printf("Hosted zone %v does not exist - cannot get delegation set", hostedZoneName)
		return types.DelegationSet{}, nil
	}

	if err != nil {
		log.Printf("Error occurred while checking for hosted zone - here is why: %v", err)
		return types.DelegationSet{}, nil
	}

	// if checks pass, proceed...

	// Grab list of existing hosted zones
	hostedZonesList, err := route53Wrapper.Route53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		log.Printf("Failed to grab hosted zones from AWS. Here's why: %v", err)
		return types.DelegationSet{}, err
	}
	// Sort list of hosted zones to contain just the hosted zone name.
	hostedZoneNameList := hostedZonesList.HostedZones
	id := ""
	// Check to see if domain exists in list of hosted zones.
	for _, hostedZone := range hostedZoneNameList {
		if *hostedZone.Name == hostedZoneName+"." {
			id = *hostedZone.Id
			break
		}
	}

	if id != "" {
		hostedZoneOutput, err := route53Wrapper.Route53Client.GetHostedZone(ctx, &route53.GetHostedZoneInput{
			Id: &id,
		})
		if err == nil {
			return *hostedZoneOutput.DelegationSet, nil
		}
	}

	return types.DelegationSet{}, err
}

// Lists the nameservers for a given hosted zone.
func (route53Wrapper route53Wrapper) ListNameservers(ctx context.Context, hostedZoneName string) ([]string, error) {
	// Check to see if hosted zone exists...
	isExistingDomain, err := route53Wrapper.IsExistingDomain(ctx, hostedZoneName)
	if !isExistingDomain {
		log.Printf("Unable to check for name servers as %v does not exist.", hostedZoneName)
		return []string{}, nil
	}

	if err != nil {
		return []string{}, err
	}

	// If checks pass, proceed...

	// First, get delegation set.

	delegationSet, err := route53Wrapper.GetDelegationSet(ctx, hostedZoneName)

	// check to see if error...

	if err != nil {
		log.Printf("Unable to retrieve delegation set to list the nameservers for %v. Here's why: %v", hostedZoneName, err)
		return []string{}, err
	}

	// If checks pass, proceed...

	// take nameservers from delegation set
	nameservers := delegationSet.NameServers

	return nameservers, nil
}

// Deletes the hosted zone by domain name.
func (route53Wrapper route53Wrapper) DeleteHostedZone(ctx context.Context, hostedZoneName string) error {
	// First, check if hosted zone exists...

	isExistingDomain, err := route53Wrapper.IsExistingDomain(ctx, hostedZoneName)

	if !isExistingDomain {
		log.Printf("Hosted Zone %v does not exist. Unable to delete...", hostedZoneName)
	}
	if err != nil {
		log.Printf("An error occurred while checking to see if the hosted zone %v exists. Here's why --> %v", hostedZoneName, err)
		return err
	}

	// If checks pass, proceed...

	// Get id of hosted zone

	id, err := route53Wrapper.GetHostedZoneId(ctx, hostedZoneName)

	if err != nil {
		log.Printf("An error occurred while trying the get the id of %v. Here's why: %v", hostedZoneName, err)
		return err
	}

	// If checks pass, proceed...

	deleteHostedZoneInput := route53.DeleteHostedZoneInput{
		Id: &id,
	}

	_, err = route53Wrapper.Route53Client.DeleteHostedZone(ctx, &deleteHostedZoneInput)

	if err != nil {
		log.Printf("An error occurred while deleting the hosted zone %v. Here's the error: %v", hostedZoneName, err)
		return err
	}

	// Delete was successful!!!
	return nil
}

// Deletes the nameserver record of the hosted zone.
func (route53Wrapper route53Wrapper) DeleteNameserverRecordFromHostedZone(ctx context.Context, hostedZoneName string, recordName string) error {

	// Get nameserver records for record name
	nameservers, err := route53Wrapper.ListNameservers(ctx, recordName)

	if err != nil {
		return err
	}

	// Create resource record array
	resourceRecords := []types.ResourceRecord{}

	// Iterate through nameserver list, appending the nameserver to the list of resource records.
	for _, a := range nameservers {
		// Create value string
		value := "<Value>" + a + "</Value>"
		// Create resource record
		resourceRecord := types.ResourceRecord{
			Value: &value,
		}
		resourceRecords = append(resourceRecords, resourceRecord)
	}

	// Create resource record set
	resourceRecordSet := types.ResourceRecordSet{
		Name:            &recordName,
		Type:            "NS",
		ResourceRecords: resourceRecords,
	}

	// Create change
	change := types.Change{
		Action:            "DELETE",
		ResourceRecordSet: &resourceRecordSet,
	}

	// Create changes array and append change.

	changes := []types.Change{}

	changes = append(changes, change)

	// Create change batch
	changeBatch := types.ChangeBatch{
		Changes: changes,
	}

	// Get id for hosted zone
	id, err := route53Wrapper.GetHostedZoneId(ctx, hostedZoneName)

	if err != nil {
		return err
	}

	// Create record set input
	recordSetInput := route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &changeBatch,
		HostedZoneId: &id,
	}

	// Delete nameserver records...
	_, err = route53Wrapper.Route53Client.ChangeResourceRecordSets(ctx, &recordSetInput)

	if err != nil {
		return err
	}

	// Nameserver records deleted successfully!
	return nil
}

// Gets the id of a hosted zone
func (route53Wrapper route53Wrapper) GetHostedZoneId(ctx context.Context, hostedZoneName string) (string, error) {
	// Check to see if hosted zone exists
	isExistingDomain, err := route53Wrapper.IsExistingDomain(ctx, hostedZoneName)

	if !isExistingDomain {
		log.Printf("Unable to get hosted zone id as %v does not exist", hostedZoneName)
	}

	if err != nil {
		return "", err
	}

	// If checks pass, proceed...

	// Grab list of existing hosted zones
	hostedZonesList, err := route53Wrapper.Route53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
	if err != nil {
		log.Printf("Failed to grab hosted zones from AWS. Here's why: %v", err)
		return "", err
	}

	// Sort list of hosted zones to contain just the hosted zone name.
	hostedZoneNameList := hostedZonesList.HostedZones
	id := ""
	// Check to see if domain exists in list of hosted zones.
	for _, hostedZone := range hostedZoneNameList {
		if *hostedZone.Name == hostedZoneName+"." {
			id = *hostedZone.Id
			break
		}
	}

	return id, nil
}
