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
	"crypto/sha1"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/ExpediaGroup/overwhelm/analyzer"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1 "github.com/ExpediaGroup/overwhelm/api/v1alpha1"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
	Retries         int64
	Events          record.EventRecorder
}

const (
	FinalizerName             = "overwhelm.expediagroup.com/finalizer"
	ValuesChecksumName        = "overwhelm.expediagroup.com/values-checksum"
	ManagedBy                 = "app.kubernetes.io/managed-by"
	LabelHelmReleaseName      = "helm.toolkit.fluxcd.io/name"
	LabelHelmReleaseNamespace = "helm.toolkit.fluxcd.io/namespace"
	InfoReason                = "Info"
	ErrorReason               = "Error"
)

var log logr.Logger

//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;watch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get
//+kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get

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
	application := &v1.Application{}
	pod := &corev1.Pod{}
	var err error
	if err = r.Get(ctx, req.NamespacedName, application); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Error reading application object")
			return ctrl.Result{}, err
		}
		// if application is not found the request could be from a pod which has a different name
		if err = r.getApplicationFromPod(req, pod, application); err != nil {
			return ctrl.Result{}, nil
		}

		err = r.reconcilePodStatus(application, pod)
		if patchErr := r.patchStatus(ctx, application); patchErr != nil {
			log.Error(patchErr, "Error updating application status")
			return ctrl.Result{}, patchErr
		}
		if err != nil {
			log.Error(err, "Error reconciling pod status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil

	}
	if application.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(application, FinalizerName) {
			patch := client.MergeFrom(application.DeepCopy())
			controllerutil.AddFinalizer(application, FinalizerName)
			if patchErr := r.Patch(ctx, application, patch); patchErr != nil {
				log.Error(patchErr, "Error adding a finalizer")
				return ctrl.Result{}, patchErr
			}
		}
		// New Application and Application update are identified by gen and observed gen mismatch
		if application.Status.ObservedGeneration != application.Generation {
			application.Status.ObservedGeneration = application.Generation
			application.Status.HelmReleaseGeneration = 0
			application.Status.ValuesCheckSum = ""
			application.Status.Conditions = nil
			v1.AppInProgressStatus(application)
			r.Events.Eventf(application, corev1.EventTypeNormal, InfoReason, "Creating ConfigMap and HelmRelease Objects")
			if err = r.patchStatus(ctx, application); err != nil {
				return ctrl.Result{RequeueAfter: r.RequeueInterval}, err
			}
		}
		// Create or Update resources (configmap and HR) if not already installed
		if err = r.CreateOrUpdateResources(ctx, application); err != nil {
			v1.AppErrorStatus(application, err.Error())
			if patchErr := r.patchStatus(ctx, application); patchErr != nil {
				return ctrl.Result{RequeueAfter: r.RequeueInterval}, patchErr
			}
			return ctrl.Result{}, err
		} else { // need to update the hr and cm generation in the application status if create or update succeeds
			if patchErr := r.patchStatus(ctx, application); patchErr != nil {
				return ctrl.Result{RequeueAfter: r.RequeueInterval}, patchErr
			}
		}
		// At this point the Helm Release can be reconciled
		err := r.reconcileHelmReleaseStatus(ctx, application)
		if patchErr := r.patchStatus(ctx, application); patchErr != nil {
			log.Error(patchErr, "Error updating application status")
			return ctrl.Result{}, patchErr
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if controllerutil.ContainsFinalizer(application, FinalizerName) {
			helmRelease := &v2beta1.HelmRelease{}
			if err = r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, helmRelease); err != nil {
				if !apierrors.IsNotFound(err) {
					err = fmt.Errorf("failed to get HelmRelease '%s': %w", req.Name, err)
					return ctrl.Result{}, err
				}
			} else {
				if err = r.Client.Delete(ctx, helmRelease); err != nil {
					err = fmt.Errorf("failed to delete HelmRelease '%s': %w", req.Name, err)
					return ctrl.Result{}, err
				}
			}
			patch := client.MergeFrom(application.DeepCopy())
			controllerutil.RemoveFinalizer(application, FinalizerName)
			if err = r.Patch(ctx, application, patch); err != nil {
				return ctrl.Result{}, err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) patchStatus(ctx context.Context, application *v1.Application) error {
	key := client.ObjectKeyFromObject(application)
	latest := &v1.Application{}
	if err := r.Get(ctx, key, latest); err != nil {
		return err
	}
	return r.Status().Patch(ctx, application, client.MergeFrom(latest))
}

func (r *ApplicationReconciler) reconcileHelmReleaseStatus(ctx context.Context, application *v1.Application) error {
	hr := v2beta1.HelmRelease{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, &hr)
	if err == nil && hr.Generation != application.Status.HelmReleaseGeneration {
		err = errors.New("HelmRelease not updated")
	}

	if err != nil {
		if application.Status.Failures == r.Retries {
			if apierrors.IsNotFound(err) {
				v1.AppErrorStatus(application, "Helm Release not created")
			} else {
				v1.AppErrorStatus(application, err.Error())
			}
			return nil
		}
		application.Status.Failures++
		return err
	}
	application.Status.Failures = 0
	if hr.Status.ObservedGeneration != hr.Generation {
		v1.AppErrorStatus(application, "updated Helm Release status not available")
		return errors.New("HelmRelease status is not current")
	}
	for _, condition := range hr.GetConditions() {
		apimeta.SetStatusCondition(&application.Status.Conditions, condition)
	}
	return nil
}

func (r *ApplicationReconciler) reconcilePodStatus(application *v1.Application, pod *corev1.Pod) error {
	log.Info("Analyzing latest pod status", "podName", pod.GetName(), "application", application.Name, "namespace", application.Namespace)
	result := analyzer.Pod(pod)
	v1.AppPodAnalysisCondition(application, result)
	return nil
}

func extractReplicaSetNameFromDeployment(deployment *appsv1.Deployment) string {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing {
			words := strings.Fields(condition.Message)
			if len(words) < 2 {
				return ""
			}
			return strings.Trim(words[1], `"`)
		}
	}
	return ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Application{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&v2beta1.HelmRelease{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}).
		Complete(r)
}

func (r *ApplicationReconciler) createOrUpdateConfigMap(ctx context.Context, application *v1.Application) error {
	currentCM := corev1.ConfigMap{}
	currentCMError := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, &currentCM)

	if currentCMError == nil && application.Status.ValuesCheckSum != "" && currentCM.Annotations[ValuesChecksumName] != "" && currentCM.Annotations[ValuesChecksumName] == application.Status.ValuesCheckSum {
		return nil
	}

	newCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   application.Namespace,
			Labels:      application.Spec.Template.Labels,
			Annotations: application.Spec.Template.Annotations,
		},
		Data: application.Spec.Data,
	}
	if newCM.Labels == nil {
		newCM.Labels = make(map[string]string)
	}
	if newCM.Annotations == nil {
		newCM.Annotations = make(map[string]string)
	}
	newCM.Labels[ManagedBy] = "overwhelm"
	if err := ctrl.SetControllerReference(application, newCM, r.Scheme); err != nil {
		return err
	}

	if err := r.renderValues(application); err != nil {
		log.Error(err, "error rendering values", "values", application.Spec.Data)
		r.Events.Eventf(application, corev1.EventTypeWarning, ErrorReason, err.Error())
		return err
	}
	data, err := yaml.Marshal(application.Spec.Data)
	if err == nil {
		checksum := fmt.Sprintf("%x", sha1.Sum(data))
		application.Status.ValuesCheckSum = checksum
		newCM.Annotations[ValuesChecksumName] = checksum
	} // need to fix the checksum calculation and check for zero values checksum. Also then need to check pod status
	if currentCMError != nil {
		if apierrors.IsNotFound(currentCMError) {
			if err := r.Create(ctx, newCM); err != nil {
				log.Error(err, "error creating the configmap", "configMap", newCM)
				r.Events.Eventf(application, corev1.EventTypeWarning, ErrorReason, err.Error())

				return err
			}
			return nil
		}
		log.Error(currentCMError, "error retrieving current configmap if exists", "configMap", newCM)
		r.Events.Eventf(application, corev1.EventTypeWarning, ErrorReason, currentCMError.Error())
		return currentCMError
	}

	if err := r.Update(ctx, newCM); err != nil {
		log.Error(err, "error updating the configmap", "configMap", newCM)
		r.Events.Eventf(application, corev1.EventTypeWarning, ErrorReason, err.Error())
		return err
	}
	return nil
}

