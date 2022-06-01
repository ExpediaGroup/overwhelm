package controllers

import (
	"context"
	"time"

	"github.com/ExpediaGroup/overwhelm/api/v1alpha1"
	"github.com/ExpediaGroup/overwhelm/data/reference"
	"github.com/fluxcd/helm-controller/api/v2beta1"
	helmControllerV1 "github.com/fluxcd/helm-controller/api/v2beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var application = &v1alpha1.Application{
	TypeMeta: metav1.TypeMeta{
		APIVersion: "core.expediagroup.com/v1alpha1",
		Kind:       "Application",
	},
	ObjectMeta: metav1.ObjectMeta{
		Name:      "application-test",
		Namespace: "default",
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

var clusterProperties = &v1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "cluster-properties",
		Namespace: "kube-public",
		Labels:    map[string]string{"k8s.expediagroup.com/helm-values-source": "cluster"},
	},
	Data: map[string]string{
		"cluster": "rcp-xyz",
		"region":  "us-west-2",
		"account": "1234",
		"segment": "oos",
	},
}

var someOtherProperties = &v1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "some-other-properties",
		Namespace: "kube-public",
		Labels:    map[string]string{"k8s.expediagroup.com/helm-values-source": "egdata"},
	},
	Data: map[string]string{
		"environment": "test",
	},
}

func getResourceFunc(ctx context.Context, key client.ObjectKey, obj client.Object) func() error {
	return func() error {
		return k8sClient.Get(ctx, key, obj)
	}
}

var _ = Describe("Application controller", func() {
	ctx := context.Background()

	Context("When creating an Application resource", func() {
		It("Should Deploy Successfully", func() {
			By("By creating a new ConfigMap")
			reference.LoadTestPrerenderData()
			Expect(k8sClient.Create(ctx, application)).Should(Succeed())
			configMap := &v1.ConfigMap{}
			Eventually(
				getResourceFunc(ctx, client.ObjectKey{Name: application.Name, Namespace: application.Namespace}, configMap),
				time.Second*5, time.Millisecond*500).Should(BeNil())
		})
	})
})

var _ = Describe("Application controller", func() {
	ctx := context.Background()

	Context("When creating an Application resource", func() {
		It("Should Deploy Successfully", func() {
			By("By creating a new HelmRelease")
			reference.LoadTestPrerenderData()
			Expect(k8sClient.Create(ctx, application)).Should(Succeed())
			hr := &helmControllerV1.HelmRelease{}
			Eventually(
				getResourceFunc(ctx, client.ObjectKey{Name: application.Name, Namespace: application.Namespace}, hr),
				time.Second*5, time.Millisecond*500).Should(BeNil())
		})
	})
})
