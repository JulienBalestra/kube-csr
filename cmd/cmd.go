package cmd

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/JulienBalestra/kube-csr/pkg/operation"
	"github.com/JulienBalestra/kube-csr/pkg/operation/approve"
	"github.com/JulienBalestra/kube-csr/pkg/operation/fetch"
	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/operation/purge"
	"github.com/JulienBalestra/kube-csr/pkg/operation/query"
	"github.com/JulienBalestra/kube-csr/pkg/operation/submit"
	"github.com/JulienBalestra/kube-csr/pkg/renew"
)

const (
	programName = "kube-csr"
)

var (
	viperConfig = viper.New()
)

// NewCommand creates a cobra command to be consumed in the main package
func NewCommand() (*cobra.Command, *int) {
	var verbose int
	var exitCode int

	rootCommand := &cobra.Command{
		Use:   programName,
		Short: "Use this command to manage Kubernetes certificates",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			flag.Lookup("alsologtostderr").Value.Set("true")
			flag.Lookup("v").Value.Set(strconv.Itoa(verbose))
		},
	}
	rootCommand.PersistentFlags().IntVarP(&verbose, "verbose", "v", 0, "verbose level")
	viperConfig.SetDefault("kubeconfig-path", "")
	rootCommand.PersistentFlags().String("kubeconfig-path", viperConfig.GetString("kubeconfig-path"), "Kubernetes config path, leave empty for inCluster config")
	viperConfig.BindPFlag("kubeconfig-path", rootCommand.PersistentFlags().Lookup("kubeconfig-path"))

	garbageCommandName := fmt.Sprintf("%s gc", programName)
	garbageCommand := &cobra.Command{
		Use:        "garbage-collect",
		Args:       cobra.ExactArgs(0),
		Aliases:    []string{"gc"},
		SuggestFor: []string{"purge", "delete", "remove", "del", "rm"},
		Short:      "Garbage collect Kubernetes certificates on different parameters",
		Example: fmt.Sprintf(`
# Garbage collect all csr already fetched with a grace period of 12 hours
%s --fetched --grace-period=12h

# Garbage collect all csr denied with a grace period of 15 minutes
%s --fetched --grace-period=15m

# Garbage collect now all csr already fetched
%s --fetched --grace-period=0s

# Garbage collect every 10min all csr already fetched with a grace period of 1 hour
%s --fetched --daemon polling-period=10m --grace-period=1h
`,
			garbageCommandName,
			garbageCommandName,
			garbageCommandName,
			garbageCommandName,
		),
		Run: func(cmd *cobra.Command, args []string) {
			if !viperConfig.GetBool("denied") &&
				!viperConfig.GetBool("fetched") &&
				!viperConfig.GetBool("expired") {
				glog.Errorf("Must choose at least one flag: --denied, --fetched, --expired")
				exitCode = 1
				return
			}
			gc, err := newGarbageCollector()
			if err != nil {
				exitCode = 1
				return
			}
			if viperConfig.GetBool("daemon") {
				err = gc.GarbageCollectLoop()
			} else {
				err = gc.GarbageCollect()
			}
			if err != nil {
				exitCode = 2
				return
			}
		},
	}
	rootCommand.AddCommand(garbageCommand)

	// gc safety period
	viperConfig.SetDefault("grace-period", time.Hour*48)
	garbageCommand.PersistentFlags().Duration("grace-period", viperConfig.GetDuration("grace-period"), "duration to wait before deleting Kubernetes csr objects")
	viperConfig.BindPFlag("grace-period", garbageCommand.PersistentFlags().Lookup("grace-period"))

	// gc functions
	viperConfig.SetDefault("denied", false)
	garbageCommand.PersistentFlags().Bool("denied", viperConfig.GetBool("denied"), "delete any denied Kubernetes csr")
	viperConfig.BindPFlag("denied", garbageCommand.PersistentFlags().Lookup("denied"))

	viperConfig.SetDefault("fetched", false)
	garbageCommand.PersistentFlags().Bool("fetched", viperConfig.GetBool("fetched"), fmt.Sprintf("delete any already fetched Kubernetes csr, the state is tracked with kube-annotations %q", fetch.KubeCSRFetchedAnnotationPrefix))
	viperConfig.BindPFlag("fetched", garbageCommand.PersistentFlags().Lookup("fetched"))

	viperConfig.SetDefault("expired", false)
	garbageCommand.PersistentFlags().Bool("expired", viperConfig.GetBool("expired"), fmt.Sprintf("delete any Kubernetes csr with an expired certificate"))
	viperConfig.BindPFlag("expired", garbageCommand.PersistentFlags().Lookup("expired"))

	// daemon flags
	pollingPeriod, daemon := "polling-period", "daemon"
	viperConfig.SetDefault(pollingPeriod, time.Minute*10)
	garbageCommand.PersistentFlags().Duration(pollingPeriod, viperConfig.GetDuration(pollingPeriod), fmt.Sprintf("duration to wait between each gc call, paired with --%s", daemon))
	viperConfig.BindPFlag(pollingPeriod, garbageCommand.PersistentFlags().Lookup(pollingPeriod))

	viperConfig.SetDefault(daemon, false)
	garbageCommand.PersistentFlags().Bool(daemon, viperConfig.GetBool(daemon), fmt.Sprintf("continually gc Kubernetes csr, paired with --%s", pollingPeriod))
	viperConfig.BindPFlag(daemon, garbageCommand.PersistentFlags().Lookup(daemon))

	viperConfig.SetDefault("disable-prometheus-exporter", false)
	garbageCommand.PersistentFlags().Bool("disable-prometheus-exporter", viperConfig.GetBool("disable-prometheus-exporter"), fmt.Sprintf("disable /metrics, paired with --%s", daemon))
	viperConfig.BindPFlag("disable-prometheus-exporter", garbageCommand.PersistentFlags().Lookup("disable-prometheus-exporter"))

	viperConfig.SetDefault("prometheus-exporter-bind", "0.0.0.0:8484")
	garbageCommand.PersistentFlags().Bool("prometheus-exporter-bind", viperConfig.GetBool("prometheus-exporter-bind"), fmt.Sprintf("prometheus exporter bind address, paired with --%s", daemon))
	viperConfig.BindPFlag("prometheus-exporter-bind", garbageCommand.PersistentFlags().Lookup("prometheus-exporter-bind"))

	// issue command
	issueCommandName := fmt.Sprintf("%s issue", programName)
	issueCommand := &cobra.Command{
		Use:        "issue",
		Aliases:    []string{"i"},
		SuggestFor: []string{"issuer", "isue", "issu"},
		Short:      "Use this command to generate, approve, fetch and self-delete Kubernetes certificates",
		Args:       cobra.ExactArgs(1),
		Example: fmt.Sprintf(`
# generate the private key and the csr
%s my-app --generate

# generate the private key, the csr and then submit the csr
%s my-app --generate --submit

# generate the private key, the csr, submit and approve the csr
%s my-app --generate --submit --approve

# Generate the private key, the csr, submit, approve and fetch the csr
%s my-app --generate --submit --approve --fetch
%s my-app -gsaf
%s my-app -gsaf --subject-alternative-names 192.168.1.1,etcd-0.default.svc.cluster.local

# Generate the private key, the csr, submit and fetch the csr when externally approved
%s my-app --generate --submit --fetch --fetch-interval 10s --fetch-timeout 10m

# Generate the private key, the csr, submit, approve, fetch and delete the csr
%s my-app --generate --submit --approve --fetch --delete 

# Generate the private key, the csr, submit, approve and fetch the csr. Override any existing and use a kubeconfig
%s my-app -gsaf --override --kubeconfig-path ~/.kube/config

# Execute all steps with a custom kubernetes csr name
%s skydns --csr-name kv-etcd -gsafd --override --kubeconfig-path ~/.kube/config
`,
			// TODO use go template here
			issueCommandName,
			issueCommandName,
			issueCommandName,
			issueCommandName,
			issueCommandName,
			issueCommandName,
			issueCommandName,
			issueCommandName,
			issueCommandName,
			issueCommandName,
		),
		Run: func(cmd *cobra.Command, args []string) {
			if !viperConfig.GetBool("generate") &&
				!viperConfig.GetBool("renew") &&
				!viperConfig.GetBool("submit") &&
				!viperConfig.GetBool("approve") &&
				!viperConfig.GetBool("fetch") &&
				!viperConfig.GetBool("delete") {
				glog.Errorf("Must choose at least one flag: --generate, --submit, --approve, --fetch, --delete")
				exitCode = 1
				return
			}
			// build common name and csr name
			commonName := args[0]
			csrName, err := generateCertificateSigningRequestName(commonName)
			if err != nil {
				exitCode = 1
				return
			}

			csrConfig, err := newCSRConfig(commonName, csrName)
			if err != nil {
				exitCode = 1
				return
			}
			var querier *query.Query
			var generator *generate.Generator
			var submitter *submit.Submit
			var approval *approve.Approval
			var fetcher *fetch.Fetch
			var purger *purge.Purge

			svcToQuery := viperConfig.GetStringSlice("query-svc")
			if len(svcToQuery) > 0 {
				querier, err = newQuery(svcToQuery)
				if err != nil {
					exitCode = 1
					return
				}
			}

			if viperConfig.GetBool("generate") {
				generator = generate.NewGenerator(csrConfig)
			}
			if viperConfig.GetBool("submit") || viperConfig.GetBool("renew") {
				submitter, err = newSubmitClient()
				if err != nil {
					exitCode = 1
					return
				}
			}
			if viperConfig.GetBool("approve") {
				approval, err = newApproveClient()
				if err != nil {
					exitCode = 1
					return
				}
			}
			if viperConfig.GetBool("fetch") || viperConfig.GetBool("renew") {
				fetcher, err = newFetchClient()
				if err != nil {
					exitCode = 1
					return
				}
			}
			if viperConfig.GetBool("delete") {
				purger, err = newDeleteClient()
				if err != nil {
					exitCode = 1
					return
				}
			}
			op := operation.NewOperation(
				&operation.Config{
					SourceConfig: csrConfig,
					Query:        querier,
					Generate:     generator,
					Submit:       submitter,
					Approve:      approval,
					Fetch:        fetcher,
					Purge:        purger,
				},
			)
			if !viperConfig.GetBool("renew") {
				err = op.Run()
				if err != nil {
					glog.Errorf("Unexpected error: %v", err)
					exitCode = 2
					return
				}
				return
			}
			re, err := newRenew(op)
			if err != nil {
				exitCode = 1
				return
			}
			err = re.Renew()
			if err != nil {
				glog.Errorf("Unexpected error: %v", err)
				exitCode = 2
				return
			}
		},
	}
	rootCommand.AddCommand(issueCommand)

	// all
	viperConfig.SetDefault("csr-name", "")
	issueCommand.PersistentFlags().String("csr-name", viperConfig.GetString("csr-name"), "Kubernetes CSR name, leave empty for CN-hostname")
	viperConfig.BindPFlag("csr-name", issueCommand.PersistentFlags().Lookup("csr-name"))

	viperConfig.SetDefault("hostname", "")
	issueCommand.PersistentFlags().String("hostname", viperConfig.GetString("hostname"), "Hostname, leave empty to fulfill with hostname")
	viperConfig.BindPFlag("hostname", issueCommand.PersistentFlags().Lookup("hostname"))

	viperConfig.SetDefault("override", false)
	issueCommand.PersistentFlags().Bool("override", viperConfig.GetBool("override"), "Override any existing file pem and k8s csr resource")
	viperConfig.BindPFlag("override", issueCommand.PersistentFlags().Lookup("override"))

	// query
	viperConfig.SetDefault("query-svc", nil)
	issueCommand.PersistentFlags().StringSliceP("query-svc", "q", viperConfig.GetStringSlice("query-svc"), "Query the kube-apiserver services to get additional SAN (namespaceName/serviceName) comma separated")
	viperConfig.BindPFlag("query-svc", issueCommand.PersistentFlags().Lookup("query-svc"))

	viperConfig.SetDefault("query-interval", time.Second*2)
	issueCommand.PersistentFlags().Duration("query-interval", viperConfig.GetDuration("query-interval"), "Polling interval for kube-service query")
	viperConfig.BindPFlag("query-interval", issueCommand.PersistentFlags().Lookup("query-interval"))

	viperConfig.SetDefault("query-timeout", time.Second*20)
	issueCommand.PersistentFlags().Duration("query-timeout", viperConfig.GetDuration("query-timeout"), "Polling timeout for kube-service query")
	viperConfig.BindPFlag("query-timeout", issueCommand.PersistentFlags().Lookup("query-timeout"))

	// generate
	viperConfig.SetDefault("generate", false)
	issueCommand.PersistentFlags().BoolP("generate", "g", viperConfig.GetBool("generate"), "Generate CSR")
	viperConfig.BindPFlag("generate", issueCommand.PersistentFlags().Lookup("generate"))

	viperConfig.SetDefault("rsa-bits", 2048)
	issueCommand.PersistentFlags().String("rsa-bits", viperConfig.GetString("rsa-bits"), "RSA bits for the private key")
	viperConfig.BindPFlag("rsa-bits", issueCommand.PersistentFlags().Lookup("rsa-bits"))

	viperConfig.SetDefault("subject-alternative-names", nil)
	issueCommand.PersistentFlags().StringSlice("subject-alternative-names", viperConfig.GetStringSlice("subject-alternative-names"), "Subject Alternative Names (SANs) comma separated")
	viperConfig.BindPFlag("subject-alternative-names", issueCommand.PersistentFlags().Lookup("subject-alternative-names"))

	// generate - private key
	viperConfig.SetDefault("private-key-perm", 0600)

	viperConfig.SetDefault("private-key-file", "kube-csr.private_key")
	issueCommand.PersistentFlags().String("private-key-file", viperConfig.GetString("private-key-file"), "Private key file target")
	viperConfig.BindPFlag("private-key-file", issueCommand.PersistentFlags().Lookup("private-key-file"))

	viperConfig.SetDefault("load-private-key", false)
	issueCommand.PersistentFlags().Bool("load-private-key", viperConfig.GetBool("load-private-key"), "Load the private key file instead of generating one")
	viperConfig.BindPFlag("load-private-key", issueCommand.PersistentFlags().Lookup("load-private-key"))

	// generate - csr
	viperConfig.SetDefault("csr-perm", 0600)

	viperConfig.SetDefault("csr-file", "kube-csr.csr")
	issueCommand.PersistentFlags().String("csr-file", viperConfig.GetString("csr-file"), "Certificate Signing Request file target")
	viperConfig.BindPFlag("csr-file", issueCommand.PersistentFlags().Lookup("csr-file"))

	// submit
	viperConfig.SetDefault("submit", false)
	issueCommand.PersistentFlags().BoolP("submit", "s", viperConfig.GetBool("submit"), "Submit the CSR")
	viperConfig.BindPFlag("submit", issueCommand.PersistentFlags().Lookup("submit"))

	// approve
	viperConfig.SetDefault("approve", false)
	issueCommand.PersistentFlags().BoolP("approve", "a", viperConfig.GetBool("approve"), "Approve the CSR")
	viperConfig.BindPFlag("approve", issueCommand.PersistentFlags().Lookup("approve"))

	// fetch
	viperConfig.SetDefault("fetch", false)
	viperConfig.SetDefault("certificate-perm", 0600)

	issueCommand.PersistentFlags().BoolP("fetch", "f", viperConfig.GetBool("fetch"), "Fetch the CSR")
	viperConfig.BindPFlag("fetch", issueCommand.PersistentFlags().Lookup("fetch"))

	viperConfig.SetDefault("certificate-file", "kube-csr.certificate")
	issueCommand.PersistentFlags().String("certificate-file", viperConfig.GetString("certificate-file"), "Certificate file target")
	viperConfig.BindPFlag("certificate-file", issueCommand.PersistentFlags().Lookup("certificate-file"))

	viperConfig.SetDefault("fetch-interval", time.Second*1)
	issueCommand.PersistentFlags().Duration("fetch-interval", viperConfig.GetDuration("fetch-interval"), "Polling interval for certificate fetching")
	viperConfig.BindPFlag("fetch-interval", issueCommand.PersistentFlags().Lookup("fetch-interval"))

	viperConfig.SetDefault("fetch-timeout", time.Second*10)
	issueCommand.PersistentFlags().Duration("fetch-timeout", viperConfig.GetDuration("fetch-timeout"), "Polling timeout for certificate fetching")
	viperConfig.BindPFlag("fetch-timeout", issueCommand.PersistentFlags().Lookup("fetch-timeout"))

	viperConfig.SetDefault("skip-fetch-annotate", false)
	issueCommand.PersistentFlags().Bool("skip-fetch-annotate", viperConfig.GetBool("skip-fetch-annotate"), "Skip the update of annotations when successfully fetched the certificate")
	viperConfig.BindPFlag("skip-fetch-annotate", issueCommand.PersistentFlags().Lookup("skip-fetch-annotate"))

	// delete
	viperConfig.SetDefault("delete", false)
	issueCommand.PersistentFlags().BoolP("delete", "d", viperConfig.GetBool("delete"), "Delete the given CSR from the kube-apiserver")
	viperConfig.BindPFlag("delete", issueCommand.PersistentFlags().Lookup("delete"))

	// renew
	viperConfig.SetDefault("renew", false)
	issueCommand.PersistentFlags().Bool("renew", viperConfig.GetBool("renew"), "Renew")
	viperConfig.BindPFlag("renew", issueCommand.PersistentFlags().Lookup("renew"))

	viperConfig.SetDefault("renew-exit", false)
	issueCommand.PersistentFlags().Bool("renew-exit", viperConfig.GetBool("renew-exit"), "Exit 0 after a successful renew")
	viperConfig.BindPFlag("renew-exit", issueCommand.PersistentFlags().Lookup("renew-exit"))

	viperConfig.SetDefault("renew-command", "")
	issueCommand.PersistentFlags().String("renew-command", viperConfig.GetString("renew-command"), "Command to execute after a successful renew (using /bin/sh as interpreter)")
	viperConfig.BindPFlag("renew-command", issueCommand.PersistentFlags().Lookup("renew-command"))

	viperConfig.SetDefault("renew-threshold", time.Hour)
	issueCommand.PersistentFlags().Duration("renew-threshold", viperConfig.GetDuration("renew-threshold"), "Renew expiration threshold")
	viperConfig.BindPFlag("renew-threshold", issueCommand.PersistentFlags().Lookup("renew-threshold"))

	viperConfig.SetDefault("renew-check-interval", time.Minute*15)
	issueCommand.PersistentFlags().Duration("renew-check-interval", viperConfig.GetDuration("renew-check-interval"), "Interval between check of the certificate expiration")
	viperConfig.BindPFlag("renew-check-interval", issueCommand.PersistentFlags().Lookup("renew-check-interval"))

	issueCommand.PersistentFlags().Bool("disable-prometheus-exporter", viperConfig.GetBool("disable-prometheus-exporter"), "disable /metrics, paired with --renew")
	viperConfig.BindPFlag("disable-prometheus-exporter", garbageCommand.PersistentFlags().Lookup("disable-prometheus-exporter"))

	issueCommand.PersistentFlags().Bool("prometheus-exporter-bind", viperConfig.GetBool("prometheus-exporter-bind"), "prometheus exporter bind address, paired with --renew")
	viperConfig.BindPFlag("prometheus-exporter-bind", garbageCommand.PersistentFlags().Lookup("prometheus-exporter-bind"))
	return rootCommand, &exitCode
}

