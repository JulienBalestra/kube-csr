package purge

import (
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/golang/glog"
	certificates "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/operation/fetch"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
)

// Config contains purge functions and the grace period
type Config struct {
	ShouldGC      []func(*certificates.CertificateSigningRequest, time.Duration) bool
	GracePeriod   time.Duration
	PollingPeriod time.Duration
}

// Purge state
type Purge struct {
	conf       *Config
	kubeClient *kubeclient.KubeClient
}

// NewPurgeConfig returns a Purge Config
func NewPurgeConfig(gracePeriod time.Duration, fns ...func(csr *certificates.CertificateSigningRequest, gracePeriod time.Duration) bool) *Config {
	return &Config{
		GracePeriod: gracePeriod,
		ShouldGC:    fns,
	}
}

// NewPurge creates a new Fetch
func NewPurge(kubeConfigPath string, conf *Config) (*Purge, error) {
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	return &Purge{
		conf:       conf,
		kubeClient: k,
	}, nil
}

// IsConditionDenied returns if the first condition of the csr is a type Denied
func IsConditionDenied(csr *certificates.CertificateSigningRequest, gracePeriod time.Duration) bool {
	nbCondition := len(csr.Status.Conditions)
	if nbCondition == 0 {
		glog.V(2).Infof("csr/%s uid: %s does not have any condition", csr.Name, csr.UID)
		return false
	}
	if nbCondition != 1 {
		glog.Warningf("csr/%s uid: %s has an unexpected number of conditions: %d", csr.Name, csr.UID, nbCondition)
	}
	condition := csr.Status.Conditions[0]
	if condition.Type != certificates.CertificateDenied {
		glog.V(2).Infof("csr/%s uid: %s does not start with condition %q: %q", csr.Name, csr.UID, certificates.CertificateDenied, condition.Type)
		return false
	}
	glog.V(1).Infof("csr/%s uid: %s is %q", csr.Name, csr.UID, certificates.CertificateDenied)

	now := time.Now()
	gracePeriodLimit := now.Add(-gracePeriod)
	glog.V(2).Infof("csr/%s uid: %s has been fetched on %s and grace period ends at %s", csr.Name, csr.UID, condition.LastUpdateTime.Format(fetch.KubeCsrFetchedAnnotationDateFormat), gracePeriodLimit.Format(fetch.KubeCsrFetchedAnnotationDateFormat))
	if condition.LastUpdateTime.After(gracePeriodLimit) {
		return false
	}
	glog.V(2).Infof("csr/%s uid: %s can be GC, has been %s", csr.Name, csr.UID, certificates.CertificateDenied)
	return true
}

// IsAnnotationFetched returns if the first condition of the csr is a type Denied
func IsAnnotationFetched(csr *certificates.CertificateSigningRequest, gracePeriod time.Duration) bool {
	if csr.Annotations == nil {
		glog.V(2).Infof("csr/%s uid: %s does not have any annotation", csr.Name, csr.UID)
		return false
	}
	nbString := csr.Annotations[fetch.KubeCsrFetchedAnnotationNb]
	nb, err := strconv.Atoi(nbString)
	if err != nil {
		glog.V(2).Infof("csr/%s uid: %s does not have a valid annotation %s: %q", csr.Name, csr.UID, fetch.KubeCsrFetchedAnnotationNb, nbString)
		return false
	}
	glog.V(2).Infof("csr/%s uid: %s has been fetched %d times", csr.Name, csr.UID, nb)
	if nb == 0 {
		return false
	}

	lastFetchTimeStr, ok := csr.Annotations[fetch.KubeCsrFetchedAnnotationDate]
	if !ok || lastFetchTimeStr == "" {
		glog.V(2).Infof("csr/%s uid: %s does not have a valid annotation %s: %q", csr.Name, csr.UID, fetch.KubeCsrFetchedAnnotationDate, nbString)
		return false
	}
	lastFetchTime, err := time.Parse(fetch.KubeCsrFetchedAnnotationDateFormat, lastFetchTimeStr)
	if err != nil {
		glog.Errorf("Fail to parse csr/%s uid: %s annotation date %s: %q: %v", csr.Name, csr.UID, fetch.KubeCsrFetchedAnnotationDate, lastFetchTimeStr, err)
		return false
	}

	now := time.Now()
	gracePeriodLimit := now.Add(-gracePeriod)
	glog.V(2).Infof("csr/%s uid: %s has been fetched on %s and grace period ends at %s", csr.Name, csr.UID, lastFetchTime.Format(fetch.KubeCsrFetchedAnnotationDateFormat), gracePeriodLimit.Format(fetch.KubeCsrFetchedAnnotationDateFormat))
	if lastFetchTime.After(gracePeriodLimit) {
		return false
	}
	glog.V(2).Infof("csr/%s uid: %s can be GC, already fetched", csr.Name, csr.UID)
	return true
}

// Delete asked for a delete of the given csrName to the kube-apiserver
func (p *Purge) Delete(csrName string) error {
	err := p.kubeClient.GetCertificateClient().CertificateSigningRequests().Delete(csrName, &v1.DeleteOptions{})
	if err != nil {
		glog.Errorf("Unexpected error during delete csr/%s: %v", csrName, err)
		return err
	}
	glog.V(0).Infof("Successfully deleted csr/%s", csrName)
	return nil
}

// GarbageCollect iter over all CSR from the kube-apiserver and delete them if needed
func (p *Purge) GarbageCollect() error {
	csrList, err := p.kubeClient.GetCertificateClient().CertificateSigningRequests().List(v1.ListOptions{})
	if err != nil {
		glog.Errorf("Cannot list all csr: %v", err)
		return err
	}
	glog.V(2).Infof("Kube-apiserver returns %d csr", len(csrList.Items))
	purged := 0
	for _, elt := range csrList.Items {
		glog.V(4).Infof("Got csr/%s", elt.Name)
		for _, fn := range p.conf.ShouldGC {
			if !fn(&elt, p.conf.GracePeriod) {
				continue
			}
			err := p.Delete(elt.Name)
			if err != nil {
				return err
			}
			purged++
		}
	}
	if purged > 0 {
		glog.V(0).Infof("Successfully garbage collected %d csr", purged)
	}
	return nil
}

// GarbageCollectLoop runs the GC on ticker, returns on error or SIGINT/TERM
func (p *Purge) GarbageCollectLoop() error {
	tick := time.NewTicker(p.conf.PollingPeriod)
	defer tick.Stop()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)
	defer close(ch)

	glog.V(0).Infof("Starting gc loop, next run in %s", p.conf.PollingPeriod.String())
	for {
		select {
		case <-ch:
			glog.V(0).Infof("Exiting ...")
			return nil

		case <-tick.C:
			err := p.GarbageCollect()
			if err != nil {
				return err
			}
			glog.V(0).Infof("GC loop, next run in %s", p.conf.PollingPeriod.String())
		}
	}
}
