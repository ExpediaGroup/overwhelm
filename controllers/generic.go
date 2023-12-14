package controllers

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

var clusterData = make(map[string]map[string]string)

const ReferenceLabel = "overwhelm.expediagroup.com/render-values-source"
const ApplicationKey = "application"
const ExpediaType = "k8s.expediagroup.com"

func LoadClusterData() {
	labelOptions := informers.WithTweakListOptions(func(opts *metav1.ListOptions) {
		opts.LabelSelector = ReferenceLabel
	})
	factory := informers.NewSharedInformerFactoryWithOptions(kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie()), 0, labelOptions)
	informer := factory.Core().V1().ConfigMaps().Informer()
	stop := make(chan struct{})
	defer close(stop)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(cm interface{}) {
			AddToClusterData(cm.(*v1.ConfigMap))
		},
		UpdateFunc: func(oldCM interface{}, cm interface{}) {
			AddToClusterData(cm.(*v1.ConfigMap))
		},
		DeleteFunc: func(cm interface{}) {
			delete(clusterData, cm.(*v1.ConfigMap).Labels[ReferenceLabel])
		},
	})
	informer.Run(stop)
}

func AddToClusterData(cm *v1.ConfigMap) {
	for k, v := range cm.Data {
		if clusterData[cm.Labels[ReferenceLabel]] == nil {
			clusterData[cm.Labels[ReferenceLabel]] = make(map[string]string)
		}
		clusterData[cm.Labels[ReferenceLabel]][k] = v
	}
}

func GetPreRenderData(labels map[string]string) map[string]map[string]string {
	preRenderData := make(map[string]map[string]string)
	for key, value := range clusterData {
		preRenderData[key] = value
	}
	for label, labelValue := range labels {
		if strings.HasPrefix(label, ExpediaType) {
			//Initialise the map, even if there is a single label matching criteria
			if preRenderData[ApplicationKey] == nil {
				preRenderData[ApplicationKey] = make(map[string]string)
			}
			trimmedLabel := strings.Trim(strings.TrimPrefix(label, ExpediaType), "/")
			preRenderData[ApplicationKey][trimmedLabel] = labelValue
		}
	}

	return preRenderData
}
