package fetch

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/golang/glog"
	certificates "k8s.io/api/certificates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
	"github.com/JulienBalestra/kube-csr/pkg/utils/pemio"
)

const (
	// KubeCSRFetchedAnnotationPrefix prefix
	KubeCSRFetchedAnnotationPrefix = "alpha.kube-csr/"

	// KubeCsrFetchedAnnotationDate is an annotation used to store the timestamp when the certificated has been fetched
	// This annotation is overridden by the latest fetch
	KubeCsrFetchedAnnotationDate = KubeCSRFetchedAnnotationPrefix + "lastFetchTime"

	// KubeCsrFetchedAnnotationDateFormat matches the Kubernetes date format
	KubeCsrFetchedAnnotationDateFormat = "2006-01-02T15:04:05Z"

	// KubeCsrFetchedAnnotationNb is an annotation used to count the number of fetches of the certificate
	KubeCsrFetchedAnnotationNb = KubeCSRFetchedAnnotationPrefix + "fetchCount"
)

// Config of the Fetch
type Config struct {
	Override              bool
	PollingInterval       time.Duration
	PollingTimeout        time.Duration
	CertificateABSPath    string
	CertificatePermission os.FileMode
	Annotate              bool
}

// Fetch state
type Fetch struct {
	Conf       *Config
	kubeClient *kubeclient.KubeClient
}

// NewFetcher creates a new Fetch
func NewFetcher(kubeConfigPath string, conf *Config) (*Fetch, error) {
	if conf.PollingInterval == 0 {
		err := fmt.Errorf("invalid value for PollingInterval: %s", conf.PollingInterval.String())
		glog.Errorf("Cannot use the provided config: %v", err)
		return nil, err
	}
	if conf.PollingTimeout == 0 {
		err := fmt.Errorf("invalid value for PollingTimeout: %s", conf.PollingTimeout.String())
		glog.Errorf("Cannot use the provided config: %v", err)
		return nil, err
	}
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return &Fetch{
		kubeClient: k,
		Conf:       conf,
	}, nil
}

func (f *Fetch) updateAnnotations(r *certificates.CertificateSigningRequest) error {
	if !f.Conf.Annotate {
		glog.V(1).Infof("Skipping the annotations update")
		return nil
	}
	now := time.Now().UTC().Format(KubeCsrFetchedAnnotationDateFormat)
	if r.Annotations == nil {
		r.Annotations = map[string]string{
			KubeCsrFetchedAnnotationDate: now,
			KubeCsrFetchedAnnotationNb:   "1",
		}
	} else {
		r.Annotations[KubeCsrFetchedAnnotationDate] = now
		nbString := r.Annotations[KubeCsrFetchedAnnotationNb]
		// if the annotation doesn't exists, nbString is set to "" which is transformed to 0 by Atoi
		nb, err := strconv.Atoi(nbString)
		if err != nil {
			glog.Warningf("Cannot parse the annotation %s: %q: %v", KubeCsrFetchedAnnotationNb, nbString, err)
		}
		r.Annotations[KubeCsrFetchedAnnotationNb] = strconv.Itoa(nb + 1)
		glog.V(1).Infof("csr/%s was already fetched before, incr %q annotation to %s", r.Name, KubeCsrFetchedAnnotationNb, r.Annotations[KubeCsrFetchedAnnotationNb])
	}

	glog.V(2).Infof("Annotate csr/%s: %s: %s", r.Name, KubeCsrFetchedAnnotationDate, now)
	r, err := f.kubeClient.GetCertificateClient().CertificateSigningRequests().Update(r)
	if err != nil {
		glog.Errorf("Cannot update annotation of csr/%s: %v", r.Name, err)
		return err
	}
	return nil
}

// Fetch the generated certificate from the CSR
func (f *Fetch) Fetch(csrName string) error {
	glog.V(0).Infof("Start polling for certificate of csr/%s, every %s, timeout after %s", csrName, f.Conf.PollingInterval.String(), f.Conf.PollingTimeout.String())

	tick := time.NewTicker(f.Conf.PollingInterval)
	defer tick.Stop()

	timeout := time.NewTimer(f.Conf.PollingTimeout)
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
			// TODO as we are waiting the ticker, if the ticker is set to 10s, we start polling after 10s
			r, err := f.kubeClient.GetCertificateClient().CertificateSigningRequests().Get(csrName, metav1.GetOptions{})
			if err != nil {
				glog.Errorf("Unexpected error during certificate fetching of csr/%s: %s", csrName, err)
				return err
			}
			if r.Status.Certificate != nil {
				err := f.updateAnnotations(r)
				if err != nil {
					return err
				}
				glog.V(0).Infof("Certificate successfully fetched, writing %d chars to %s", len(r.Status.Certificate), f.Conf.CertificateABSPath)
				glog.V(2).Infof("csr/%s:\n%s", csrName, string(r.Status.Certificate))
				return pemio.WriteFile(r.Status.Certificate, f.Conf.CertificateABSPath, f.Conf.CertificatePermission, f.Conf.Override)
			}
			for _, c := range r.Status.Conditions {
				if c.Type == certificates.CertificateDenied {
					err := fmt.Errorf("csr/%s uid: %s is %q: %s", r.Name, r.UID, c.Type, c.String())
					glog.Errorf("Unexpected error during fetch: %v", err)
					return err
				}
			}
			glog.V(1).Infof("Certificate of csr/%s still not available, next try in %s", csrName, f.Conf.PollingInterval.String())

		case <-timeout.C:
			return fmt.Errorf("timeout during certificate fetching of csr/%s", csrName)
		}
	}
}
