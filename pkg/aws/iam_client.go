package aws

import (
	"context"
	"errors"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"

	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
)

func isNoSuchEntityException(err error) bool {
	var apiError smithy.APIError
	errors.As(err, &apiError)
	switch apiError.(type) {
	case *types.NoSuchEntityException:
		return true
	default:
		return false
	}
}

type iamWrapper struct {
	IamClient *iam.Client
}

func NewIamWrapper() (*iamWrapper, error) {
	// TODO: take config/credentials in this constructor
	sdkConfig, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		return nil, err
	}

	iamWrapper := iamWrapper{
		IamClient: iam.NewFromConfig(sdkConfig),
	}
	return &iamWrapper, nil
}

func (wrapper iamWrapper) GetUser(ctx context.Context, userName string) (*types.User, error) {
	result, err := wrapper.IamClient.GetUser(ctx, &iam.GetUserInput{
		UserName: aws.String(userName),
	})
	if isNoSuchEntityException(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return result.User, err
}

func (wrapper iamWrapper) IsExistingUser(ctx context.Context, userName string) (bool, error) {
	_, err := wrapper.IamClient.GetUser(ctx, &iam.GetUserInput{
		UserName: aws.String(userName),
	})
	if isNoSuchEntityException(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (wrapper iamWrapper) HasLoginProfile(ctx context.Context, userName string) (bool, error) {
	_, err := wrapper.IamClient.GetLoginProfile(ctx, &iam.GetLoginProfileInput{
		UserName: aws.String(userName),
	})
	if isNoSuchEntityException(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (wrapper iamWrapper) HasAccessKey(ctx context.Context, userName string) (bool, error) {
	result, err := wrapper.IamClient.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return false, err
	}
	return len(result.AccessKeyMetadata) > 0, nil
}

func (wrapper iamWrapper) ListGroupsForUser(ctx context.Context, userName string) ([]types.Group, error) {
	result, err := wrapper.IamClient.ListGroupsForUser(ctx, &iam.ListGroupsForUserInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return nil, err
	}
	return result.Groups, nil
}

func (wrapper iamWrapper) CreateUser(ctx context.Context, userName string) (*types.User, error) {
	var user *types.User
	result, err := wrapper.IamClient.CreateUser(ctx, &iam.CreateUserInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		log.Printf("Couldn't create user %v. Here's why: %v\n", userName, err)
	} else {
		user = result.User
	}
	return user, err
}

func (wrapper iamWrapper) CreateUserIfNotExists(ctx context.Context, userName string) error {
	_, err := wrapper.IamClient.CreateUser(ctx, &iam.CreateUserInput{
		UserName: aws.String(userName),
	})
	if err != nil && !isNoSuchEntityException(err) {
		return err
	}
	return nil
}

func (wrapper iamWrapper) ListUsers(ctx context.Context, maxUsers int32) ([]types.User, error) {
	var users []types.User
	result, err := wrapper.IamClient.ListUsers(ctx, &iam.ListUsersInput{
		MaxItems: aws.Int32(maxUsers),
	})
	if err != nil {
		log.Printf("Couldn't list users. Here's why: %v\n", err)
	} else {
		users = result.Users
	}
	return users, err
}

func (wrapper iamWrapper) CreateLoginProfile(ctx context.Context, password string, userName string, passwordResetRequired bool) (types.LoginProfile, error) {
	result, err := wrapper.IamClient.CreateLoginProfile(ctx, &iam.CreateLoginProfileInput{
		Password:              &password,
		UserName:              &userName,
		PasswordResetRequired: passwordResetRequired,
	})
	var loginProfile types.LoginProfile
	if err != nil {
		log.Printf("Couldn't create login profile. Here's why: %v\n", err)
	} else {
		loginProfile = *result.LoginProfile
	}
	return loginProfile, err
}

func (wrapper iamWrapper) CreateLoginProfileIfNotExists(ctx context.Context, password string, userName string, passwordResetRequired bool) error {
	_, err := wrapper.IamClient.CreateLoginProfile(ctx, &iam.CreateLoginProfileInput{
		Password:              &password,
		UserName:              &userName,
		PasswordResetRequired: passwordResetRequired,
	})
	if err != nil && !isNoSuchEntityException(err) {
		return err
	}
	return nil
}

func (wrapper iamWrapper) CreateAccessKeyPair(ctx context.Context, userName string) (*types.AccessKey, error) {
	var key *types.AccessKey
	result, err := wrapper.IamClient.CreateAccessKey(ctx, &iam.CreateAccessKeyInput{
		UserName: aws.String(userName)})
	if err != nil {
		log.Printf("Couldn't create access key pair for user %v. Here's why: %v\n", userName, err)
	} else {
		key = result.AccessKey
	}
	return key, err
}

func (wrapper iamWrapper) AddUserToGroup(ctx context.Context, groupName string, userName string) (middleware.Metadata, error) {
	var metadata middleware.Metadata
	result, err := wrapper.IamClient.AddUserToGroup(ctx, &iam.AddUserToGroupInput{
		GroupName: aws.String(groupName),
		UserName:  aws.String(userName)})
	if err != nil {
		log.Printf("Couldn't add user %v to group. Here's why: %v\n", userName, err)
	} else {
		metadata = result.ResultMetadata
	}
	return metadata, err
}

func (wrapper iamWrapper) RemoveUserFromGroup(ctx context.Context, groupName string, userName string) (middleware.Metadata, error) {
	var metadata middleware.Metadata
	result, err := wrapper.IamClient.RemoveUserFromGroup(ctx, &iam.RemoveUserFromGroupInput{
		GroupName: aws.String(groupName),
		UserName:  aws.String(userName)})
	if err != nil {
		log.Printf("Couldn't remove user %v from group %v. Here's why: %v\n", userName, groupName, err)
	} else {
		metadata = result.ResultMetadata
	}
	return metadata, err
}

func (wrapper iamWrapper) DeleteUser(ctx context.Context, userName string) error {
	_, err := wrapper.IamClient.DeleteUser(ctx, &iam.DeleteUserInput{
		UserName: aws.String(userName),
	})
	return err
}

func (wrapper iamWrapper) DeleteLoginProfileIfExists(ctx context.Context, userName string) error {
	_, err := wrapper.IamClient.DeleteLoginProfile(ctx, &iam.DeleteLoginProfileInput{
		UserName: aws.String(userName),
	})
	if isNoSuchEntityException(err) {
		return nil
	}
	return err
}

func (wrapper iamWrapper) ListAccessKeys(ctx context.Context, userName string) ([]types.AccessKeyMetadata, error) {
	result, err := wrapper.IamClient.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
		UserName: aws.String(userName),
	})
	if err != nil {
		return nil, err
	}
	return result.AccessKeyMetadata, nil
}

func (wrapper iamWrapper) DeleteAccessKeyIfExists(ctx context.Context, userName string, keyId string) error {
	_, err := wrapper.IamClient.DeleteAccessKey(ctx, &iam.DeleteAccessKeyInput{
		AccessKeyId: aws.String(keyId),
		UserName:    aws.String(userName),
	})
	if isNoSuchEntityException(err) {
		return nil
	}
	return err
}
