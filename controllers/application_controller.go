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
	"regexp"
	"text/template"
	"time"

	"github.com/ExpediaGroup/overwhelm/data/reference"
	helmControllerV1 "github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/ExpediaGroup/overwhelm/api/v1alpha1"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	FinalizerName = "overwhelm.expediagroup.com/finalizer"
	ManagedBy     = "app.kubernetes.io/managed-by"
)

var log logr.Logger

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
	log = ctrllog.FromContext(ctx)
	// name of our custom finalizer
	log.Info("Starting to read Application")
	application := &corev1alpha1.Application{}
	if err := r.Get(ctx, req.NamespacedName, application); err != nil {
		log.Error(err, "Error reading application object")
	}
	if application.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(application, FinalizerName) {
			controllerutil.AddFinalizer(application, FinalizerName)
			if err := r.Update(ctx, application); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.CreateOrUpdateResources(application, ctx); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(application, FinalizerName) {
			//Add any pre delete actions here.
			controllerutil.RemoveFinalizer(application, FinalizerName)
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
		Owns(&v1.ConfigMap{}).
		Complete(r)
}

func (r *ApplicationReconciler) createOrUpdateConfigMap(application *corev1alpha1.Application, ctx context.Context) error {

	if err := r.renderValues(application); err != nil {
		log.Error(err, "error rendering values", "values", application.Spec.Data)
		return err
	}

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   application.Namespace,
			Labels:      application.Labels,
			Annotations: application.Annotations,
		},
		Data: application.Spec.Data,
	}
	if cm.Labels == nil {
		cm.Labels = make(map[string]string)
	}
	cm.Labels[ManagedBy] = "overwhelm"
	if err := ctrl.SetControllerReference(application, cm, r.Scheme); err != nil {
		return err
	}
	currentCM := v1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, &currentCM); err != nil {
		if err := r.Create(ctx, cm); err != nil {
			log.Error(err, "error creating the configmap", "cm", cm)
			return err
		}
	} else {
		log.Info("updating configmap")
		if err := r.Update(ctx, cm); err != nil {
			log.Error(err, "error updating the configmap", "cm", cm)
			return err
		}
	}
	return nil
}

func (r *ApplicationReconciler) createOrUpdateHelmRelease(application *corev1alpha1.Application, ctx context.Context) error {
	configMapName := application.Name
	//additionalValues := map[string]string{"releaseId": release.ReleaseId}
	//Adding release Id (as a random ID) to the values field in the template below as a trigger to restart the
	//helm release every time theres an update even if there are no other changes in the HR template.
	//Doing this will immediately trigger Helm Controller into action, rather than waiting for the normal reconciliation interval.
	//And the reason we are doing that is to improve the user experience (not force them to wait for up to 10 minutes or even 3 minutes to get an updated status).
	//values, _ := json.Marshal(&additionalValues)
	//rawValues := v1.JSON{Raw: values}

	helmChartTemplate := &helmControllerV1.HelmChartTemplate{
		Spec: helmControllerV1.HelmChartTemplateSpec{
			Chart:   application.Name,
			Version: application.APIVersion,
			SourceRef: helmControllerV1.CrossNamespaceObjectReference{
				Kind:      "HelmRepository",
				Name:      "public-helm-virtual",
				Namespace: "runtime",
			},
		},
	}

	hr := &helmControllerV1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   application.Namespace,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Spec: helmControllerV1.HelmReleaseSpec{
			Interval: metav1.Duration{
				Duration: 10 * time.Minute,
			},
			Chart:           *helmChartTemplate,
			ReleaseName:     application.Name,
			TargetNamespace: application.Namespace,
			ValuesFrom: []helmControllerV1.ValuesReference{
				{
					Kind:      "ConfigMap",
					Name:      configMapName,
					ValuesKey: "values.yaml",
					Optional:  false,
				},
				{
					Kind:      "ConfigMap",
					Name:      configMapName,
					ValuesKey: "global.yaml",
					Optional:  false,
				},
				{
					Kind:      "ConfigMap",
					Name:      configMapName,
					ValuesKey: "immutable.yaml",
					Optional:  false,
				},
			},
			//Values: &rawValues,
			Upgrade: &helmControllerV1.Upgrade{
				Remediation: &helmControllerV1.UpgradeRemediation{
					Retries: 3,
				},
			},
		},
	}
	if err := ctrl.SetControllerReference(application, hr, r.Scheme); err != nil {
		return err
	}
	currentHR := v1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, &currentHR); err != nil {
		if err := r.Create(ctx, hr); err != nil {
			log.Error(errors.New("unable to update HelmRelease for Application"), "")
			return err
		}
	} else {
		if err := r.Update(ctx, hr); err != nil {
			log.Error(errors.New("unable to create HelmRelease for Application"), "")
			return err
		}
	}
	return nil
}

func (r *ApplicationReconciler) renderValues(application *corev1alpha1.Application) error {
	values := application.Spec.Data
	leftDelimiter := "{{"
	rightDelimiter := "}}"
	preRenderer := &application.Spec.PreRenderer
	if preRenderer != nil {

		// when no pre-rendering is desired. Only Helm Templating
		if preRenderer.EnableHelmTemplating && preRenderer.LeftDelimiter == "" && preRenderer.RightDelimiter == "" {
			return nil
		}

		// when delimiters are specified but only partially
		if preRenderer.LeftDelimiter == "" || preRenderer.RightDelimiter == "" {
			if preRenderer.LeftDelimiter != "" || preRenderer.RightDelimiter != "" {
				return errors.New("application preRenderer has partial delimiter information")
			}
		}
		if preRenderer.LeftDelimiter != "" && preRenderer.RightDelimiter != "" {
			leftDelimiter = preRenderer.LeftDelimiter
			rightDelimiter = preRenderer.RightDelimiter
			if !isDelimValid(leftDelimiter) || !isDelimValid(rightDelimiter) {
				return errors.New("application preRenderer has invalid delimiters. Make sure it has two characters, is non alpha numeric and with no spaces")
			}
		}
	}

	for key, value := range values {
		buf := new(bytes.Buffer)
		tmpl, err := template.New("properties").Option("missingkey=error").Delims(leftDelimiter, rightDelimiter).Parse(value)
		if err != nil {
			return err
		}
		if err = tmpl.Execute(buf, reference.GetPreRenderData()); err != nil {
			return err
		}
		values[key] = buf.String()
	}
	return nil
}
func isDelimValid(delim string) bool {
	r := regexp.MustCompile(`^.*([a-zA-Z0-9 ])+.*$`)
	return len(delim) == 2 && !r.MatchString(delim)
}

func (r *ApplicationReconciler) CreateOrUpdateResources(application *corev1alpha1.Application, ctx context.Context) error {
	r.createOrUpdateConfigMap(application, ctx)
	r.createOrUpdateHelmRelease(application, ctx)
	return nil
}
