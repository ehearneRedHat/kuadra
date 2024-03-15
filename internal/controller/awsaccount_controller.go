/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"reflect"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/sethvargo/go-password/password"

	kuadrav1 "github.com/Kuadrant/kuadra/api/v1"
	slice "github.com/Kuadrant/kuadra/pkg/_internal"
)

const (
	AwsAccountFinalizer = "kuadra.kuadrant.io/aws-account"
)

// AwsAccountReconciler reconciles a AwsAccount object
type AwsAccountReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	IamWrapper IamWrapper
}

//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=awsaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=awsaccounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=awsaccounts/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AwsAccount object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *AwsAccountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var awsAccount kuadrav1.AwsAccount
	if err := r.Get(ctx, req.NamespacedName, &awsAccount); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if awsAccount.DeletionTimestamp != nil && !awsAccount.DeletionTimestamp.IsZero() {
		if err := r.deleteNamespace(ctx, awsAccount.Spec.UserName); err != nil {
			log.Error(err, "Failed to delete namespace", "namespace", awsAccount.Spec.UserName)
			return ctrl.Result{}, err
		}
		if err := r.deleteIamUser(ctx, awsAccount.Spec.UserName); err != nil {
			log.Error(err, "Failed to delete IAM user", "userName", awsAccount.Spec.UserName)
			return ctrl.Result{}, err
		}
		controllerutil.RemoveFinalizer(&awsAccount, AwsAccountFinalizer)

		if err := r.Update(ctx, &awsAccount); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&awsAccount, AwsAccountFinalizer) {
		controllerutil.AddFinalizer(&awsAccount, AwsAccountFinalizer)
		if err := r.Update(ctx, &awsAccount); err != nil {
			return ctrl.Result{}, err
		}
	}

	refreshedStatus, err := r.getRefreshedStatus(ctx, awsAccount)
	if err != nil {
		log.Error(err, "unable to get refreshed status")
		return ctrl.Result{}, err
	}
	awsAccount.Status = *refreshedStatus

	if !awsAccount.Status.NamespaceCreated {
		if err := r.createNamespaceIfNotExists(ctx, awsAccount.Spec.UserName); err != nil {
			log.Error(err, "unable to create namespace")
			return ctrl.Result{}, err
		}
		log.V(1).Info("created namespace", "namespace", awsAccount.Spec.UserName)
		awsAccount.Status.NamespaceCreated = true
	}

	if !awsAccount.Status.UserCreated {
		if err := r.IamWrapper.CreateUserIfNotExists(ctx, awsAccount.Spec.UserName); err != nil {
			log.Error(err, "unable to create IAM user")
			return ctrl.Result{}, err
		}
		log.V(1).Info("created user", "userName", awsAccount.Spec.UserName)
		awsAccount.Status.UserCreated = true
	}

	if !awsAccount.Status.LoginProfileCreated {
		pass, err := password.Generate(20, 3, 3, false, true)
		if err != nil {
			log.Error(err, "unable to generate password")
			return ctrl.Result{}, err
		}
		secretData := map[string]string{
			"userName": awsAccount.Spec.UserName,
			"password": pass,
		}
		if err := r.createSecretIfNotExists(ctx, secretData, "aws-login", awsAccount.Spec.UserName); err != nil {
			log.Error(err, "unable to create secret for AWS password")
			return ctrl.Result{}, err
		}
		// Use password value from retrieved secret so that possible creation errors do not cause incorrect password to be set
		retrievedSecret := &v1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: "aws-login", Namespace: awsAccount.Spec.UserName}, retrievedSecret); err != nil {
			log.Error(err, "unable to get secret for AWS password")
			return ctrl.Result{}, err
		}
		if err := r.IamWrapper.CreateLoginProfileIfNotExists(ctx, string(retrievedSecret.Data["password"]), awsAccount.Spec.UserName, true); err != nil {
			log.Error(err, "unable to create login profile")
			return ctrl.Result{}, err
		}
		log.V(1).Info("created login profile")
		awsAccount.Status.LoginProfileCreated = true
	}

	if !awsAccount.Status.AccessKeyCreated {
		accessKey, err := r.IamWrapper.CreateAccessKeyPair(ctx, awsAccount.Spec.UserName)
		if err != nil {
			log.Error(err, "unable to create access key")
			return ctrl.Result{}, err
		}
		secretData := map[string]string{
			"AWS_ACCESS_KEY_ID":     *accessKey.AccessKeyId,
			"AWS_SECRET_ACCESS_KEY": *accessKey.SecretAccessKey,
		}
		if err := r.createSecretIfNotExists(ctx, secretData, "aws-credentials", awsAccount.Spec.UserName); err != nil {
			log.Error(err, "unable to create secret for AWS credentials")
			return ctrl.Result{}, err
		}
		log.V(1).Info("created access key", "accessKeyId", accessKey.AccessKeyId)
		awsAccount.Status.AccessKeyCreated = true
	}

	groupsToAddUserTo := slice.GetLeftDifference(awsAccount.Spec.Groups, awsAccount.Status.UserGroups)
	for _, group := range groupsToAddUserTo {
		if _, err := r.IamWrapper.AddUserToGroup(ctx, group, awsAccount.Spec.UserName); err != nil {
			log.Error(err, "unable to add user to group", "groupName", group)
			return ctrl.Result{}, err
		}
		log.V(1).Info("Added user to group", "group name:", group)
		awsAccount.Status.UserGroups = append(awsAccount.Status.UserGroups, group)
	}

	groupsToRemoveUserFrom := slice.GetLeftDifference(awsAccount.Status.UserGroups, awsAccount.Spec.Groups)
	for _, group := range groupsToRemoveUserFrom {
		if _, err := r.IamWrapper.RemoveUserFromGroup(ctx, group, awsAccount.Spec.UserName); err != nil {
			log.Error(err, "unable to remove user from group", "groupName", group)
			return ctrl.Result{}, err
		}
		log.V(1).Info("removed user from group", "groupName", group)
		awsAccount.Status.UserGroups = slice.Remove(awsAccount.Status.UserGroups, func(g string) bool { return g == group })
	}

	var latest kuadrav1.AwsAccount
	if err := r.Get(ctx, req.NamespacedName, &latest); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !reflect.DeepEqual(latest.Status, awsAccount.Status) {
		if err := r.Status().Update(ctx, &awsAccount); err != nil {
			log.Error(err, "unable to update awsAccount status")
			return ctrl.Result{RequeueAfter: time.Second * 3}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AwsAccountReconciler) isNamespace(ctx context.Context, namespace string) (bool, error) {
	ns := &v1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: namespace, Namespace: v1.NamespaceAll}, ns); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

