package purge

import (
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
)

// Purge state
type Purge struct {
	kubeClient *kubeclient.KubeClient
}

// NewPurge creates a new Fetch
func NewPurge(kubeConfigPath string) (*Purge, error) {
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return &Purge{
		kubeClient: k,
	}, nil
}

// Purge asked for a delete of the given csrName to the kube-apiserver
func (p *Purge) Purge(csr *generate.Config) error {
	err := p.kubeClient.GetCertificateClient().CertificateSigningRequests().Delete(csr.Name, &v1.DeleteOptions{})
	if err != nil {
		glog.Errorf("Unexpected error during delete csr/%s: %v", csr.Name, err)
		return err
	}
	glog.V(0).Infof("Successfully deleted csr/%s", csr.Name)
	return nil
}
