package controller

import (
	"context"
	"errors"
	"time"

	kuadrav1 "github.com/Kuadrant/kuadra/api/v1"
	slice "github.com/Kuadrant/kuadra/pkg/_internal"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go/middleware"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8Types "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("AwsAccount controller", func() {

	const (
		AwsAccountName      = "awsaccount-test"
		AwsAccountNamespace = "default"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	ctx := context.Background()

	awsController := &kuadrav1.AwsAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AwsAccount",
			APIVersion: "kuadra.kuadrant.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      AwsAccountName,
			Namespace: AwsAccountNamespace,
		},
		Spec: kuadrav1.AwsAccountSpec{
			UserName: "ib-dns",
			Groups: []string{
				"dns-management",
				"test-group",
			},
		},
	}

	awsAccountLookupKey := k8Types.NamespacedName{Name: AwsAccountName, Namespace: AwsAccountNamespace}
	createdAwsAccount := &kuadrav1.AwsAccount{
		Spec: kuadrav1.AwsAccountSpec{
			UserName: "ib-dns",
		},
		Status: kuadrav1.AwsAccountStatus{},
	}

	Context("When checking AwsController Status", func() {
		It("Should run AwsAccount reconcile", func() {
			By("By checking reconcile")
			req := reconcile.Request{
				NamespacedName: awsAccountLookupKey,
			}

			client := fake.NewClientBuilder().Build()

			mockIam := mockIamWrapper{
				Users:        []types.User{},
				LoginProfile: map[string]types.LoginProfile{},
				AccessKeys:   map[string][]types.AccessKey{},
				Groups:       map[string][]types.Group{},
			}

			client.Create(ctx, awsController)

			r := &AwsAccountReconciler{
				Client:     client,
				Scheme:     scheme.Scheme,
				IamWrapper: &mockIam,
			}

			_, err := r.Reconcile(ctx, req)
			if err != nil {
				Expect(err).Should(BeNil())
			}

			By("By checking if AwsAccount status is correct")
			Eventually(func() kuadrav1.AwsAccountStatus {
				err := client.Get(ctx, awsAccountLookupKey, createdAwsAccount)
				if err != nil {
					return createdAwsAccount.Status
				}
				return createdAwsAccount.Status
			}, timeout, interval).Should(Equal(kuadrav1.AwsAccountStatus{
				UserCreated:         true,
				LoginProfileCreated: true,
				AccessKeyCreated:    true,
				UserGroups:          awsController.Spec.Groups,
				NamespaceCreated:    true,
			}))

			By("By checking created user")
			Expect(mockIam.Users).Should(Equal([]types.User{
				{
					UserName: &awsController.Spec.UserName,
				},
			}))

			By("By checking if user has login profile")
			Expect(mockIam.LoginProfile[awsController.Spec.UserName]).Should(Equal(types.LoginProfile{
				UserName:              &awsController.Spec.UserName,
				PasswordResetRequired: true,
			}))

			By("By checking if user has access key")
			Expect(mockIam.AccessKeys[awsController.Spec.UserName]).Should(Equal([]types.AccessKey{
				{
					AccessKeyId:     aws.String("AccessKeyId"),
					SecretAccessKey: aws.String("SecretAccessKey"),
				},
			}))

			By("By checking if user has correct groups")
			Expect(mockIam.Groups[awsController.Spec.UserName]).Should(Equal([]types.Group{
				{
					GroupName: &awsController.Spec.Groups[0],
				},
				{
					GroupName: &awsController.Spec.Groups[1],
				},
			}))
		})
	})
})

type mockIamWrapper struct {
	Users        []types.User
	LoginProfile map[string]types.LoginProfile
	AccessKeys   map[string][]types.AccessKey
	Groups       map[string][]types.Group
}

func (c mockIamWrapper) GetUser(userName string) (*types.User, error) {
	for _, user := range c.Users {
		if *user.UserName == userName {
			return &user, nil
		}
	}
	return &types.User{}, errors.New("User does not exist")
}

