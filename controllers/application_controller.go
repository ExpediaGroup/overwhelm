// Copyright 2022 Expedia Group
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"bytes"
	"context"
	"errors"
	"github.com/ExpediaGroup/overwhelm/pkg/data"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"text/template"

	corev1alpha1 "github.com/ExpediaGroup/overwhelm/api/v1alpha1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	// name of our custom finalizer
	myFinalizerName := "overwhelm.expediagroup.com/finalizer"
	application := &corev1alpha1.Application{}
	if err := r.Get(ctx, req.NamespacedName, application); err != nil {
		log.Log.Error(err, "Error reading application object")
	}
	if application.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(application, myFinalizerName) {
			controllerutil.AddFinalizer(application, myFinalizerName)
			if err := r.Update(ctx, application); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.CreateOrUpdateResources(application, ctx); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(application, myFinalizerName) {
			//if err := r.deleteAssociatedResources(application, ctx); err != nil {
			//	return ctrl.Result{}, err
			//}
			controllerutil.RemoveFinalizer(application, myFinalizerName)
			if err := r.Update(ctx, application); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Application{}).
		Complete(r)
}

func (r *ApplicationReconciler) createOrUpdateConfigMap(application *corev1alpha1.Application, ctx context.Context) error {
	values, _ := r.renderValues(application.Spec.Data)

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   application.Namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Data: values,
	}
	if err := ctrl.SetControllerReference(application, cm, r.Scheme); err != nil {
		return err
	}
	currentCM := v1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, &currentCM); err != nil {
		if err := r.Create(ctx, cm); err != nil {
			log.Log.Error(errors.New("unable to update Configmap for Application"), "")
			return err
		}
	} else {
		if err := r.Update(ctx, cm); err != nil {
			log.Log.Error(errors.New("unable to create Configmap for Application"), "")
			return err
		}
	}
	return nil
}

func (r *ApplicationReconciler) renderValues(values map[string]string) (map[string]string, error) {

	for key, value := range values {
		buf := new(bytes.Buffer)
		tmpl, _ := template.New("test").Parse(value)
		_ = tmpl.Execute(buf, data.GetPreRenderReference())
		values[key] = buf.String()
	}
	return values, nil
}

//func (r *ApplicationReconciler) deleteAssociatedResources(application *corev1alpha1.Application, ctx context.Context) error {
//
//}

func (r *ApplicationReconciler) CreateOrUpdateResources(application *corev1alpha1.Application, ctx context.Context) error {
	return r.createOrUpdateConfigMap(application, ctx)

}
