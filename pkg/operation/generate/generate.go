package generate

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/golang/glog"
	"net"
	"os"
	"sort"

	"encoding/pem"
	"github.com/JulienBalestra/kube-csr/pkg/utils/pemio"
	"io/ioutil"
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

	LoadPrivateKey       bool
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
	hostSet := make(map[string]struct{}, len(g.conf.Hosts))

	for _, host := range g.conf.Hosts {
		_, ok := hostSet[host]
		if ok {
			glog.V(0).Infof("Host %s already added, skipping", host)
			continue
		}
		hostSet[host] = struct{}{}
		ip := net.ParseIP(host)
		if ip != nil {
			ipAddresses = append(ipAddresses, ip)
			glog.V(0).Infof("Added IP address %s", ip.String())
			continue
		}
		dnsNames = append(dnsNames, host)
		glog.V(0).Infof("Added DNS name %s", host)
	}

	// sort to get a stable result
	sort.Strings(dnsNames)
	sort.Slice(ipAddresses, func(i, j int) bool {
		return ipAddresses[i].String() < ipAddresses[i].String()
	})

	glog.V(0).Infof("CSR with %d DNS names and %d IP addresses", len(dnsNames), len(ipAddresses))
	return dnsNames, ipAddresses, nil
}

func (g *Generator) generateCryptoData() ([]byte, []byte, error) {
	var privateKey *rsa.PrivateKey
	var err error

	if g.conf.LoadPrivateKey {
		glog.V(0).Infof("Loading private key %v", g.conf.PrivateKeyABSPath)
		b, err := ioutil.ReadFile(g.conf.PrivateKeyABSPath)
		if err != nil {
			glog.Errorf("Cannot load existing private key: %v", err)
			return nil, nil, err
		}
		p, _ := pem.Decode(b)
		if p == nil {
			err = fmt.Errorf("cannot decode private key %s", g.conf.PrivateKeyABSPath)
			glog.Errorf("Unexpected error: %v", err)
			return nil, nil, err
		}
		privateKey, err = x509.ParsePKCS1PrivateKey(p.Bytes)
		if err != nil {
			glog.Errorf("Cannot parse the given private key %s: %v", g.conf.PrivateKeyABSPath, err)
			return nil, nil, err
		}

	} else {
		glog.V(0).Infof("Generating private key")
		privateKey, err = rsa.GenerateKey(rand.Reader, g.conf.RSABits)
		if err != nil {
			glog.Errorf("Unexpected error during the RSA Key generation: %v", err)
			return nil, nil, err
		}
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
	err = pemio.WritePem(csrBytes, csrType, g.conf.CSRABSPath, g.conf.CSRPermission, g.conf.Override)
	if err != nil {
		return err
	}
	if g.conf.LoadPrivateKey {
		return nil
	}
	return pemio.WritePem(privKeyBytes, rsaPrivateKeyType, g.conf.PrivateKeyABSPath, g.conf.PrivateKeyPermission, g.conf.Override)
}
