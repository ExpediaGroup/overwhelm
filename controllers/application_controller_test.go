package controllers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/ExpediaGroup/overwhelm/api/v1alpha2"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	"github.com/fluxcd/pkg/apis/meta"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

var ctx context.Context

var application = &v1alpha2.Application{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "core.expediagroup.com/v1alpha2",
		Kind:       "Application",
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Labels: map[string]string{
			"test-my-label": "ok",
		},
	},
	Spec: v1alpha2.ApplicationSpec{
		Template: v1alpha2.ReleaseTemplate{
			Metadata: v1alpha2.Metadata{
				Labels: map[string]string{"test-temp-label": "ok"},
			},
			Spec: v2beta1.HelmReleaseSpec{
				Chart: v2beta1.HelmChartTemplate{
					Spec: v2beta1.HelmChartTemplateSpec{
						Chart:   "good-chart",
						Version: "0.0.1",
						SourceRef: v2beta1.CrossNamespaceObjectReference{
							Kind: "HelmRepository",
							Name: "public-helm-virtual",
						},
					},
				},
				Interval:    metav1.Duration{Duration: time.Millisecond * 250},
				ReleaseName: "hr-test",
				Timeout:     &metav1.Duration{Duration: time.Millisecond * 10},
			},
		},
		Data: map[string]string{"values.yaml": "deployment : hello-world \naccount : {{ .cluster.account }}\nregion : {{ .cluster.region }}\nenvironment : {{ .egdata.environment }}"},
	},
}

var deployment = &appsv1.Deployment{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "apps/v1",
		Kind:       "Deployment",
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "appname",
		Labels: map[string]string{
			"app":                     "appname",
			LabelHelmReleaseName:      application.Name,
			LabelHelmReleaseNamespace: application.Namespace,
		},
	},
	Spec: appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "appname"},
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": "appname"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "container",
						Image: "good-image",
					},
				},
			},
		},
	},
}

var replicaSet = &appsv1.ReplicaSet{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "apps/v1",
		Kind:       "ReplicaSet",
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "appname-55f99cdb4b",
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "appname",
			UID:        "3192f76b-ba27-400d-a490-ee8c1871aa83",
		}},
	},
	Spec: appsv1.ReplicaSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": "appname", "pod-template-hash": "55f99cdb4b"},
		},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{"app": "appname", "pod-template-hash": "55f99cdb4b"},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					{
						Name:  "container",
						Image: "good-image",
					},
				},
			},
		},
	},
}

var pod = &v1.Pod{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Pod",
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Name:      "appname-55f99cdb4b-eeeeeeee",
		Labels: map[string]string{
			"app":               "appname",
			"pod-template-hash": "55f99cdb4b",
		},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "ReplicaSet",
			Name:       "appname-55f99cdb4b",
			UID:        "3192f76b-ba27-400d-a490-ee8c1871aa83",
		}},
	},
	Spec: v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:  "container",
				Image: "good-image",
			},
		},
	},
}

func LoadTestPrerenderData() {
	preRenderData["cluster"] = map[string]string{
		"cluster": "some-cluster",
		"region":  "us-west-2",
		"account": "1234",
		"segment": "some-segment",
	}
	preRenderData["egdata"] = map[string]string{
		"environment": "test",
	}
}

func cmEquals(key client.ObjectKey, expectedCM *v1.ConfigMap) func() error {
	var log = ctrllog.FromContext(ctx)
	log.Info("cmKeys", "keys", key)
	return func() error {
		actual := &v1.ConfigMap{}

		if err := k8sClient.Get(ctx, key, actual); err != nil {
			log.Error(err, "error getting configMap")
			return err
		}
		if Expect(actual.Data).Should(Equal(expectedCM.Data)) &&
			Expect(actual.Labels).Should(Equal(expectedCM.Labels)) &&
			Expect(actual.OwnerReferences).Should(Not(BeNil())) &&
			Expect(actual.OwnerReferences[0].UID).Should(Equal(expectedCM.OwnerReferences[0].UID)) {
			return nil
		}
		return errors.New("actual cm not equal to expected cm")
	}
}

