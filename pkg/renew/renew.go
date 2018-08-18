package renew

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/util/uuid"

	"github.com/JulienBalestra/kube-csr/pkg/operation"
	"github.com/JulienBalestra/kube-csr/pkg/utils/api"
	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
	"strings"
)

// Config of the Renew
type Config struct {
	Operation                     *operation.Operation
	RenewThreshold                time.Duration
	RenewCommand                  string
	ExitOnRenew                   bool
	GenerateNewKubernetesCSR      bool
	PrometheusExporterBindAddress string
	RenewCheckInterval            time.Duration
}

// Renew state
type Renew struct {
	conf *Config

	kubernetesCSRBasename string
	kubeClient            *kubeclient.KubeClient
	promCertExpiration    prometheus.Gauge
	promCertNextRenew     prometheus.Gauge
	promRenewCount        prometheus.Counter
	promRenewErrorCount   prometheus.Counter
}

// RegisterPrometheusMetrics is a convenient function to create and register prometheus metrics
func RegisterPrometheusMetrics(r *Renew) error {
	r.promRenewCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_renew",
		Help: "Total number of certificate renew",
	})

	r.promRenewErrorCount = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "total_renew_errors",
		Help: "Total number of certificates renew errors",
	})
	r.promCertExpiration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "seconds_before_expiration",
		Help: "Total number of seconds left before the certificate is expired",
	})
	r.promCertNextRenew = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "seconds_before_renew",
		Help: "Total number of seconds left before the certificate is renewed",
	})
	err := prometheus.Register(r.promRenewCount)
	if err != nil {
		return err
	}
	err = prometheus.Register(r.promRenewErrorCount)
	if err != nil {
		return err
	}
	err = prometheus.Register(r.promCertExpiration)
	if err != nil {
		return err
	}
	err = prometheus.Register(r.promCertNextRenew)
	if err != nil {
		return err
	}
	return nil
}

func checkPaths(paths ...string) error {
	var errs []string
	for _, p := range paths {
		glog.V(1).Infof("Checking file requirement %s", p)
		_, err := os.Stat(p)
		if err == nil {
			glog.V(1).Infof("File exists %s", p)
			continue
		}
		errs = append(errs, err.Error())
	}
	if errs == nil {
		return nil
	}
	return fmt.Errorf("%s", strings.Join(errs, ", "))
}

// NewRenewer instantiate a new Renew with the given config
func NewRenewer(kubeConfigPath string, conf *Config) (*Renew, error) {
	if conf.RenewCheckInterval <= 0 {
		err := fmt.Errorf("non-positive interval for the renew check interval: %d", conf.RenewCheckInterval)
		glog.Errorf("Cannot use the given configuration: %v", err)
		return nil, err
	}
	err := checkPaths(conf.Operation.SourceConfig.PrivateKeyABSPath, conf.Operation.SourceConfig.CSRABSPath, conf.Operation.Fetch.Conf.CertificateABSPath)
	if err != nil {
		glog.Errorf("Missing files: %v", err)
		return nil, err
	}
	k, err := kubeclient.NewKubeClient(kubeConfigPath)
	if err != nil {
		return nil, err
	}
	conf.Operation.SourceConfig.Override = true
	conf.Operation.Fetch.Conf.Override = true
	r := &Renew{
		conf:                  conf,
		kubeClient:            k,
		kubernetesCSRBasename: conf.Operation.SourceConfig.Name,
	}
	err = RegisterPrometheusMetrics(r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Renew) shouldRenew() (bool, error) {
	certABSPath := r.conf.Operation.Fetch.Conf.CertificateABSPath
	b, err := ioutil.ReadFile(certABSPath)
	if err != nil {
		glog.Errorf("Cannot read current certificate: %v", err)
		return false, err
	}
	p, _ := pem.Decode(b)
	if p == nil {
		err = fmt.Errorf("cannot parse certificate %s", certABSPath)
		glog.Errorf("Unexpected error: %v")
		return false, err
	}
	cert, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		glog.Errorf("Cannot parse %s as certificate: %v", certABSPath, err)
		return false, err
	}
	now := time.Now()

	timeLeft := cert.NotAfter.Sub(now)
	r.promCertExpiration.Set(timeLeft.Seconds())

	timeLeftThreshold := timeLeft - r.conf.RenewThreshold
	r.promCertNextRenew.Set(timeLeftThreshold.Seconds())

	glog.V(0).Infof("Certificate %s is valid until: %s, time left: %s, time left with threshold: %s", certABSPath, cert.NotAfter, timeLeft.Round(time.Second).String(), timeLeftThreshold.String())
	if timeLeftThreshold.Seconds() > 0 {
		glog.V(1).Infof("Certificate %s doesn't need a renew yet", certABSPath)
		return false, nil
	}
	glog.V(0).Infof("Certificate %s needs renew since %s", certABSPath, timeLeftThreshold)
	return true, nil
}

func (r *Renew) processRenew() (bool, error) {
	needRenew, err := r.shouldRenew()
	if err != nil {
		return false, err
	}
	if !needRenew {
		return false, nil
	}
	if r.conf.GenerateNewKubernetesCSR {
		r.conf.Operation.SourceConfig.Name = fmt.Sprintf("%s-%s", r.kubernetesCSRBasename, uuid.NewUUID()[:13])
	}
	glog.V(0).Infof("Renewing CN=%s csr/%s ...", r.conf.Operation.SourceConfig.CommonName, r.conf.Operation.SourceConfig.Name)
	err = r.conf.Operation.Run()
	if err != nil {
		return false, err
	}
	r.promRenewCount.Inc()
	glog.V(0).Infof("Successfully renewed")
	return true, nil
}

// Renew starts the renew process
func (r *Renew) Renew() error {
	api.RegisterAPI(r.conf.PrometheusExporterBindAddress, api.PprofBindDefault)

	renewedCh := make(chan struct{}, 1)
	defer close(renewedCh)

	// processRenew once and fail fast to crash the Pod in case of error
	renewed, err := r.processRenew()
	if err != nil {
		return err
	}
	if renewed {
		renewedCh <- struct{}{}
	}

	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(r.conf.RenewCheckInterval)
	defer ticker.Stop()

	glog.V(0).Infof("Starting the renew process for the certificate %s, check every %s", r.conf.Operation.Fetch.Conf.CertificateABSPath, r.conf.RenewCheckInterval)
	for {
		select {
		case s := <-sigCh:
			glog.Infof("Signal %s received, exiting ...", s.String())
			return nil

		case <-renewedCh:
			if r.conf.RenewCommand != "" {
				b, err := exec.Command("/bin/sh", "-c", r.conf.RenewCommand).CombinedOutput()
				glog.V(0).Infof("Renew command %q output:\n%s", r.conf.RenewCommand, string(b))
				if err != nil {
					return err
				}
			}
			if !r.conf.ExitOnRenew {
				glog.V(0).Infof("Restarting the renew process for the certificate %s, check every %s", r.conf.Operation.Fetch.Conf.CertificateABSPath, r.conf.RenewCheckInterval)
				continue
			}
			glog.V(0).Infof("Exit on successful renew")
			return nil

		case <-ticker.C:
			renewed, err := r.processRenew()
			if err != nil {
				r.promRenewErrorCount.Inc()
				continue
			}
			if renewed {
				renewedCh <- struct{}{}
			}
		}
	}
}
