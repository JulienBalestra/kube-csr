package cli

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
	"github.com/JulienBalestra/kube-csr/pkg/operation/submit"
)

const (
	programName = "kube-csr"
)

var viperConfig = viper.New()

// NewCommand creates a cobra command to be consumed in the main package
func NewCommand() (*cobra.Command, *int) {
	var verbose int
	var exitCode int

	rootCommand := &cobra.Command{
		Use:   fmt.Sprintf("%s command line", programName),
		Short: "Use this command to generate, approve and fetch Kubernetes certificates",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			flag.Lookup("alsologtostderr").Value.Set("true")
			flag.Lookup("v").Value.Set(strconv.Itoa(verbose))
		},
		Args: cobra.ExactArgs(1),
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
%s skydns --csr-name kv-etcd -gsafp --override --kubeconfig-path ~/.kube/config
`, programName, programName, programName, programName, programName, programName, programName, programName, programName, programName),
		Run: func(cmd *cobra.Command, args []string) {
			if !viperConfig.GetBool("generate") &&
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
			var generator *generate.Generator
			var submitter *submit.Submit
			var approval *approve.Approval
			var fetcher *fetch.Fetch
			var purger *purge.Purge

			if viperConfig.GetBool("generate") {
				generator = generate.NewGenerator(csrConfig)
			}
			if viperConfig.GetBool("submit") {
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
			if viperConfig.GetBool("fetch") {
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
			err = operation.NewOperation(
				&operation.Config{
					SourceConfig: csrConfig,
					Generate:     generator,
					Submit:       submitter,
					Approve:      approval,
					Fetch:        fetcher,
					Purge:        purger,
				},
			).Run()
			if err != nil {
				exitCode = 2
				return
			}
		},
	}

	rootCommand.PersistentFlags().IntVarP(&verbose, "verbose", "v", 0, "verbose level")

	// all
	viperConfig.SetDefault("csr-name", "")
	rootCommand.PersistentFlags().String("csr-name", viperConfig.GetString("csr-name"), "Kubernetes CSR name, leave empty for CN-hostname")
	viperConfig.BindPFlag("csr-name", rootCommand.PersistentFlags().Lookup("csr-name"))

	viperConfig.SetDefault("hostname", "")
	rootCommand.PersistentFlags().String("hostname", viperConfig.GetString("hostname"), "Hostname, leave empty to fulfill with hostname")
	viperConfig.BindPFlag("hostname", rootCommand.PersistentFlags().Lookup("hostname"))

	viperConfig.SetDefault("override", false)
	rootCommand.PersistentFlags().Bool("override", viperConfig.GetBool("override"), "Override any existing file pem and k8s csr resource")
	viperConfig.BindPFlag("override", rootCommand.PersistentFlags().Lookup("override"))

	// needed by submit, approve, fetch, delete
	viperConfig.SetDefault("kubeconfig-path", "")
	rootCommand.PersistentFlags().String("kubeconfig-path", viperConfig.GetString("kubeconfig-path"), "Kubernetes config path, leave empty for inCluster config")
	viperConfig.BindPFlag("kubeconfig-path", rootCommand.PersistentFlags().Lookup("kubeconfig-path"))

	// generate
	viperConfig.SetDefault("generate", false)
	rootCommand.PersistentFlags().BoolP("generate", "g", viperConfig.GetBool("generate"), "Generate CSR")
	viperConfig.BindPFlag("generate", rootCommand.PersistentFlags().Lookup("generate"))

	viperConfig.SetDefault("rsa-bits", 2048)
	rootCommand.PersistentFlags().String("rsa-bits", viperConfig.GetString("rsa-bits"), "RSA bits for the private key")
	viperConfig.BindPFlag("rsa-bits", rootCommand.PersistentFlags().Lookup("rsa-bits"))

	viperConfig.SetDefault("subject-alternative-names", nil)
	rootCommand.PersistentFlags().StringSlice("subject-alternative-names", viperConfig.GetStringSlice("subject-alternative-names"), "Subject Alternative Names (SANs) comma separated")
	viperConfig.BindPFlag("subject-alternative-names", rootCommand.PersistentFlags().Lookup("subject-alternative-names"))

	// generate - private key
	viperConfig.SetDefault("private-key-perm", 0600)

	viperConfig.SetDefault("private-key-file", "kube-csr.private_key")
	rootCommand.PersistentFlags().String("private-key-file", viperConfig.GetString("private-key-file"), "Private key file target")
	viperConfig.BindPFlag("private-key-file", rootCommand.PersistentFlags().Lookup("private-key-file"))

	// generate - csr
	viperConfig.SetDefault("csr-perm", 0600)

	viperConfig.SetDefault("csr-file", "kube-csr.csr")
	rootCommand.PersistentFlags().String("csr-file", viperConfig.GetString("csr-file"), "Certificate Signing Request file target")
	viperConfig.BindPFlag("csr-file", rootCommand.PersistentFlags().Lookup("csr-file"))

	// submit
	viperConfig.SetDefault("submit", false)
	rootCommand.PersistentFlags().BoolP("submit", "s", viperConfig.GetBool("submit"), "Submit the CSR")
	viperConfig.BindPFlag("submit", rootCommand.PersistentFlags().Lookup("submit"))

	// approve
	viperConfig.SetDefault("approve", false)
	rootCommand.PersistentFlags().BoolP("approve", "a", viperConfig.GetBool("approve"), "Approve the CSR")
	viperConfig.BindPFlag("approve", rootCommand.PersistentFlags().Lookup("approve"))

	// fetch
	viperConfig.SetDefault("fetch", false)
	viperConfig.SetDefault("certificate-perm", 0600)

	rootCommand.PersistentFlags().BoolP("fetch", "f", viperConfig.GetBool("fetch"), "Fetch the CSR")
	viperConfig.BindPFlag("fetch", rootCommand.PersistentFlags().Lookup("fetch"))

	viperConfig.SetDefault("certificate-file", "kube-csr.certificate")
	rootCommand.PersistentFlags().String("certificate-file", viperConfig.GetString("certificate-file"), "Certificate file target")
	viperConfig.BindPFlag("certificate-file", rootCommand.PersistentFlags().Lookup("certificate-file"))

	viperConfig.SetDefault("fetch-interval", time.Second*1)
	rootCommand.PersistentFlags().Duration("fetch-interval", viperConfig.GetDuration("fetch-interval"), "Polling interval for certificate fetching")
	viperConfig.BindPFlag("fetch-interval", rootCommand.PersistentFlags().Lookup("fetch-interval"))

	viperConfig.SetDefault("fetch-timeout", time.Second*10)
	rootCommand.PersistentFlags().Duration("fetch-timeout", viperConfig.GetDuration("fetch-timeout"), "Polling timeout for certificate fetching")
	viperConfig.BindPFlag("fetch-timeout", rootCommand.PersistentFlags().Lookup("fetch-timeout"))

	viperConfig.SetDefault("skip-fetch-annotate", false)
	rootCommand.PersistentFlags().Bool("skip-fetch-annotate", viperConfig.GetBool("skip-fetch-annotate"), "Skip the update of annotations when successfully fetched the certificate")
	viperConfig.BindPFlag("skip-fetch-annotate", rootCommand.PersistentFlags().Lookup("skip-fetch-annotate"))

	// delete
	viperConfig.SetDefault("delete", false)
	rootCommand.PersistentFlags().BoolP("delete", "d", viperConfig.GetBool("delete"), "Delete the given CSR from the kube-apiserver")
	viperConfig.BindPFlag("delete", rootCommand.PersistentFlags().Lookup("delete"))

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
