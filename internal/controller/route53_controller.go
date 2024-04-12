/*
Copyright 2024.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	kuadrav1 "github.com/Kuadrant/kuadra/api/v1"
)

// Route53Reconciler reconciles a Route53 object
type Route53Reconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Route53Wrapper Route53Wrapper
}

// Create finalizer for deletion purposes
const Route53Finalizer = "kuadra.kuadrant.io/route53"

//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=route53s,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=route53s/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=kuadra.kuadrant.io,resources=route53s/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Route53 object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *Route53Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var route53 kuadrav1.Route53

	// Pull in the CR sample to perform CRUD actions against

	if err := r.Get(ctx, req.NamespacedName, &route53); err != nil {
		log.Error(err, "error getting the route53 cr", "route53", route53)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Delete Hosted Zone
	if route53.DeletionTimestamp != nil && !route53.DeletionTimestamp.IsZero() {
		var err error

		// if root domain was specified in the CR sample...
		if route53.Spec.RootDomainName != "" {
			if err = r.Route53Wrapper.DeleteNameserverRecordFromHostedZone(ctx, route53.Spec.RootDomainName, route53.Spec.DomainName); err != nil {
				log.Error(err, "failed to delete nameserver record", "nameserverRecord", route53.Spec.DomainName)
			}
			log.Info("deleted nameserver record from root hosted zone", "rootDomainName", route53.Spec.RootDomainName)
		}

		if err = r.Route53Wrapper.DeleteHostedZone(ctx, route53.Spec.DomainName); err != nil {
			log.Error(err, "failed to delete hosted zone", "hostedZone", route53.Spec.DomainName)
			return ctrl.Result{}, err
		}
		log.Info("deleted hosted zone", "domainName", route53.Spec.DomainName)
		// No need for finalizer since deleted
		controllerutil.RemoveFinalizer(&route53, Route53Finalizer)
		if err := r.Update(ctx, &route53); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for deletion
	if !controllerutil.ContainsFinalizer(&route53, Route53Finalizer) {
		controllerutil.AddFinalizer(&route53, Route53Finalizer)
		if err := r.Update(ctx, &route53); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create Hosted Zone
	if !route53.Status.HostedZoneCreated {
		var err error
		// if root domain was specified in the CR sample...
		if route53.Spec.RootDomainName != "" {
			if err = r.Route53Wrapper.CreateHostedZoneRootDomain(ctx, route53.Spec.DomainName, route53.Spec.RootDomainName, route53.Spec.IsPrivateHostedZone); err != nil {
				log.Error(err, "unable to create hosted zone")
				return ctrl.Result{}, err
			}
		} else { // if root domain was not specified in the CR sample...
			if err = r.Route53Wrapper.CreateHostedZone(ctx, route53.Spec.DomainName, route53.Spec.IsPrivateHostedZone); err != nil {
				log.Error(err, "unable to create hosted zone")
				return ctrl.Result{}, err
			}
		}
		// Log the creation of the Hosted Zone
		log.Info("created hosted zone", "domainName", route53.Spec.DomainName)
		// Mark the hosted zone as created in the spec
		route53.Status.HostedZoneCreated = true
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Route53Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kuadrav1.Route53{}).
		Complete(r)
}
