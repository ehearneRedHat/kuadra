package controller

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/smithy-go/middleware"
)

type IamWrapper interface {
	IsExistingUser(ctx context.Context, userName string) (bool, error)
	HasLoginProfile(ctx context.Context, userName string) (bool, error)
	HasAccessKey(ctx context.Context, userName string) (bool, error)
	ListGroupsForUser(ctx context.Context, userName string) ([]types.Group, error)
	CreateUserIfNotExists(ctx context.Context, userName string) error
	CreateLoginProfileIfNotExists(ctx context.Context, password string, userName string, passwordResetRequired bool) error
	CreateAccessKeyPair(ctx context.Context, userName string) (*types.AccessKey, error)
	AddUserToGroup(ctx context.Context, groupName string, userName string) (middleware.Metadata, error)
	RemoveUserFromGroup(ctx context.Context, groupName string, userName string) (middleware.Metadata, error)
	DeleteUser(ctx context.Context, userName string) error
	DeleteLoginProfileIfExists(ctx context.Context, userName string) error
	ListAccessKeys(ctx context.Context, userName string) ([]types.AccessKeyMetadata, error)
	DeleteAccessKeyIfExists(ctx context.Context, userName string, keyId string) error
}

type Route53Wrapper interface {
	// Checks to see if domain exists.
	IsExistingDomain(ctx context.Context, domain string) (bool, error)
	// Creates a Hosted Zone.
	CreateHostedZone(ctx context.Context, name string, isPrivateHostedZone bool) error
	// Also creates a Hosted Zone, but also attaches its nameserver records to the given root domain.
	CreateHostedZoneRootDomain(ctx context.Context, name string, rootDomain string, isPrivateHostedZone bool) error
	// Adds the nameservers to the root domain if root domain exists for given subdomain.
	AddNameserverRecordsToDomain(ctx context.Context, domain string, nameservers []string) error
	// Get Delegation Set for given hosted zone.
	GetDelegationSet(ctx context.Context, hostedZoneName string) (route53Types.DelegationSet, error)
	// Lists the nameservers for a given hosted zone.
	ListNameservers(ctx context.Context, hostedZoneName string) ([]string, error)
	// Deletes the hosted zone by domain name.
	DeleteHostedZone(ctx context.Context, hostedZoneName string) error
	// Deletes the nameserver record of the hosted zone.
	DeleteNameserverRecordFromHostedZone(ctx context.Context, hostedZoneName string, nameservers []string) error
}
