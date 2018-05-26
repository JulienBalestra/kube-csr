package submit

import (
	"io/ioutil"

	"github.com/golang/glog"
	certificates "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/generate"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
)

type Submit struct {
	Override       bool
	Approve        bool
	KubeConfigPath string

	kube *kubeclient.KubeClient
}

func (s *Submit) Submit(csr *generate.CSRConfig) error {
	s.kube = kubeclient.NewKubeClient(s.KubeConfigPath)
	err := s.kube.Build()
	if err != nil {
		return err
	}

	csrBytes, err := ioutil.ReadFile(csr.CSRABSPath)
	if err != nil {
		glog.Errorf("Cannot read CSR from file: %v", err)
		return err
	}
	csrString := string(csrBytes)
	glog.V(3).Infof("Creating csr/%s:\n%s", csr.Name, csrString)

	kubeCSR := &certificates.CertificateSigningRequest{
		TypeMeta: v1.TypeMeta{
			APIVersion: "certificates.k8s.io/v1beta1",
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
			},
		},
	}

	r, err := s.kube.GetCertificateClient().CertificateSigningRequests().Create(kubeCSR)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			glog.Errorf("Unexpected error during the creation of the CSR: %v", err)
			return err
		}
		if !s.Override {
			glog.Errorf("csr/%s already exists, use override or delete it before", csr.Name)
			return err
		}
		glog.Warningf("csr/%s already exists, deleting ...", csr.Name)
		err = s.kube.GetCertificateClient().CertificateSigningRequests().Delete(kubeCSR.Name, nil)
		if err != nil {
			glog.Errorf("Cannot delete csr/%s: %v", csr.Name, err)
			return err
		}
		glog.V(2).Infof("Successfully deleted csr/%s, re-creating ...", csr.Name)
		r, err = s.kube.GetCertificateClient().CertificateSigningRequests().Create(kubeCSR)
		if err != nil {
			glog.Errorf("Unexpected error during the creation of the csr/%s: %v", csr.Name, err)
			return err
		}
	}
	glog.V(2).Infof("Successfully created csr/%s", csr.Name)
	if s.Approve {
		return s.approve(r)
	}
	return nil
}

func (s *Submit) approve(r *certificates.CertificateSigningRequest) error {
	glog.V(2).Infof("Approving csr/%s ...", r.Name)
	r.Status.Conditions = append(r.Status.Conditions, certificates.CertificateSigningRequestCondition{
		Type:    certificates.CertificateApproved,
		Reason:  "kubeCSRApprove",
		Message: "This CSR was approved by kube-csr",
	})
	r, err := s.kube.GetCertificateClient().CertificateSigningRequests().UpdateApproval(r)
	if err != nil {
		glog.Errorf("Unexpected error during approval of the CSR: %v", err)
		return err
	}
	glog.V(2).Infof("csr/%s is approved", r.Name)
	// TODO propose to generate an kubernetes event
	return nil
}
