package cli

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/JulienBalestra/kube-csr/pkg/operation"
	"github.com/JulienBalestra/kube-csr/pkg/operation/approve"
	"github.com/JulienBalestra/kube-csr/pkg/operation/fetch"
	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/operation/submit"
)

const programName = "kube-csr"

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
%s my-app --generate --submit --fetch --fetch-timeout 10m

# Generate the private key, the csr, submit, approve and fetch the csr. Override any existing and use a kubeconfig
%s my-app -gsaf --override --kubeconfig-path ~/.kube/config
`, programName, programName, programName, programName, programName, programName, programName, programName),
		Run: func(cmd *cobra.Command, args []string) {
			if !viperConfig.GetBool("generate") &&
				!viperConfig.GetBool("submit") &&
				!viperConfig.GetBool("approve") &&
				!viperConfig.GetBool("fetch") {
				glog.Errorf("Must choose at least one flag: --generate, --submit, --approve, --fetch")
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
			err = operation.NewOperation(
				&operation.Config{
					SourceConfig: csrConfig,
					Generate:     generator,
					Submit:       submitter,
					Approve:      approval,
					Fetch:        fetcher,
				},
			).Run()
			if err != nil {
				exitCode = 2
				return
			}
		},
	}

	rootCommand.PersistentFlags().IntVarP(&verbose, "verbose", "v", 2, "verbose level")

	// all
	rootCommand.PersistentFlags().String("csr-name", viperConfig.GetString("csr-name"), "Kubernetes CSR name, leave empty for CN/hostname")
	viperConfig.BindPFlag("csr-name", rootCommand.PersistentFlags().Lookup("csr-name"))

	rootCommand.PersistentFlags().String("hostname", viperConfig.GetString("hostname"), "Hostname, leave empty to fulfill with hostname")
	viperConfig.BindPFlag("hostname", rootCommand.PersistentFlags().Lookup("hostname"))

	rootCommand.PersistentFlags().Bool("override", viperConfig.GetBool("override"), "Override any existing file pem and k8s csr resource")
	viperConfig.BindPFlag("override", rootCommand.PersistentFlags().Lookup("override"))

	// generate
	rootCommand.PersistentFlags().BoolP("generate", "g", viperConfig.GetBool("generate"), "Generate CSR")
	viperConfig.BindPFlag("generate", rootCommand.PersistentFlags().Lookup("generate"))

	rootCommand.PersistentFlags().String("rsa-bits", viperConfig.GetString("rsa-bits"), "RSA bits for the private key")
	viperConfig.BindPFlag("rsa-bits", rootCommand.PersistentFlags().Lookup("rsa-bits"))

	rootCommand.PersistentFlags().StringSlice("subject-alternative-names", viperConfig.GetStringSlice("subject-alternative-names"), "Subject Alternative Names (SANs) comma separated")
	viperConfig.BindPFlag("subject-alternative-names", rootCommand.PersistentFlags().Lookup("subject-alternative-names"))

	// generate - private key
	rootCommand.PersistentFlags().String("private-key-file", viperConfig.GetString("private-key-file"), "Private key file target")
	viperConfig.BindPFlag("private-key-file", rootCommand.PersistentFlags().Lookup("private-key-file"))

	// generate - csr
	rootCommand.PersistentFlags().String("csr-file", viperConfig.GetString("csr-file"), "Certificate Signing Request file target")
	viperConfig.BindPFlag("csr-file", rootCommand.PersistentFlags().Lookup("csr-file"))

	// approve & fetch
	rootCommand.PersistentFlags().String("kubeconfig-path", viperConfig.GetString("kubeconfig-path"), "Kubernetes config path, leave empty for inCluster config")
	viperConfig.BindPFlag("kubeconfig-path", rootCommand.PersistentFlags().Lookup("kubeconfig-path"))

	// approve
	rootCommand.PersistentFlags().BoolP("submit", "s", viperConfig.GetBool("submit"), "Submit the CSR")
	viperConfig.BindPFlag("submit", rootCommand.PersistentFlags().Lookup("submit"))
	rootCommand.PersistentFlags().BoolP("approve", "a", viperConfig.GetBool("approve"), "Approve the CSR")
	viperConfig.BindPFlag("approve", rootCommand.PersistentFlags().Lookup("approve"))

	// fetch
	rootCommand.PersistentFlags().BoolP("fetch", "f", viperConfig.GetBool("fetch"), "Fetch the CSR")
	viperConfig.BindPFlag("fetch", rootCommand.PersistentFlags().Lookup("fetch"))

	rootCommand.PersistentFlags().String("certificate-file", viperConfig.GetString("certificate-file"), "Certificate file target")
	viperConfig.BindPFlag("certificate-file", rootCommand.PersistentFlags().Lookup("certificate-file"))

	rootCommand.PersistentFlags().Duration("fetch-interval", viperConfig.GetDuration("fetch-interval"), "Polling interval for certificate fetching")
	viperConfig.BindPFlag("fetch-interval", rootCommand.PersistentFlags().Lookup("fetch-interval"))

	rootCommand.PersistentFlags().Duration("fetch-timeout", viperConfig.GetDuration("fetch-timeout"), "Polling timeout for certificate fetching")
	viperConfig.BindPFlag("fetch-timeout", rootCommand.PersistentFlags().Lookup("fetch-timeout"))

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

	fetchInterval := viperConfig.GetDuration("fetch-interval")
	timeoutInterval := viperConfig.GetDuration("fetch-timeout")
	if !viperConfig.GetBool("approve") {
		fetchInterval = defaultFetchInterval * 10
		timeoutInterval = defaultTimeoutInterval * 10
		glog.V(2).Infof("csr externally approved, setting the polling interval to %s and the timeout to %s", fetchInterval.String(), timeoutInterval.String())
	}

	conf := &fetch.Config{
		Override:              viperConfig.GetBool("override"),
		PollingInterval:       fetchInterval,
		PollingTimeout:        timeoutInterval,
		CertificatePermission: os.FileMode(viperConfig.GetInt("certificate-perm")),
		CertificateABSPath:    crtPath,
	}
	f, err := fetch.NewFetcher(viperConfig.GetString("kubeconfig-path"), conf)
	if err != nil {
		return nil, err
	}
	return f, nil
}