func (c mockIamWrapper) IsExistingUser(ctx context.Context, userName string) (bool, error) {
	_, err := c.GetUser(userName)
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c mockIamWrapper) HasLoginProfile(ctx context.Context, userName string) (bool, error) {
	if _, exists := c.LoginProfile[userName]; !exists {
		return false, nil
	}
	return true, nil
}

func (c mockIamWrapper) HasAccessKey(ctx context.Context, userName string) (bool, error) {
	if _, exists := c.AccessKeys[userName]; !exists {
		return false, nil
	}
	return true, nil
}

func (c mockIamWrapper) ListGroupsForUser(ctx context.Context, userName string) ([]types.Group, error) {
	if _, exists := c.Groups[userName]; !exists {
		return nil, nil
	}
	return c.Groups[userName], nil
}

func (c *mockIamWrapper) CreateUser(ctx context.Context, userName string) (*types.User, error) {
	user := types.User{
		UserName: &userName,
	}
	c.Users = append(c.Users, user)
	return &user, nil
}

func (c *mockIamWrapper) CreateUserIfNotExists(ctx context.Context, userName string) error {
	c.CreateUser(ctx, userName)
	return nil
}

func (c mockIamWrapper) ListUsers(ctx context.Context, maxUsers int32) ([]types.User, error) {
	var users []types.User

	for i := int32(0); i < maxUsers && i < int32(len(c.Users)); i++ {
		users = append(users, c.Users[i])
	}
	return users, nil
}

func (c mockIamWrapper) CreateLoginProfile(ctx context.Context, password string, userName string, passwordResetRequired bool) (types.LoginProfile, error) {
	loginProfile := types.LoginProfile{
		UserName:              &userName,
		PasswordResetRequired: passwordResetRequired,
	}
	c.LoginProfile[userName] = loginProfile
	return c.LoginProfile[userName], nil
}

func (c mockIamWrapper) CreateLoginProfileIfNotExists(ctx context.Context, password string, userName string, passwordResetRequired bool) error {
	c.CreateLoginProfile(ctx, password, userName, passwordResetRequired)
	return nil
}

func (c mockIamWrapper) CreateAccessKeyPair(ctx context.Context, userName string) (*types.AccessKey, error) {
	accessKey := types.AccessKey{
		AccessKeyId:     aws.String("AccessKeyId"),
		SecretAccessKey: aws.String("SecretAccessKey"),
	}
	c.AccessKeys[userName] = append(c.AccessKeys[userName], accessKey)

	return &accessKey, nil
}

func (c mockIamWrapper) AddUserToGroup(ctx context.Context, groupName string, userName string) (middleware.Metadata, error) {
	userGroup := types.Group{
		GroupName: &groupName,
	}
	c.Groups[userName] = append(c.Groups[userName], userGroup)
	return middleware.Metadata{}, nil
}

func (c *mockIamWrapper) RemoveUserFromGroup(ctx context.Context, groupName string, userName string) (middleware.Metadata, error) {
	c.Groups[userName] = slice.Remove(c.Groups[userName], func(g types.Group) bool { return g == types.Group{GroupName: &groupName} })
	return middleware.Metadata{}, nil
}

func (c *mockIamWrapper) DeleteUser(ctx context.Context, userName string) error {
	c.Users = slice.Remove(c.Users, func(u types.User) bool { return u.UserName == &userName })
	return nil
}

func (c *mockIamWrapper) DeleteLoginProfileIfExists(ctx context.Context, userName string) error {
	delete(c.LoginProfile, userName)
	return nil
}

func (c *mockIamWrapper) ListAccessKeys(ctx context.Context, userName string) ([]types.AccessKeyMetadata, error) {
	var accessKeys []types.AccessKeyMetadata
	for _, accessKey := range c.AccessKeys[userName] {
		ak := types.AccessKeyMetadata{
			AccessKeyId: accessKey.AccessKeyId,
		}
		accessKeys = append(accessKeys, ak)
	}
	return accessKeys, nil
}

func (c *mockIamWrapper) DeleteAccessKeyIfExists(ctx context.Context, userName string, keyId string) error {
	c.AccessKeys[userName] = slice.Remove(c.AccessKeys[userName], func(a types.AccessKey) bool { return a.AccessKeyId == &keyId })
	return nil
}