func generateCertificateSigningRequestName(commonName string) (string, error) {
	csrName := viperConfig.GetString("csr-name")
	if csrName != "" {
		return csrName, nil
	}

	hostname := viperConfig.GetString("hostname")
	if hostname != "" {
		return fmt.Sprintf("%s-%s", commonName, hostname), nil
	}

	hostname, err := os.Hostname()
	if err != nil {
		glog.Errorf("Cannot get hostname: %v", err)
		return "", err
	}
	return fmt.Sprintf("%s-%s", commonName, hostname), nil
}

func newCSRConfig(commonName, csrName string) (*generate.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		glog.Errorf("Unexpected error: %v", err)
		return nil, err
	}

	privateKeyPath := viperConfig.GetString("private-key-file")
	if !path.IsAbs(privateKeyPath) {
		privateKeyPath = path.Join(wd, privateKeyPath)
	}

	csrPath := viperConfig.GetString("csr-file")
	if !path.IsAbs(csrPath) {
		csrPath = path.Join(wd, csrPath)
	}
	return &generate.Config{
		Name:       csrName,
		Override:   viperConfig.GetBool("override"),
		CommonName: commonName,
		Hosts:      viperConfig.GetStringSlice("subject-alternative-names"),
		RSABits:    viperConfig.GetInt("rsa-bits"),

		LoadPrivateKey:       viperConfig.GetBool("load-private-key"),
		PrivateKeyABSPath:    privateKeyPath,
		PrivateKeyPermission: os.FileMode(viperConfig.GetInt("private-key-perm")),

		CSRABSPath:    csrPath,
		CSRPermission: os.FileMode(viperConfig.GetInt("csr-perm")),
	}, nil
}

