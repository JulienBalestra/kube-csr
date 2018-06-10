package kubeclient

import "github.com/golang/glog"

import (
	"time"

	certapi "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubeClient state for a Kubernetes client (inCluster or regular one)
type KubeClient struct {
	KubeConfigPath string

	certClient *certapi.CertificatesV1beta1Client
	restConfig *rest.Config
}

// NewKubeClient instanciate a new Kubernetes client, pass kubeConfigPath == "" to build an InCluster client
func NewKubeClient(kubeConfigPath string) (*KubeClient, error) {
	c := &KubeClient{
		KubeConfigPath: kubeConfigPath,
	}
	err := c.build()
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (k *KubeClient) buildInClusterConfig() error {
	glog.V(3).Infof("Building inCluster kube-config")
	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("Fail to build inCluster config: %v", err)
		return err
	}
	k.restConfig = kubeConfig
	return nil
}

func (k *KubeClient) buildFlagsConfig() error {
	glog.V(3).Infof("Building flags kube-config with %s", k.KubeConfigPath)
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", k.KubeConfigPath)
	if err != nil {
		glog.Errorf("Fail to build flags config: %v", err)
		return err
	}
	k.restConfig = kubeConfig
	return nil
}

func (k *KubeClient) build() error {
	kubeConfigFn := k.buildFlagsConfig
	if k.KubeConfigPath == "" {
		kubeConfigFn = k.buildInClusterConfig
	}
	err := kubeConfigFn()
	if err != nil {
		return err
	}
	k.restConfig.Timeout = time.Second * 3 // TODO conf this
	k.certClient, err = certapi.NewForConfig(k.restConfig)
	if err != nil {
		glog.Errorf("Cannot create certificate client: %v", err)
		return err
	}
	return nil
}

// GetCertificateClient returns the k8s object to work with the certificates API
func (k *KubeClient) GetCertificateClient() *certapi.CertificatesV1beta1Client {
	return k.certClient
}
