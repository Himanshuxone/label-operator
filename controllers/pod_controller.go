/*
Copyright 2021.

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

package controllers

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	addPodNameLabelAnnotation = "custom/add-pod-name-label"
	podNameLabel              = "custom/pod-name"
)

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// Reconcile handles a reconciliation request for a Pod.
// If the Pod has the addPodNameLabelAnnotation annotation, then Reconcile
// will make sure the podNameLabel label is present with the correct value.
// If the annotation is absent, then Reconcile will make sure the label is too.
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Step 0: Fetch the Pod from the Kubernetes API.
	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			// we'll ignore not-found errors, since we can get them on deleted requests.
			return ctrl.Result{}, nil
		}
		log.Log.Error(err, "unable to fetch Pod")
		return ctrl.Result{}, err
	}

	// Step 1: Add or remove the label.
	labelShouldBePresent := pod.Annotations[addPodNameLabelAnnotation] == "true"
	labelIsPresent := pod.Labels[podNameLabel] == pod.Name
	if labelShouldBePresent == labelIsPresent {
		// The desired state and actual state of the Pod are the same.
		// No further action is required by the operator at this moment.
		log.Log.Info("no update required")
		return ctrl.Result{}, nil
	}
	if labelShouldBePresent {
		// If the label should be set but is not, set it.
		if pod.Labels == nil {
			pod.Labels = make(map[string]string)
		}
		pod.Labels[podNameLabel] = pod.Name
		log.Log.Info("adding label")
	} else {
		// If the label should not be set but is, remove it.
		delete(pod.Labels, podNameLabel)
		log.Log.Info("removing label")
	}

	// Step 2: Update the Pod in the Kubernetes API.
	if err := r.Update(ctx, &pod); err != nil {
		if apierrors.IsConflict(err) {
			// The Pod has been updated since we read it.
			// Requeue the Pod to try to reconciliate again.
			return ctrl.Result{Requeue: true}, nil
		}
		if apierrors.IsNotFound(err) {
			// The Pod has been deleted since we read it.
			// Requeue the Pod to try to reconciliate again.
			return ctrl.Result{Requeue: true}, nil
		}
		log.Log.Error(err, "unable to update Pod")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}
