package client

import (
	"os/user"
	"path"

	certmanagerclientset "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func NewClientset() (*kubernetes.Clientset, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func NewCertManagerClientset() (*certmanagerclientset.Clientset, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}

	return certmanagerclientset.NewForConfig(config)
}

func GetConfig() (*restclient.Config, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	kubeconfig := path.Join(usr.HomeDir, ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
