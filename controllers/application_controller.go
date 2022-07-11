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
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/ExpediaGroup/overwhelm/analyzer"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/ExpediaGroup/overwhelm/api/v1alpha1"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
	Retries         int64
}

const (
	FinalizerName = "overwhelm.expediagroup.com/finalizer"
	ManagedBy     = "app.kubernetes.io/managed-by"

	AnnotationHelmReleaseName      = "meta.helm.sh/release-name"
	AnnotationHelmReleaseNamespace = "meta.helm.sh/release-namespace"
)

var log logr.Logger

//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.expediagroup.com,resources=applications/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=helm.toolkit.fluxcd.io,resources=helmreleases/status,verbs=get;update;patch

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
	var err error
	log.Info("Starting to read application")
	if err = r.Get(ctx, req.NamespacedName, application); err != nil {
		if !apierrors.IsNotFound(err) {
			log.Error(err, "Error reading application object")
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
			application.Status.ValuesResourceVersion = ""
			application.Status.HelmReleaseResourceVersion = ""
			v1.AppInProgressStatus(application)
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
		}

		// At this point the Helm Release can be reconciled
		hr, err := r.reconcileHelmReleaseStatus(ctx, application)
		if patchErr := r.patchStatus(ctx, application); patchErr != nil {
			log.Error(patchErr, "Error updating application status")
			return ctrl.Result{}, patchErr
		}
		if err != nil {
			return ctrl.Result{}, err
		}

		// Retrieve the pod status
		err = r.reconcilePodStatus(ctx, application, hr)
		if patchErr := r.patchStatus(ctx, application); patchErr != nil {
			log.Error(patchErr, "Error updating application status")
			return ctrl.Result{}, patchErr
		}
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		if controllerutil.ContainsFinalizer(application, FinalizerName) {
			// Pre Delete actions go here.
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

func (r *ApplicationReconciler) reconcileHelmReleaseStatus(ctx context.Context, application *v1.Application) (*v2beta1.HelmRelease, error) {
	hr := v2beta1.HelmRelease{}
	err := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, &hr)
	if err == nil && hr.ResourceVersion != application.Status.HelmReleaseResourceVersion {
		err = errors.New("HelmRelease not updated")
	}

	if err != nil {
		if application.Status.Failures == r.Retries {
			if apierrors.IsNotFound(err) {
				v1.AppErrorStatus(application, "Helm Release not created")
			} else {
				v1.AppErrorStatus(application, err.Error())
			}
			return &hr, nil
		}
		application.Status.Failures++
		return nil, err
	}
	application.Status.Failures = 0
	if hr.Status.ObservedGeneration != hr.Generation {
		v1.AppErrorStatus(application, "updated Helm Release status not available")
		return nil, errors.New("HelmRelease status is not current")
	}
	for _, condition := range hr.GetConditions() {
		apimeta.SetStatusCondition(&application.Status.Conditions, condition)
	}
	return &hr, nil
}

func (r *ApplicationReconciler) reconcilePodStatus(ctx context.Context, application *v1.Application, helmRelease *v2beta1.HelmRelease) error {
	// Analyze the status of the pods under the HR, if any
	// TODO: use AppInProgressStatus and AppErrorStatus.. should also prob rename these two functions
	fmt.Println("reconciling pod status")

	// 1. use meta.helm.sh/release-name and meta.helm.sh/release-namespace to get the deployment
	// 2. get the spec.selector.matchLabels from the deployment
	// 3. retrieve the pods using the deployment

	// XXX: This is for deployments. We need to make it clear that right now, this feature is "best effort".
	var deploymentList appsv1.DeploymentList
	if err := r.List(ctx, &deploymentList, &client.ListOptions{Namespace: helmRelease.GetReleaseNamespace()}); err != nil {
		return err
	}

	var releaseDeployment *appsv1.Deployment
	for _, deployment := range deploymentList.Items {
		if deployment.GetAnnotations()[AnnotationHelmReleaseName] == helmRelease.GetReleaseName() && deployment.GetAnnotations()[AnnotationHelmReleaseNamespace] == helmRelease.GetReleaseNamespace() {
			releaseDeployment = &deployment
			break
		}
	}
	if releaseDeployment == nil {
		// We didn't find a release deployment, so no pod status.
		return nil
	}
	// Get ReplicaSet name
	replicaSetName := extractReplicaSetNameFromDeployment(releaseDeployment)
	if replicaSetName == "" {
		return errors.New("failed to extract ReplicaSet from Deployment")
	}
	// Make sure that the ReplicaSet name has the right format. This is optional, mostly for testing purposes.
	replicaSetNameParts := strings.Split(replicaSetName, "-")
	if len(replicaSetNameParts) != 2 {
		return errors.New("invalid ReplicaSet name format: expected format <DEPLOYMENT-NAME>-<RANDOM-STRING>")
	}
	var replicaSet appsv1.ReplicaSet
	if err := r.Get(ctx, types.NamespacedName{Namespace: releaseDeployment.Namespace, Name: replicaSetName}, &replicaSet); err != nil {
		return fmt.Errorf("failed to get ReplicaSet %s: %w", replicaSetName, err)
	}
	// Unlike the Deployment's Spec.Selector.MatchLabels, ReplicaSet's match labels include pod-template-hash, which
	// means we can use that to specifically target the pods of the version we want to analyze.
	matchLabels := replicaSet.Spec.Selector.MatchLabels
	var requirements []labels.Requirement
	for key, value := range matchLabels {
		requirement, err := labels.NewRequirement(key, selection.Equals, []string{value})
		if err != nil {
			return fmt.Errorf("error creating requirement from match label with key %s and value %s: %v", key, value, err)
		}
		requirements = append(requirements, *requirement)
	}
	podList := corev1.PodList{}
	if err := r.List(ctx, &podList, &client.ListOptions{Namespace: application.Namespace, LabelSelector: labels.NewSelector().Add(requirements...), Limit: 1}); err != nil {
		return err
	}
	for _, pod := range podList.Items {
		result := analyzer.Pod(pod)
		v1.AppAnalysisCondition(application, result)
		break
	}
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
		Complete(r)
}

func (r *ApplicationReconciler) createOrUpdateConfigMap(ctx context.Context, application *v1.Application) error {
	currentCM := corev1.ConfigMap{}
	currentCMError := r.Get(ctx, types.NamespacedName{
		Namespace: application.Namespace,
		Name:      application.Name,
	}, &currentCM)

	if currentCMError == nil && currentCM.ResourceVersion == application.Status.ValuesResourceVersion {
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
	newCM.Labels[ManagedBy] = "overwhelm"
	if err := ctrl.SetControllerReference(application, newCM, r.Scheme); err != nil {
		return err
	}

	if err := r.renderValues(application); err != nil {
		log.Error(err, "error rendering values", "values", application.Spec.Data)
		return err
	}

	if currentCMError != nil {
		if apierrors.IsNotFound(currentCMError) {
			if err := r.Create(ctx, newCM); err != nil {
				log.Error(err, "error creating the configmap", "configMap", newCM)
				return err
			}
			application.Status.ValuesResourceVersion = newCM.ResourceVersion
			return nil
		}
		log.Error(currentCMError, "error retrieving current configmap if exists", "configMap", newCM)
		return currentCMError
	}

	if err := r.Update(ctx, newCM); err != nil {
		log.Error(err, "error updating the configmap", "configMap", newCM)
		return err
	}
	application.Status.ValuesResourceVersion = newCM.ResourceVersion
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
				return err
			}
			application.Status.HelmReleaseResourceVersion = newHR.ResourceVersion
			return nil
		}
		log.Error(err, "error checking if current HelmRelease exists")
		return err
	}
	if application.Status.HelmReleaseResourceVersion == "" || currentHR.ResourceVersion != application.Status.HelmReleaseResourceVersion {
		newHR.ResourceVersion = currentHR.ResourceVersion

		if err := r.Update(ctx, newHR); err != nil {
			log.Error(err, "error updating the HelmRelease Resource")
			return err
		}
		application.Status.HelmReleaseResourceVersion = newHR.ResourceVersion
	}
	return nil
}
