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
	// Checks to see if root domain exists for given sub domain.
	IsExistingRootDomain(ctx context.Context, subDomain string) (bool, error)
	// Gives us the delegation set after creation of hosted zone, which contains the nameserver entries to add to the root domain.
	CreateHostedZone(ctx context.Context, name string, isPublicHostedZone bool) (route53Types.DelegationSet, error)
	// Adds the nameservers to the root domain if root domain exists for given subdomain.
	AddNameserversToRootDomain(ctx context.Context, nameservers []string) error
	// Deletes the hosted zone by domain name.
	DeleteHostedZone(ctx context.Context, name string) error
	// Deletes the associated nameservers of the given domain from the root domain.
	DeleteNameserversFromRootDomain(ctx context.Context, name string) error
}
