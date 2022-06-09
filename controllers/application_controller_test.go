package controllers

import (
	"context"
	"errors"
	"time"

	"github.com/ExpediaGroup/overwhelm/api/v1alpha1"
	"github.com/ExpediaGroup/overwhelm/pkg/data"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ctx context.Context

var application = &v1alpha1.Application{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "core.expediagroup.com/v1alpha1",
		Kind:       "Application",
	},
	ObjectMeta: metav1.ObjectMeta{
		Namespace: "default",
		Labels: map[string]string{
			"test-my-label": "ok",
		},
	},
	Spec: v1alpha1.ApplicationSpec{
		Chart: v2beta1.HelmChartTemplate{Spec: v2beta1.HelmChartTemplateSpec{
			Chart:   "good-chart",
			Version: "0.0.1",
			SourceRef: v2beta1.CrossNamespaceObjectReference{
				Kind: "HelmRepository",
				Name: "public-helm-virtual",
			},
		}},
		Interval:        metav1.Duration{Duration: time.Millisecond * 250},
		HelmReleaseName: "hr-test",
		Timeout:         &metav1.Duration{Duration: time.Millisecond * 10},
		Data:            map[string]string{"values.yaml": "deployment : hello-world \naccount : {{ .cluster.account }}\nregion : {{ .cluster.region }}\nenvironment : {{ .egdata.environment }}"},
	},
}

func cmEquals(key client.ObjectKey, expectedCM *v1.ConfigMap) func() error {
	return func() error {
		actual := &v1.ConfigMap{}
		if err := k8sClient.Get(ctx, key, actual); err != nil {
			return err
		}

		if Expect(actual.Data).Should(Equal(expectedCM.Data)) &&
			Expect(actual.Labels).Should(Equal(expectedCM.Labels)) &&
			Expect(actual.OwnerReferences).Should(Not(BeNil())) {
			return nil
		}
		return errors.New("actual cm not equal to expected cm")
	}
}

var _ = Describe("Application controller", func() {
	ctx = context.Background()
	data.LoadTestPrerenderData()

	Context("When creating an Application resource", func() {
		It("Should Deploy Successfully", func() {
			By("Creating a new ConfigMap and rendering it with default delimiter", func() {
				a := application.DeepCopy()
				a.Name = "a-app"
				Expect(k8sClient.Create(ctx, a)).Should(Succeed())
				expected := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        a.Name,
						Namespace:   a.Namespace,
						Labels:      a.Labels,
						Annotations: a.Annotations,
					},
					Data: map[string]string{"values.yaml": "deployment : hello-world \naccount : 1234\nregion : us-west-2\nenvironment : test"},
				}
				expected.Labels["app.kubernetes.io/managed-by"] = "overwhelm"

				Eventually(
					cmEquals(client.ObjectKey{Name: a.Name, Namespace: a.Namespace}, expected),
					time.Second*5, time.Millisecond*500).Should(BeNil())
			})
			By("Creating a new ConfigMap and rendering it with custom delimiter", func() {
				b := application.DeepCopy()
				b.Name = "b-app"
				b.Spec.PreRenderer = v1alpha1.PreRenderer{
					LeftDelimiter:        "<%",
					RightDelimiter:       "%>",
					EnableHelmTemplating: true,
				}
				b.Spec.Data = map[string]string{"values.yaml": "deployment : hello-world \naccount : {{ .cluster.account }}\nregion : <% .cluster.region %>\nenvironment : {{ .egdata.environment }}"}
				Expect(k8sClient.Create(ctx, b)).Should(Succeed())
				expected := &v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:        b.Name,
						Namespace:   b.Namespace,
						Labels:      b.Labels,
						Annotations: b.Annotations,
					},
					Data: map[string]string{"values.yaml": "deployment : hello-world \naccount : {{ .cluster.account }}\nregion : us-west-2\nenvironment : {{ .egdata.environment }}"},
				}
				expected.Labels["app.kubernetes.io/managed-by"] = "overwhelm"

				Eventually(
					cmEquals(client.ObjectKey{Name: b.Name, Namespace: b.Namespace}, expected),
					time.Second*5, time.Millisecond*500).Should(BeNil())
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

			Eventually(
				cmEquals(client.ObjectKey{Name: c.Name, Namespace: c.Namespace}, expected),
				time.Second*5, time.Millisecond*500).Should(Not(BeNil()))
		})
		By("having missing custom rendering keys in values", func() {
			d := application.DeepCopy()
			d.Name = "d-app"
			d.Spec.PreRenderer = v1alpha1.PreRenderer{
				LeftDelimiter:        "<%",
				RightDelimiter:       "%>",
				EnableHelmTemplating: true,
			}
			d.Spec.Data = map[string]string{"values.yaml": "deployment : hello-world \naccount : <% .cluster.someKey %>\nregion : <% .cluster.region %>\nenvironment : {{ .egdata.environment }}"}
			Expect(k8sClient.Create(ctx, d)).Should(Succeed())
			expected := &v1.ConfigMap{}

			Eventually(
				cmEquals(client.ObjectKey{Name: d.Name, Namespace: d.Namespace}, expected),
				time.Second*5, time.Millisecond*500).Should(Not(BeNil()))
		})
	})
})
