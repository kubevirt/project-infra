package client

import (
	"os"
	"os/user"
	"path"

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

func GetConfig() (*restclient.Config, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	kubeconfig := path.Join(usr.HomeDir, ".kube", "config")
	if os.Getenv("KUBECONFIG") != "" {
		kubeconfig = os.Getenv("KUBECONFIG")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