func newSubmitClient() (*submit.Submit, error) {
	s, err := submit.NewSubmitter(
		viperConfig.GetString("kubeconfig-path"),
		&submit.Config{
			Override: viperConfig.GetBool("override"),
		},
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func newApproveClient() (*approve.Approval, error) {
	s, err := approve.NewApproval(viperConfig.GetString("kubeconfig-path"))
	if err != nil {
		return nil, err
	}
	return s, nil
}

func newFetchClient() (*fetch.Fetch, error) {
	wd, err := os.Getwd()
	if err != nil {
		glog.Errorf("Unexpected error: %v", err)
		return nil, err
	}

	crtPath := viperConfig.GetString("certificate-file")
	if !path.IsAbs(crtPath) {
		crtPath = path.Join(wd, crtPath)
	}

	annotate := !viperConfig.GetBool("skip-fetch-annotate")
	if annotate && viperConfig.GetBool("delete") {
		// useless to annotate a kube resource just deleted after
		glog.V(0).Infof("As configured with delete, ignoring the annotation operations during the fetch")
		annotate = false
	}
	conf := &fetch.Config{
		Override:              viperConfig.GetBool("override"),
		PollingInterval:       viperConfig.GetDuration("fetch-interval"),
		PollingTimeout:        viperConfig.GetDuration("fetch-timeout"),
		CertificatePermission: os.FileMode(viperConfig.GetInt("certificate-perm")),
		CertificateABSPath:    crtPath,
		Annotate:              annotate,
	}
	f, err := fetch.NewFetcher(viperConfig.GetString("kubeconfig-path"), conf)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func newDeleteClient() (*purge.Purge, error) {
	s, err := purge.NewPurge(viperConfig.GetString("kubeconfig-path"), nil)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func newGarbageCollector() (*purge.Purge, error) {
	conf := purge.NewPurgeConfig(viperConfig.GetDuration("grace-period"))
	if viperConfig.GetBool("denied") {
		conf.ShouldGC = append(conf.ShouldGC, purge.IsConditionDenied)
	}
	if viperConfig.GetBool("fetched") {
		conf.ShouldGC = append(conf.ShouldGC, purge.IsAnnotationFetched)
	}
	if viperConfig.GetBool("expired") {
		conf.ShouldGC = append(conf.ShouldGC, purge.IsCertificateExpired)
	}
	conf.PollingPeriod = viperConfig.GetDuration("polling-period")
	if !viperConfig.GetBool("disable-prometheus-exporter") {
		conf.PrometheusExporterBindAddress = viperConfig.GetString("prometheus-exporter-bind")
	}
	p, err := purge.NewPurge(viperConfig.GetString("kubeconfig-path"), conf)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func newQuery(svcToQuery []string) (*query.Query, error) {
	q, err := query.NewQuery(viperConfig.GetString("kubeconfig-path"), svcToQuery, &query.Config{
		PollingTimeout:  viperConfig.GetDuration("query-timeout"),
		PollingInterval: viperConfig.GetDuration("query-interval"),
	})
	if err != nil {
		return nil, err
	}
	return q, nil
}

func newRenew(operation *operation.Operation) (*renew.Renew, error) {
	conf := &renew.Config{
		Operation:                operation,
		RenewThreshold:           viperConfig.GetDuration("renew-threshold"),
		ExitOnRenew:              viperConfig.GetBool("renew-exit"),
		GenerateNewKubernetesCSR: !viperConfig.GetBool("override"),
		RenewCommand:             viperConfig.GetString("renew-command"),
		RenewCheckInterval:       viperConfig.GetDuration("renew-check-interval"),
	}
	if !viperConfig.GetBool("disable-prometheus-exporter") {
		conf.PrometheusExporterBindAddress = viperConfig.GetString("prometheus-exporter-bind")
	}
	return renew.NewRenewer(viperConfig.GetString("kubeconfig-path"), conf)
}
