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

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kuadrav1 "github.com/Kuadrant/kuadra/api/v1"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=users/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	user := kuadrav1.User{}
	if err := r.Get(ctx, req.NamespacedName, &user); err != nil {
		log.Error(err, "unable to fetch User")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	awsAccount := r.createAwsAccountScheme(&user, req.Namespace)

	if err := controllerutil.SetControllerReference(&user, awsAccount, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference for AwsAccount")
		return reconcile.Result{}, err
	}

	existingAwsAccount, err := r.getExistingAwsAccount(ctx, awsAccount)
	if err != nil {
		return reconcile.Result{}, err
	}
	if existingAwsAccount == nil {
		err := r.createAwsAccount(ctx, awsAccount)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else {
		err := r.updateAwsAccount(ctx, awsAccount, existingAwsAccount)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *UserReconciler) createAwsAccountScheme(user *kuadrav1.User, namespace string) *kuadrav1.AwsAccount {
	return &kuadrav1.AwsAccount{
		TypeMeta: v1.TypeMeta{},
		ObjectMeta: v1.ObjectMeta{
			Name:      user.Spec.AwsAccount.Spec.User.UserName,
			Namespace: namespace,
		},
		Spec: kuadrav1.AwsAccountSpec{
			UserName: user.Spec.AwsAccount.Spec.User.UserName,
			Groups:   user.Spec.AwsAccount.Spec.User.Groups,
		},
	}
}

// getExistingAwsAccount retrieves the existing AwsAccount object, if it exists.
func (r *UserReconciler) getExistingAwsAccount(ctx context.Context, awsAccount *kuadrav1.AwsAccount) (*kuadrav1.AwsAccount, error) {
	existingAwsAccount := &kuadrav1.AwsAccount{}
	err := r.Get(ctx, types.NamespacedName{Name: awsAccount.Spec.UserName, Namespace: awsAccount.Namespace}, existingAwsAccount)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return existingAwsAccount, nil
}

// updateAwsAccount updates the existing AwsAccount object.
func (r *UserReconciler) updateAwsAccount(ctx context.Context, awsAccount, existingAwsAccount *kuadrav1.AwsAccount) error {
	awsAccount.ObjectMeta.ResourceVersion = existingAwsAccount.ObjectMeta.ResourceVersion
	err := r.Update(ctx, awsAccount)
	if err != nil {
		log.Log.Error(err, "Failed to update AwsAccount")
		return err
	}
	return nil
}

// createAwsAccount creates a new AwsAccount object.
func (r *UserReconciler) createAwsAccount(ctx context.Context, awsAccount *kuadrav1.AwsAccount) error {
	err := r.Create(ctx, awsAccount)
	if err != nil && client.IgnoreAlreadyExists(err) != nil {
		log.Log.Error(err, "Failed to create AwsAccount")
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuadrav1.User{}).
		Complete(r)
}