func (r *AwsAccountReconciler) createNamespaceIfNotExists(ctx context.Context, namespace string) error {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	err := r.Create(ctx, ns)
	return client.IgnoreAlreadyExists(err)
}

func (r *AwsAccountReconciler) createSecretIfNotExists(ctx context.Context, data map[string]string, name string, namespace string) error {
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		StringData: data,
	}
	err := r.Create(ctx, secret)
	return client.IgnoreAlreadyExists(err)
}

func (r *AwsAccountReconciler) getRefreshedStatus(ctx context.Context, awsAccount kuadrav1.AwsAccount) (*kuadrav1.AwsAccountStatus, error) {
	var status kuadrav1.AwsAccountStatus

	namespaceExists, err := r.isNamespace(ctx, awsAccount.Spec.UserName)
	if err != nil {
		return nil, err
	}
	status.NamespaceCreated = namespaceExists

	userExists, err := r.IamWrapper.IsExistingUser(ctx, awsAccount.Spec.UserName)
	if err != nil {
		return nil, err
	}
	if !userExists {
		// Return struct with zero values
		return &status, nil
	}
	status.UserCreated = true

	loginProfileExists, err := r.IamWrapper.HasLoginProfile(ctx, awsAccount.Spec.UserName)
	if err != nil {
		return nil, err
	}
	status.LoginProfileCreated = loginProfileExists

	accessKeyExists, err := r.IamWrapper.HasAccessKey(ctx, awsAccount.Spec.UserName)
	if err != nil {
		return nil, err
	}
	status.AccessKeyCreated = accessKeyExists

	groups, err := r.IamWrapper.ListGroupsForUser(ctx, awsAccount.Spec.UserName)
	if err != nil {
		return nil, err
	}
	for _, group := range groups {
		status.UserGroups = append(status.UserGroups, *group.GroupName)
	}

	return &status, nil
}

func (r *AwsAccountReconciler) deleteNamespace(ctx context.Context, namespace string) error {
	ns := &v1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: namespace, Namespace: v1.NamespaceAll}, ns); err != nil {
		return client.IgnoreNotFound(err)
	}
	return r.Delete(ctx, ns)
}

func (r *AwsAccountReconciler) deleteIamUser(ctx context.Context, userName string) error {
	userExists, err := r.IamWrapper.IsExistingUser(ctx, userName)
	if err != nil {
		return err
	}
	if !userExists {
		return nil
	}

	groups, err := r.IamWrapper.ListGroupsForUser(ctx, userName)
	if err != nil {
		return err
	}
	for _, group := range groups {
		if _, err := r.IamWrapper.RemoveUserFromGroup(ctx, *group.GroupName, userName); err != nil {
			return err
		}
	}

	if err := r.IamWrapper.DeleteLoginProfileIfExists(ctx, userName); err != nil {
		return err
	}

	accessKeys, err := r.IamWrapper.ListAccessKeys(ctx, userName)
	if err != nil {
		return err
	}
	for _, accessKey := range accessKeys {
		if err := r.IamWrapper.DeleteAccessKeyIfExists(ctx, userName, *accessKey.AccessKeyId); err != nil {
			return err
		}
	}

	return r.IamWrapper.DeleteUser(ctx, userName)
}

// SetupWithManager sets up the controller with the Manager.
func (r *AwsAccountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuadrav1.AwsAccount{}).
		Complete(r)
}
