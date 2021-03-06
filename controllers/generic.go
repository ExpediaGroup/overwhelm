package controllers

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
)

var preRenderData = make(map[string]map[string]string)

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
			preRenderData[cm.(*v1.ConfigMap).Labels[ReferenceLabel]] = cm.(*v1.ConfigMap).Data
		},
		UpdateFunc: func(oldCM interface{}, cm interface{}) {
			preRenderData[cm.(*v1.ConfigMap).Labels[ReferenceLabel]] = cm.(*v1.ConfigMap).Data
		},
		DeleteFunc: func(cm interface{}) {
			delete(preRenderData, cm.(*v1.ConfigMap).Labels[ReferenceLabel])
		},
	})
	informer.Run(stop)
}

func GetPreRenderData() map[string]map[string]string {
	return preRenderData
}