func (r *ApplicationReconciler) renderValues(application *v1.Application) error {
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
			// when custom delimiters are specified but only partially
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
		if err = tmpl.Execute(buf, GetPreRenderData()); err != nil {
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

func (r *ApplicationReconciler) CreateOrUpdateResources(ctx context.Context, application *v1.Application) error {
	if err := r.createOrUpdateConfigMap(ctx, application); err != nil {
		return err
	}
	return r.createOrUpdateHelmRelease(ctx, application)
}

func (r *ApplicationReconciler) createOrUpdateHelmRelease(ctx context.Context, application *v1.Application) error {
	newHR := &v2beta1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:        application.Name,
			Namespace:   application.Namespace,
			Labels:      application.Spec.Template.Labels,
			Annotations: application.Spec.Template.Annotations,
		},
		Spec: application.Spec.Template.Spec,
	}

	if err := ctrl.SetControllerReference(application, newHR, r.Scheme); err != nil {
		return err
	}
	currentHR := &v2beta1.HelmRelease{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, currentHR); err != nil {
		if apierrors.IsNotFound(err) {
			if err = r.Create(ctx, newHR); err != nil {
				log.Error(err, "error creating the HelmRelease Resource")
				r.Events.Eventf(application, corev1.EventTypeWarning, ErrorReason, err.Error())
				return err
			}
			application.Status.HelmReleaseGeneration = newHR.Generation
			return nil
		}
		log.Error(err, "error checking if current HelmRelease exists")
		return err
	}
	if currentHR.Generation != application.Status.HelmReleaseGeneration {
		newHR.ResourceVersion = currentHR.ResourceVersion

		if err := r.Update(ctx, newHR); err != nil {
			log.Error(err, "error updating the HelmRelease Resource")
			r.Events.Eventf(application, corev1.EventTypeWarning, ErrorReason, err.Error())
			return err
		}
		application.Status.HelmReleaseGeneration = newHR.Generation
	}
	return nil
}

