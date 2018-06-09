package cli

import (
	"github.com/spf13/viper"
	"time"
)

const (
	defaultFetchInterval   = time.Second * 1
	defaultTimeoutInterval = defaultFetchInterval * 10
)

var viperConfig = viper.New()

func init() {
	// all
	viperConfig.SetDefault("csr-name", "")
	viperConfig.SetDefault("hostname", "")
	viperConfig.SetDefault("override", false)

	// generate
	viperConfig.SetDefault("generate", false)
	viperConfig.SetDefault("rsa-bits", 2048)
	viperConfig.SetDefault("subject-alternative-names", nil)
	viperConfig.SetDefault("private-key-file", "kube-csr.private_key")
	viperConfig.SetDefault("private-key-perm", 0600)
	viperConfig.SetDefault("csr-file", "kube-csr.csr")

	viperConfig.SetDefault("csr-perm", 0600)
	// approve & fetch
	viperConfig.SetDefault("kubeconfig-path", "")

	// submit - approve
	viperConfig.SetDefault("submit", false)
	viperConfig.SetDefault("approve", false)

	// fetch
	viperConfig.SetDefault("fetch", false)
	viperConfig.SetDefault("certificate-file", "kube-csr.certificate")
	viperConfig.SetDefault("certificate-perm", 0600)
	viperConfig.SetDefault("fetch-interval", defaultFetchInterval)
	viperConfig.SetDefault("fetch-timeout", defaultTimeoutInterval)
	viperConfig.SetDefault("skip-fetch-annotate", false)
}
