package controller

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam/types"
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
