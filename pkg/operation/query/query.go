package query

import (
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang/glog"

	"github.com/JulienBalestra/kube-csr/pkg/utils/kubeclient"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

const defaultNamespace = "default"

// Config of the Query
type Config struct {
	PollingInterval time.Duration
	PollingTimeout  time.Duration
}

// Query state
type Query struct {
	conf            *Config
	kubeClient      *kubeclient.KubeClient
	servicesToQuery []*serviceToQuery
}

type serviceToQuery struct {
	ns  string
	svc string
	ok  bool
}

// NewQuery creates a new Query
func NewQuery(kubeConfigPath string, svcToQuery []string, conf *Config) (*Query, error) {
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
	currentNamespace := defaultNamespace
	if kubeConfigPath == "" {
		b, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			glog.Warningf("Unexpected error during namespace detection: %v, fallback to %q", err, defaultNamespace)
		} else {
			currentNamespace = string(b)
			glog.V(2).Infof("Detected namespace: %q", currentNamespace)
		}
	}

	var servicesToQuery []*serviceToQuery
	for _, elt := range svcToQuery {
		i := strings.IndexByte(elt, '/')
		if i != -1 {
			glog.V(2).Infof("Adding %s as kube-service to query", elt)
			servicesToQuery = append(servicesToQuery, &serviceToQuery{
				ns:  elt[:i],
				svc: elt[i+1:],
			})
			continue
		}
		glog.Warningf("Missing namespace in %s, using %q as replacement", elt, currentNamespace)
		servicesToQuery = append(servicesToQuery, &serviceToQuery{
			ns:  currentNamespace,
			svc: elt,
		})
	}
	return &Query{
		kubeClient:      k,
		conf:            conf,
		servicesToQuery: servicesToQuery,
	}, nil
}

// GetKubernetesServicesSubjectAlternativeNames query the kube-apiserver to grab all
// potentials SAN in each service given to query
func (q *Query) GetKubernetesServicesSubjectAlternativeNames() ([]string, error) {
	ticker := time.NewTicker(q.conf.PollingInterval)
	defer ticker.Stop()

	timeout := time.NewTimer(q.conf.PollingTimeout)
	defer timeout.Stop()

	var sans []string
	for {
		select {
		case <-ticker.C:
			for _, elt := range q.servicesToQuery {
				if elt.ok {
					glog.V(0).Infof("svc/%s in namespace %s already queried", elt.svc, elt.ns)
					continue
				}
				svc, err := q.kubeClient.GetKubernetesClient().CoreV1().Services(elt.ns).Get(elt.svc, metav1.GetOptions{})
				if err != nil {
					if !errors.IsNotFound(err) {
						glog.Errorf("Unexpected error during query svc/%s in namespace %s: %v", elt.svc, elt.ns, err)
						return nil, err
					}
					glog.V(0).Infof("svc/%s in namespace %s is not found", elt.svc, elt.ns)
					continue
				}
				glog.V(2).Infof("svc/%s in namespace %s returns %s", elt.svc, elt.ns, svc.String())
				if svc.Spec.ClusterIP != "" {
					glog.V(0).Infof("adding SAN .Spec.ClusterIP: %s", svc.Spec.ClusterIP)
					sans = append(sans, svc.Spec.ClusterIP)
				}
				if len(svc.Spec.ExternalIPs) > 0 {
					glog.V(0).Infof("adding SAN .Spec.ExternalIPs: %s", svc.Spec.ExternalIPs)
					sans = append(sans, svc.Spec.ExternalIPs...)
				}
				if svc.Spec.LoadBalancerIP != "" {
					glog.V(0).Infof("adding SAN .Spec.LoadBalancerIP: %s", svc.Spec.LoadBalancerIP)
					sans = append(sans, svc.Spec.LoadBalancerIP)
				}
				if svc.Spec.ExternalName != "" {
					glog.V(0).Infof("adding SAN .Spec.ExternalName: %s", svc.Spec.ExternalName)
					sans = append(sans, svc.Spec.ExternalName)
				}
				elt.ok = true
			}
			todo, done := 0, 0
			for _, elt := range q.servicesToQuery {
				if elt.ok {
					done++
					continue
				}
				todo++
			}
			// TODO remove duplicates, if any
			if todo == 0 && done == len(q.servicesToQuery) {
				glog.V(0).Infof("Successfully query %d/%d services with %d SANs", done, len(q.servicesToQuery), len(sans))
				return sans, nil
			}
		case <-timeout.C:
			err := fmt.Errorf("timeout of %s reached when trying to query kube-services", q.conf.PollingTimeout.String())
			glog.Errorf("Cannot get service SAN: %v", err)
			return nil, err
		}
	}
}
