package purge

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	certificates "k8s.io/api/certificates/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/JulienBalestra/kube-csr/pkg/operation/fetch"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
	"github.com/gorilla/mux"
	"net/http/pprof"
)

const (
	prometheusExporterPath = "/metrics"
)

// Config contains purge functions and the grace period
type Config struct {
	ShouldGC                      []func(*certificates.CertificateSigningRequest, time.Duration) bool
	GracePeriod                   time.Duration
	PollingPeriod                 time.Duration
	PrometheusExporterBindAddress string
}

// Purge state
type Purge struct {
	conf       *Config
	kubeClient *kubeclient.KubeClient

	promKubeAPICSR            prometheus.Gauge
	promGarbageCollectLatency prometheus.Histogram
	promDeleteCounter         prometheus.Counter
	promDeleteCounterError    prometheus.Counter
}

// NewPurgeConfig returns a Purge Config
func NewPurgeConfig(gracePeriod time.Duration, fns ...func(csr *certificates.CertificateSigningRequest, gracePeriod time.Duration) bool) *Config {
	return &Config{
		GracePeriod: gracePeriod,
		ShouldGC:    fns,
	}
}

// RegisterPrometheusMetrics is a convenient function to create and register prometheus metrics
func RegisterPrometheusMetrics(p *Purge) error {
	p.promKubeAPICSR = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "kubernetes_apiserver_csr",
		Help: "Number of Kubernetes Certificate Signing Requests reported by the Kubernetes API",
	})
	p.promGarbageCollectLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "kubernetes_csr_garbage_collect_latency_seconds",
		Help: "Latency of garbage collection operations",
	})
	p.promDeleteCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubernetes_csr_deletes",
		Help: "Total number of Kubernetes Certificate Signing Requests deleted",
	})
	p.promDeleteCounterError = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "kubernetes_csr_delete_errors",
		Help: "Total number of Kubernetes Certificate Signing Requests deletion errors",
	})
	err := prometheus.Register(p.promKubeAPICSR)
	if err != nil {
		return err
	}
	err = prometheus.Register(p.promGarbageCollectLatency)
	if err != nil {
		return err
	}
	err = prometheus.Register(p.promDeleteCounter)
	if err != nil {
		return err
	}
	err = prometheus.Register(p.promDeleteCounterError)
	if err != nil {
		return err
	}
	return nil
}

// NewPurge creates a new Fetch
func NewPurge(kubeConfigPath string, conf *Config) (*Purge, error) {
	if conf.PollingPeriod == 0 {
		err := fmt.Errorf("invalid value for PollingPeriod: %s", conf.PollingPeriod.String())
		glog.Errorf("Cannot use the provided config: %v", err)
		return nil, err
	}
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	p := &Purge{
		conf:       conf,
		kubeClient: k,
	}
	err = RegisterPrometheusMetrics(p)
	if err != nil {
		return nil, err
	}
	return p, nil
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
	now := time.Now().Unix()
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
			p.promDeleteCounter.Inc()
			purged++
		}
	}

	// metrics
	elapsedSeconds := time.Now().Unix() - now
	p.promGarbageCollectLatency.Observe(float64(elapsedSeconds))
	p.promKubeAPICSR.Set(float64(len(csrList.Items) - purged))

	// logging
	if purged > 0 {
		glog.V(0).Infof("Successfully garbage collected %d csr in %ds", purged, elapsedSeconds)
		return nil
	}
	glog.V(0).Infof("Ended without garbage collect in %ds", elapsedSeconds)
	return nil
}

func (p *Purge) registerAPI() {
	if p.conf.PrometheusExporterBindAddress != "" {
		promRouter := mux.NewRouter()
		promRouter.Path(prometheusExporterPath).Methods("GET").Handler(promhttp.Handler())
		promServer := &http.Server{
			Handler:      promRouter,
			Addr:         p.conf.PrometheusExporterBindAddress,
			WriteTimeout: 15 * time.Second,
			ReadTimeout:  15 * time.Second,
		}
		glog.V(0).Infof("Starting prometheus exporter on %s%s", p.conf.PrometheusExporterBindAddress, prometheusExporterPath)
		go promServer.ListenAndServe()
	}

	// Known issue with Mux and the registering of pprof:
	// https://stackoverflow.com/questions/19591065/profiling-go-web-application-built-with-gorillas-mux-with-net-http-pprof
	const pprofBind = "127.0.0.1:6060"
	pprofRouter := mux.NewRouter()
	pprofRouter.HandleFunc("/debug/pprof/", pprof.Index)
	pprofRouter.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	pprofRouter.HandleFunc("/debug/pprof/profile", pprof.Profile)
	pprofRouter.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	pprofRouter.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
	pprofServer := &http.Server{
		Handler:      pprofRouter,
		Addr:         pprofBind,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  60 * time.Second,
	}
	glog.V(0).Infof("Starting pprof on %s/debug/pprof", pprofBind)
	go pprofServer.ListenAndServe()
}

// GarbageCollectLoop runs the GC on ticker, returns on error or SIGINT/TERM
func (p *Purge) GarbageCollectLoop() error {
	p.registerAPI()
	tick := time.NewTicker(p.conf.PollingPeriod)
	defer tick.Stop()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Reset(syscall.SIGINT, syscall.SIGTERM)
	defer close(ch)

	glog.V(0).Infof("Starting gc loop, first run in %s", p.conf.PollingPeriod.String())
	for {
		select {
		case <-ch:
			glog.V(0).Infof("Exiting ...")
			return nil

		case <-tick.C:
			err := p.GarbageCollect()
			if err != nil {
				p.promDeleteCounterError.Inc()
			}
			glog.V(0).Infof("GC loop, next run in %s", p.conf.PollingPeriod.String())
		}
	}
}