var _ = Describe("Application controller", func() {
	ctx = context.Background()
	LoadTestPrerenderData()

	Context("When creating an Application resource", func() {

		It("Should Deploy Successfully", func() {

			By("Creating a new ConfigMap and rendering it with default delimiter", func() {
				a := application.DeepCopy()
				a.Name = "a-app"
				Expect(k8sClient.Create(ctx, a)).Should(Succeed())
				key := client.ObjectKey{Name: a.Name, Namespace: a.Namespace}
				expected := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        a.Name,
						Namespace:   a.Namespace,
						Labels:      map[string]string{"test-temp-label": "ok", "app.kubernetes.io/managed-by": "overwhelm"},
						Annotations: a.Annotations,
						OwnerReferences: []metav1.OwnerReference{{
							UID: a.GetUID(),
						}},
					},

					Data: map[string]string{"values.yaml": "deployment : hello-world \naccount : 1234\nregion : us-west-2\nenvironment : test"},
				}
				Eventually(cmEquals(key, expected), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				hr := &v2beta1.HelmRelease{}
				Eventually(
					func(ctx context.Context, key client.ObjectKey, hr *v2beta1.HelmRelease) func() error {
						return func() error {
							if err := k8sClient.Get(ctx, key, hr); err != nil {
								return err
							}
							if hr.OwnerReferences == nil || hr.OwnerReferences[0].UID != a.GetUID() {
								return errors.New("HelmRelease has owner reference or has incorrect owner reference")
							}
							return nil
						}
					}(ctx, key, hr), 5*time.Second, 300*time.Millisecond).Should(BeNil())
			})

			By("Updating Application Status with HR Status", func() {
				a := application.DeepCopy()
				a.Name = "a-app"
				key := client.ObjectKey{Name: a.Name, Namespace: a.Namespace}
				hr := &v2beta1.HelmRelease{}
				Eventually(
					func(ctx context.Context, key client.ObjectKey, hr *v2beta1.HelmRelease) func() error {
						return func() error {
							return k8sClient.Get(ctx, key, hr)
						}
					}(ctx, key, hr), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				hr.Status.ObservedGeneration = 1
				hr.Generation = hr.Status.ObservedGeneration
				conditions := []metav1.Condition{{
					Type:               meta.ReadyCondition,
					Status:             metav1.ConditionStatus(v1.ConditionTrue),
					ObservedGeneration: 1,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Message:            "Helm Release Reconciled",
					Reason:             meta.SucceededReason,
				}}
				hr.SetConditions(conditions)
				Expect(k8sClient.Status().Update(ctx, hr)).Should(BeNil())
				app := &v1alpha2.Application{}
				Eventually(
					func(ctx context.Context, key client.ObjectKey, app *v1alpha2.Application) func() error {
						return func() error {
							if err := k8sClient.Get(ctx, key, app); err != nil {
								return err
							}
							if app.Status.Conditions[0].Status != metav1.ConditionStatus(v1.ConditionTrue) {
								return errors.New("app status not updated with hr status")
							}
							return nil
						}
					}(ctx, key, app), 5*time.Second, 300*time.Millisecond).Should(BeNil())
			})

			By("Updating Application Status with pod Status", func() {
				a := application.DeepCopy()
				a.Name = "podstatus-app"
				Expect(k8sClient.Create(ctx, a)).Should(Succeed())
				hr := &v2beta1.HelmRelease{}
				Eventually(
					func(ctx context.Context, hr *v2beta1.HelmRelease) func() error {
						return func() error {
							return k8sClient.Get(ctx, client.ObjectKey{Name: a.Name, Namespace: a.Namespace}, hr)
						}
					}(ctx, hr), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				hr.Status.ObservedGeneration = 1
				hr.Generation = hr.Status.ObservedGeneration
				conditions := []metav1.Condition{{
					Type:               meta.ReadyCondition,
					Status:             metav1.ConditionStatus(v1.ConditionTrue),
					ObservedGeneration: 1,
					LastTransitionTime: metav1.NewTime(time.Now()),
					Message:            "Helm Release Reconciled",
					Reason:             meta.SucceededReason,
				}}
				hr.SetConditions(conditions)
				Expect(k8sClient.Status().Update(ctx, hr)).Should(BeNil())
				app := &v1alpha2.Application{}
				Eventually(
					func(ctx context.Context, app *v1alpha2.Application) func() error {
						return func() error {
							if err := k8sClient.Get(ctx, client.ObjectKey{Name: a.Name, Namespace: a.Namespace}, app); err != nil {
								return err
							}
							return nil
						}
					}(ctx, app), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				deployment.Labels[LabelHelmReleaseName] = a.Name
				deployment.Labels[LabelHelmReleaseNamespace] = a.Namespace
				Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
				deployment.Status.ObservedGeneration = 1
				deployment.Generation = deployment.Status.ObservedGeneration
				deployment.Status.Conditions = []appsv1.DeploymentCondition{
					{
						Status:             v1.ConditionTrue,
						LastTransitionTime: metav1.NewTime(time.Now()),
						Type:               appsv1.DeploymentProgressing,
						Message:            `ReplicaSet "appname-55f99cdb4b" is progressing.`,
						Reason:             "NewReplicaSetAvailable",
					},
				}
				Expect(k8sClient.Status().Update(ctx, deployment)).Should(BeNil())
				var deploymentList appsv1.DeploymentList
				Eventually(
					func(ctx context.Context, deploymentList appsv1.DeploymentList) func() error {
						return func() error {
							if err := k8sClient.List(ctx, &deploymentList, &client.ListOptions{Namespace: a.Namespace}); err != nil {
								return err
							}
							if len(deploymentList.Items) != 1 {
								return errors.New("failed to retrieve deployments")
							}
							return nil
						}
					}(ctx, deploymentList), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				Expect(k8sClient.Create(ctx, replicaSet)).Should(Succeed())
				rs := &appsv1.ReplicaSet{}
				Eventually(
					func(ctx context.Context, rs *appsv1.ReplicaSet) func() error {
						return func() error {
							if err := k8sClient.Get(ctx, client.ObjectKey{Name: replicaSet.Name, Namespace: a.Namespace}, rs); err != nil {
								return err
							}
							if rs == nil || rs.Name != replicaSet.Name {
								return errors.New("failed to retrieve ReplicaSet")
							}
							return nil
						}
					}(ctx, rs), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				Expect(k8sClient.Create(ctx, pod)).Should(Succeed())
				pod.Status.Conditions = []v1.PodCondition{
					{
						Type:               v1.PodReady,
						Status:             v1.ConditionFalse,
						LastTransitionTime: metav1.NewTime(time.Now()),
					},
				}
				pod.Status.ContainerStatuses = []v1.ContainerStatus{
					{
						Name:         "application",
						Ready:        false,
						RestartCount: 0,
						State: v1.ContainerState{
							Waiting: &v1.ContainerStateWaiting{Reason: "ImagePullBackOff", Message: `Back-off pulling image "secret/secret:secret"`},
						},
					},
				}
				Expect(k8sClient.Status().Update(ctx, pod)).Should(BeNil())
				p := &v1.Pod{}
				Eventually(
					func(ctx context.Context, p *v1.Pod) func() error {
						return func() error {
							if err := k8sClient.Get(ctx, client.ObjectKey{Name: pod.Name, Namespace: a.Namespace}, p); err != nil {
								return err
							}
							if p == nil || p.Name != pod.Name {
								return errors.New("failed to retrieve pod")
							}
							return nil
						}
					}(ctx, p), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				Eventually(
					func(ctx context.Context, app *v1alpha2.Application) func() error {
						return func() error {
							if err := k8sClient.Get(ctx, client.ObjectKey{Name: a.Name, Namespace: a.Namespace}, app); err != nil {
								return err
							}
							fmt.Println(app.Status.Conditions)
							if len(app.Status.Conditions) != 2 {
								return errors.New("waiting for Analysis condition")
							}
							if app.Status.Conditions[1].Message != `Pod appname-55f99cdb4b-eeeeeeee is unhealthy: [container 'application' is not ready and is in a waiting state due to reason 'ImagePullBackOff' with message 'Back-off pulling image "secret/secret:secret"']` {
								return errors.New("expected meaningful error message")
							}
							return nil
						}
					}(ctx, app), 5*time.Second, 300*time.Millisecond).Should(BeNil())
			})

			By("Creating a new ConfigMap and rendering it with custom delimiter", func() {
				b := application.DeepCopy()
				b.Name = "b-app"
				b.Spec.PreRenderer = v1alpha2.PreRenderer{
					LeftDelimiter:        "<%",
					RightDelimiter:       "%>",
					EnableHelmTemplating: true,
				}
				b.Spec.Data = map[string]string{"values.yaml": "deployment : hello-world \naccount : {{ .cluster.account }}\nregion : <% .cluster.region %>\nenvironment : {{ .egdata.environment }}"}
				Expect(k8sClient.Create(ctx, b)).Should(Succeed())
				key := client.ObjectKey{Name: b.Name, Namespace: b.Namespace}
				expected := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      b.Name,
						Namespace: b.Namespace,
						Labels:    map[string]string{"test-temp-label": "ok", "app.kubernetes.io/managed-by": "overwhelm"},
						OwnerReferences: []metav1.OwnerReference{{
							UID: b.GetUID(),
						}},
					},
					Data: map[string]string{"values.yaml": "deployment : hello-world \naccount : {{ .cluster.account }}\nregion : us-west-2\nenvironment : {{ .egdata.environment }}"},
				}

				Eventually(cmEquals(key, expected), 5*time.Second, 300*time.Millisecond).Should(BeNil())
			})

			By("Updating HelmRelease and configmap resources when application is updated", func() {
				a := application.DeepCopy()
				a.Name = "a-app"
				key := client.ObjectKey{Name: a.Name, Namespace: a.Namespace}
				currentApp := &v1alpha2.Application{}
				Expect(k8sClient.Get(ctx, key, currentApp)).Should(BeNil())
				a.ResourceVersion = currentApp.ResourceVersion
				a.Spec.Template.Spec.Interval = metav1.Duration{Duration: time.Millisecond * 500}
				a.Spec.Data = map[string]string{"values.yaml": "deployment : hello-world-is-updated \naccount : 1234\nregion : us-west-2\nenvironment : test"}

				expected := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        a.Name,
						Namespace:   a.Namespace,
						Labels:      map[string]string{"test-temp-label": "ok", "app.kubernetes.io/managed-by": "overwhelm"},
						Annotations: a.Annotations,
						OwnerReferences: []metav1.OwnerReference{{
							UID: currentApp.GetUID(),
						}},
					},
					Data: map[string]string{"values.yaml": "deployment : hello-world-is-updated \naccount : 1234\nregion : us-west-2\nenvironment : test"},
				}
				Expect(k8sClient.Update(ctx, a)).Should(Succeed())
				hr := &v2beta1.HelmRelease{}
				Eventually(
					func(ctx context.Context, key client.ObjectKey, hr *v2beta1.HelmRelease) func() error {
						return func() error {
							if err := k8sClient.Get(ctx, key, hr); err != nil {
								return err
							}
							if hr.Generation != 2 || hr.Status.ObservedGeneration != 1 {
								return errors.New("HelmRelease generations not updated")
							}
							if hr.OwnerReferences == nil || hr.OwnerReferences[0].UID != a.GetUID() {
								return errors.New("HelmRelease has owner reference or has incorrect owner reference")
							}
							return nil
						}
					}(ctx, key, hr), 5*time.Second, 300*time.Millisecond).Should(BeNil())
				Eventually(cmEquals(key, expected), 5*time.Second, 300*time.Millisecond).Should(BeNil())
			})

		})
	})

	It("Should Fail Resource Creation", func() {

		By("having missing rendering keys in values", func() {
			c := application.DeepCopy()
			c.Name = "c-app"
			c.Spec.Data = map[string]string{"values.yaml": "deployment : hello-world \naccount : {{ .cluster.someKey }}\nregion : <% .cluster.region %>\nenvironment : {{ .egdata.environment }}"}
			Expect(k8sClient.Create(ctx, c)).Should(Succeed())
			expected := &v1.ConfigMap{}
			key := client.ObjectKey{Name: c.Name, Namespace: c.Namespace}
			Eventually(cmEquals(key, expected), time.Second*5, time.Millisecond*500).Should(Not(BeNil()))
		})

		By("having missing custom rendering keys in values", func() {
			d := application.DeepCopy()
			d.Name = "d-app"
			d.Spec.PreRenderer = v1alpha2.PreRenderer{
				LeftDelimiter:        "<%",
				RightDelimiter:       "%>",
				EnableHelmTemplating: true,
			}
			d.Spec.Data = map[string]string{"values.yaml": "deployment : hello-world \naccount : <% .cluster.someKey %>\nregion : <% .cluster.region %>\nenvironment : {{ .egdata.environment }}"}
			Expect(k8sClient.Create(ctx, d)).Should(Succeed())
			expected := &v1.ConfigMap{}
			key := client.ObjectKey{Name: d.Name, Namespace: d.Namespace}
			Eventually(cmEquals(key, expected), time.Second*5, time.Millisecond*500).Should(Not(BeNil()))
		})
	})
})

func Test_extractReplicaSetNameFromDeployment(t *testing.T) {
	tests := []struct {
		deployment *appsv1.Deployment
		want       string
	}{
		{
			deployment: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Conditions: []appsv1.DeploymentCondition{{Type: appsv1.DeploymentProgressing, Message: `ReplicaSet "podname-55f99cdb4b" is progressing.`}},
				},
			},
			want: "podname-55f99cdb4b",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := extractReplicaSetNameFromDeployment(tt.deployment); got != tt.want {
				t.Errorf("extractReplicaSetNameFromDeployment() = %v, want %v", got, tt.want)
			}
		})
	}
}
