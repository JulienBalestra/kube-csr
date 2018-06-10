package main

import (
	"flag"
	"os"
	"path"
	"time"

	"github.com/JulienBalestra/kube-csr/pkg/operation"
	"github.com/JulienBalestra/kube-csr/pkg/operation/approve"
	"github.com/JulienBalestra/kube-csr/pkg/operation/fetch"
	"github.com/JulienBalestra/kube-csr/pkg/operation/generate"
	"github.com/JulienBalestra/kube-csr/pkg/operation/purge"
	"github.com/JulienBalestra/kube-csr/pkg/operation/submit"
)

func main() {
	// glog section - optional
	flag.Parse()
	flag.Lookup("alsologtostderr").Value.Set("true")
	flag.Lookup("v").Value.Set("2")

	kubeConfigPath := path.Join("/home", os.Getenv("USER"), ".kube/config")
	//kubeConfigPath := "" empty string to mark as inCluster

	csrConfig := &generate.Config{
		Name:                 "foo",
		CommonName:           "example",
		Hosts:                []string{"example.com", "192.168.1.1"},
		RSABits:              2048,
		PrivateKeyABSPath:    "/tmp/foo.private_key",
		PrivateKeyPermission: 0600,
		CSRABSPath:           "/tmp/foo.csr",
		CSRPermission:        0600,
		Override:             true,
	}
	generator := generate.NewGenerator(csrConfig)
	submitter, err := submit.NewSubmitter(kubeConfigPath, &submit.Config{
		Override: true,
	})
	if err != nil {
		panic(err)
	}
	approval, err := approve.NewApproval(kubeConfigPath)
	if err != nil {
		panic(err)
	}
	fetcher, err := fetch.NewFetcher(kubeConfigPath, &fetch.Config{
		PollingTimeout:        time.Second * 10,
		PollingInterval:       time.Second * 1,
		CertificateABSPath:    "/tmp/foo.certificate",
		CertificatePermission: 0600,
	})
	if err != nil {
		panic(err)
	}
	purger, err := purge.NewPurge(kubeConfigPath)
	if err != nil {
		panic(err)
	}
	err = operation.NewOperation(&operation.Config{
		SourceConfig: csrConfig,
		Generate:     generator,
		Submit:       submitter,
		Approve:      approval,
		Fetch:        fetcher,
		Purge:        purger,
	}).Run()
	if err != nil {
		panic(err)
	}
}
