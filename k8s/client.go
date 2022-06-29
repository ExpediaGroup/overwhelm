package k8s

import (
	"errors"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// CreateClient initializes a Kubernetes client using either the kubeconfig file (if ENVIRONMENT is set to dev)
// or the in-cluster config otherwise.
func CreateClient() (*kubernetes.Clientset, error) {
	var cfg *rest.Config
	if os.Getenv("ENVIRONMENT") == "dev" {
		var kubeconfig string
		if home := homeDir(); home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		} else {
			return nil, errors.New("home directory not found")
		}
		// use the current context in kubeconfig
		clientConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
		cfg = clientConfig
	} else {
		clientConfig, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		cfg = clientConfig
	}
	return kubernetes.NewForConfig(cfg)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
