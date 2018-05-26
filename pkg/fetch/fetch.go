package fetch

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/generate"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
	"github.com/JulienBalestra/kube-csr/pkg/utils/pemio"
)

type Fetch struct {
	KubeConfigPath string
	Override       bool

	PollingInterval time.Duration
	PollingTimeout  time.Duration

	CertificateABSPath    string
	CertificatePermission os.FileMode

	kube *kubeclient.KubeClient
}

func (f *Fetch) Fetch(csr *generate.CSRConfig) error {
	f.kube = kubeclient.NewKubeClient(f.KubeConfigPath)
	err := f.kube.Build()
	if err != nil {
		return err
	}
	glog.V(2).Infof("Start polling for certificate of csr/%s, every %s, timeout after %s", csr.Name, f.PollingInterval.String(), f.PollingTimeout.String())
	tick := time.NewTicker(f.PollingInterval)
	defer tick.Stop()
	timeout := time.NewTimer(f.PollingTimeout)
	defer tick.Stop()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case s := <-ch:
			glog.Infof("Signal %s received, exiting ...", s.String())
			return fmt.Errorf("%s", s.String())

		case <-tick.C:
			r, err := f.kube.GetCertificateClient().CertificateSigningRequests().Get(csr.Name, v1.GetOptions{})
			if err != nil {
				glog.Errorf("Unexpected error during certificate fetching of csr/%s: %s", csr.Name, err)
				return err
			}
			if r.Status.Certificate != nil {
				glog.V(3).Infof("csr/%s:\n%s", csr.Name, string(r.Status.Certificate))
				glog.V(2).Infof("Certificate successfully fetched, writing %d chars to %s", len(r.Status.Certificate), f.CertificateABSPath)
				return pemio.WriteFile(r.Status.Certificate, f.CertificateABSPath, f.CertificatePermission, f.Override)
			}
			glog.V(2).Infof("Certificate of csr/%s still not available, next try in %s", csr.Name, f.PollingInterval.String())

		case <-timeout.C:
			return fmt.Errorf("timeout during certificate fetching of csr/%s", csr.Name)
		}
	}
}
