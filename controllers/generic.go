package controllers

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
)

var preRenderData = make(map[string]map[string]string)

const expediaType = "k8s.expediagroup.com"

const ReferenceLabel = "overwhelm.expediagroup.com/render-values-source"

func LoadPreRenderData() {
	labelOptions := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.LabelSelector = ReferenceLabel
	})
	factory := informers.NewSharedInformerFactoryWithOptions(kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()), 0, labelOptions)
	informer := factory.Core().V1().ConfigMaps().Informer()
	stop := make(chan struct{})
	defer close(stop)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(cm interface{}) {
			addToPrerenderData(cm.(*v1.ConfigMap))
		},
		UpdateFunc: func(oldCM interface{}, cm interface{}) {
			addToPrerenderData(cm.(*v1.ConfigMap))
		},
		DeleteFunc: func(cm interface{}) {
			delete(preRenderData, cm.(*v1.ConfigMap).Labels[ReferenceLabel])
		},
	})
	informer.Run(stop)
}

func addToPrerenderData(cm *v1.ConfigMap) {
	for k, v := range cm.Data {
		if preRenderData[cm.Labels[ReferenceLabel]] == nil {
			preRenderData[cm.Labels[ReferenceLabel]] = make(map[string]string)
		}
		preRenderData[cm.Labels[ReferenceLabel]][k] = v
	}
}

func GetPreRenderData(appLabels map[string]string) map[string]map[string]string {
	for label, labelValue := range appLabels {
		if strings.HasPrefix(label, expediaType) {
			trimmedLabel := strings.Trim(strings.TrimPrefix(label, expediaType), "/")
			preRenderData["application"][trimmedLabel] = labelValue
		}
	}
	return preRenderData
}
