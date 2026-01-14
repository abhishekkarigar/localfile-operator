/*
Copyright 2026.

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
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	log "sigs.k8s.io/controller-runtime/pkg/log"

	demov1 "github.com/abhishekkarigar/localfile-operator/api/v1"
)

const localFileFinalizer = "localfile.demo.io/finalizer"

// LocalFileReconciler reconciles a LocalFile object
type LocalFileReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=demo.demo.io,resources=localfiles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=demo.demo.io,resources=localfiles/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=demo.demo.io,resources=localfiles/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the LocalFile object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *LocalFileReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var lf demov1.LocalFile
	if err := r.Get(ctx, req.NamespacedName, &lf); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 🔥 FINALIZER LOGIC
	if lf.ObjectMeta.DeletionTimestamp.IsZero() {
		// Add finalizer if not present
		if !controllerutil.ContainsFinalizer(&lf, localFileFinalizer) {
			controllerutil.AddFinalizer(&lf, localFileFinalizer)
			if err := r.Update(ctx, &lf); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// Resource is being deleted
		if controllerutil.ContainsFinalizer(&lf, localFileFinalizer) {
			log.Info("Deleting local file", "path", lf.Spec.Path)

			_ = os.Remove(lf.Spec.Path)

			controllerutil.RemoveFinalizer(&lf, localFileFinalizer)
			if err := r.Update(ctx, &lf); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// 📝 CREATE / UPDATE FILE
	err := os.WriteFile(
		lf.Spec.Path,
		[]byte(lf.Spec.Content),
		0644,
	)
	if err != nil {
		log.Error(err, "Failed to write file")
		lf.Status.Created = false
		lf.Status.Message = err.Error()
	} else {
		lf.Status.Created = true
		lf.Status.Message = "File created successfully"
	}

	if err := r.Status().Update(ctx, &lf); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LocalFileReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1.LocalFile{}).
		Named("localfile").
		Complete(r)
}
