package fetch

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
	"github.com/JulienBalestra/kube-csr/pkg/utils/pemio"
)

// Config of the Fetch
type Config struct {
	Override              bool
	PollingInterval       time.Duration
	PollingTimeout        time.Duration
	CertificateABSPath    string
	CertificatePermission os.FileMode
}

// Fetch state
type Fetch struct {
	conf       *Config
	kubeClient *kubeclient.KubeClient
}

// NewFetcher creates a new Fetch
func NewFetcher(kubeConfigPath string, conf *Config) (*Fetch, error) {
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return &Fetch{
		kubeClient: k,
		conf:       conf,
	}, nil
}

// Fetch the generated certificate from the CSR
func (f *Fetch) Fetch(csr *generate.Config) error {
	glog.V(2).Infof("Start polling for certificate of csr/%s, every %s, timeout after %s", csr.Name, f.conf.PollingInterval.String(), f.conf.PollingTimeout.String())

	tick := time.NewTicker(f.conf.PollingInterval)
	defer tick.Stop()

	timeout := time.NewTimer(f.conf.PollingTimeout)
	defer tick.Stop()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(ch)

	for {
		select {
		case s := <-ch:
			glog.Infof("Signal %s received, exiting ...", s.String())
			return fmt.Errorf("%s", s.String())

		case <-tick.C:
			r, err := f.kubeClient.GetCertificateClient().CertificateSigningRequests().Get(csr.Name, v1.GetOptions{})
			if err != nil {
				glog.Errorf("Unexpected error during certificate fetching of csr/%s: %s", csr.Name, err)
				return err
			}
			if r.Status.Certificate != nil {
				glog.V(3).Infof("csr/%s:\n%s", csr.Name, string(r.Status.Certificate))
				glog.V(2).Infof("Certificate successfully fetched, writing %d chars to %s", len(r.Status.Certificate), f.conf.CertificateABSPath)
				return pemio.WriteFile(r.Status.Certificate, f.conf.CertificateABSPath, f.conf.CertificatePermission, f.conf.Override)
			}
			glog.V(2).Infof("Certificate of csr/%s still not available, next try in %s", csr.Name, f.conf.PollingInterval.String())

		case <-timeout.C:
			return fmt.Errorf("timeout during certificate fetching of csr/%s", csr.Name)
		}
	}
}
