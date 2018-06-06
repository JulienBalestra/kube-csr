package approve

import (
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
	"github.com/golang/glog"
	certificates "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Approval state
type Approval struct {
	kubeClient *kubeclient.KubeClient
}

// NewApproval creates a new Approval
func NewApproval(kubeConfigPath string) (*Approval, error) {
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return &Approval{
		kubeClient: k,
	}, nil
}

// GetCSR query the kube-apiserver to get the csrName from it
func (a *Approval) GetCSR(csrName string) (*certificates.CertificateSigningRequest, error) {
	r, err := a.kubeClient.GetCertificateClient().CertificateSigningRequests().Get(csrName, v1.GetOptions{})
	if err != nil {
		glog.Errorf("Unexpected error during get csr/%s: %v", csrName, err)
		return nil, err
	}
	return r, nil
}

// ApproveCSR approve the CSR
func (a *Approval) ApproveCSR(r *certificates.CertificateSigningRequest) error {
	glog.V(2).Infof("Approving csr/%s ...", r.Name)
	r.Status.Conditions = append(r.Status.Conditions, certificates.CertificateSigningRequestCondition{
		Type:    certificates.CertificateApproved,
		Reason:  "kubeCSRApprove",
		Message: "This CSR was approved by kubeClient-csr",
	})
	r, err := a.kubeClient.GetCertificateClient().CertificateSigningRequests().UpdateApproval(r)
	if err != nil {
		glog.Errorf("Unexpected error during approval of the CSR: %v", err)
		return err
	}
	glog.V(2).Infof("csr/%s is approved", r.Name)
	// TODO propose to generate a kubernetes event
	return nil
}

// GetAndApproveCSR first call GetCSR and then ApproveCSR on it
func (a *Approval) GetAndApproveCSR(csrName string) error {
	r, err := a.GetCSR(csrName)
	if err != nil {
		return err
	}
	return a.ApproveCSR(r)
}
