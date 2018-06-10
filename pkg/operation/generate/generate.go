package generate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/golang/glog"

	"github.com/JulienBalestra/kube-csr/pkg/utils/pemio"
)

const (
	rsaPrivateKeyType = "RSA PRIVATE KEY"
	csrType           = "CERTIFICATE REQUEST"
)

// Config of Generator
type Config struct {
	Name     string
	Override bool

	CommonName string `json:"common-name"`
	Hosts      []string

	RSABits              int
	PrivateKeyABSPath    string
	PrivateKeyPermission os.FileMode
	CSRABSPath           string
	CSRPermission        os.FileMode
}

// Generator state
type Generator struct {
	conf *Config
}

// NewGenerator creates a new Generator
func NewGenerator(conf *Config) *Generator {
	return &Generator{
		conf: conf,
	}
}

func (g *Generator) categorizeHosts() ([]string, []net.IP, error) {
	var dnsNames []string
	var ipAddresses []net.IP
	var invalidHosts []string

	for _, host := range g.conf.Hosts {
		ip := net.ParseIP(host)
		if ip != nil {
			ipAddresses = append(ipAddresses, ip)
			glog.V(0).Infof("Added IP address %s", ip.String())
			continue
		}
		if strings.ContainsRune(host, rune('.')) {
			dnsNames = append(dnsNames, host)
			glog.V(0).Infof("Added DNS name %s", host)
			continue
		}
		glog.Errorf("Invalid entry: host %q is neither IP address nor DNS name", host)
		invalidHosts = append(invalidHosts, host)
	}
	if len(invalidHosts) > 0 {
		return nil, nil, fmt.Errorf("cannot categorize given hosts: %s", strings.Join(invalidHosts, ", "))
	}
	glog.V(0).Infof("CSR with %d DNS names and %d IP addresses", len(dnsNames), len(ipAddresses))
	return dnsNames, ipAddresses, nil
}

func (g *Generator) generateCryptoData() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, g.conf.RSABits)
	if err != nil {
		glog.Errorf("Unexpected error during the RSA Key generation: %v", err)
		return nil, nil, err
	}
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)

	if g.conf.CommonName == "" {
		glog.Errorf("Invalid empty CommonName")
		return nil, nil, fmt.Errorf("empty CommonName")
	}

	dnsNames, ipAddresses, err := g.categorizeHosts()
	if err != nil {
		return nil, nil, err
	}
	glog.V(0).Infof("Generating CSR with CN=%s", g.conf.CommonName)
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: g.conf.CommonName,
		},
		SignatureAlgorithm: x509.SHA256WithRSA,
		DNSNames:           dnsNames,
		IPAddresses:        ipAddresses,
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		glog.Errorf("Unexpected error during the CSR: %v", err)
		return nil, nil, err
	}
	return privKeyBytes, csrBytes, nil
}

// Generate the given CSR
func (g *Generator) Generate() error {
	// crypto data
	privKeyBytes, csrBytes, err := g.generateCryptoData()
	if err != nil {
		glog.Errorf("Cannot generate crypto data: %v", err)
		return err
	}

	// write to FS
	err = pemio.WritePem(privKeyBytes, rsaPrivateKeyType, g.conf.PrivateKeyABSPath, g.conf.PrivateKeyPermission, g.conf.Override)
	if err != nil {
		return err
	}
	err = pemio.WritePem(csrBytes, csrType, g.conf.CSRABSPath, g.conf.CSRPermission, g.conf.Override)
	if err != nil {
		return err
	}

	return nil
}
