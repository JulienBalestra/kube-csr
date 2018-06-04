package submit

import (
	"io/ioutil"

	"github.com/golang/glog"
	certificates "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
)

// Config contains
// - Override: allows to replace any existing csr with the same name
type Config struct {
	Override bool
}

// Submit is created with NewSubmitter
type Submit struct {
	conf *Config

	kubeClient *kubeclient.KubeClient
}

// NewSubmitter is a Kubernetes client to create/apply csr
func NewSubmitter(kubeConfigPath string, conf *Config) (*Submit, error) {
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return &Submit{
		kubeClient: k,
		conf:       conf,
	}, nil
}

// Submit is equivalent to kubectl create ${CSR}, if the override is configured, it becomes kubectl apply ${CSR}
func (s *Submit) Submit(csr *generate.Config) (*certificates.CertificateSigningRequest, error) {
	csrBytes, err := ioutil.ReadFile(csr.CSRABSPath)
	if err != nil {
		glog.Errorf("Cannot read CSR from file: %v", err)
		return nil, err
	}
	csrString := string(csrBytes)
	glog.V(3).Infof("Creating csr/%s:\n%s", csr.Name, csrString)

	kubeCSR := &certificates.CertificateSigningRequest{
		TypeMeta: v1.TypeMeta{
			APIVersion: "certificates.k8s.io/v1beta1", // TODO detect this
			Kind:       "CertificateSigningRequest",
		},
		ObjectMeta: v1.ObjectMeta{
			Name: csr.Name,
		},
		Spec: certificates.CertificateSigningRequestSpec{
			Request: csrBytes,
			Groups:  []string{"system:authenticated"},
			Usages: []certificates.KeyUsage{
				certificates.UsageDigitalSignature,
				certificates.UsageKeyEncipherment,
				certificates.UsageServerAuth,
				certificates.UsageClientAuth,
			},
		},
	}

	r, err := s.kubeClient.GetCertificateClient().CertificateSigningRequests().Create(kubeCSR)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			glog.Errorf("Unexpected error during the creation of the CSR: %v", err)
			return nil, err
		}
		if !s.conf.Override {
			glog.Errorf("csr/%s already exists, use override or delete it before", csr.Name)
			return nil, err
		}
		glog.Warningf("csr/%s already exists, deleting ...", csr.Name)
		err = s.kubeClient.GetCertificateClient().CertificateSigningRequests().Delete(kubeCSR.Name, nil)
		if err != nil {
			glog.Errorf("Cannot delete csr/%s: %v", csr.Name, err)
			return nil, err
		}
		glog.V(2).Infof("Successfully deleted csr/%s, re-creating ...", csr.Name)
		r, err = s.kubeClient.GetCertificateClient().CertificateSigningRequests().Create(kubeCSR)
		if err != nil {
			glog.Errorf("Unexpected error during the creation of the csr/%s: %v", csr.Name, err)
			return nil, err
		}
	}
	glog.V(2).Infof("Successfully created csr/%s %s", r.Name, r.UID)
	return r, nil
}