func (r *ApplicationReconciler) getApplicationFromPod(req ctrl.Request, pod *corev1.Pod, application *v1.Application) error {
	if err := r.Get(context.Background(), client.ObjectKey{Namespace: req.Namespace, Name: req.Name}, pod); err != nil {
		return err
	}
	if len(pod.GetOwnerReferences()) < 1 {
		err := errors.New("no ownerReference for Pod")
		return err
	}
	replicaSetReference := pod.GetOwnerReferences()[0]
	if replicaSetReference.Kind != "ReplicaSet" {
		err := errors.New("pod owner reference is not a replicaset")
		return err
	}
	replicaSet := &appsv1.ReplicaSet{}
	if err := r.Get(context.Background(), client.ObjectKey{Namespace: req.Namespace, Name: pod.OwnerReferences[0].Name}, replicaSet); err != nil {
		return err
	}
	deployment := &appsv1.Deployment{}
	if err := r.Get(context.Background(), client.ObjectKey{Namespace: req.Namespace, Name: replicaSet.OwnerReferences[0].Name}, deployment); err != nil {
		return err
	}
	if deployment.Labels[LabelHelmReleaseName] == "" {
		err := errors.New("deployment does not have the helm release labels")
		return err
	}

	return r.Get(context.Background(), client.ObjectKey{Namespace: req.Namespace, Name: deployment.Labels[LabelHelmReleaseName]}, application)
}
