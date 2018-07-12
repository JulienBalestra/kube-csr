package purge

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	certificates "k8s.io/api/certificates/v1beta1"
)

func TestDurationFormat(t *testing.T) {
	for _, tc := range []struct {
		d      time.Duration
		output string
	}{
		{
			d:      5 * time.Hour,
			output: "5h0m0s",
		},
		{
			d:      25 * time.Hour,
			output: "1 days and 1h0m0s",
		},
		{
			d:      49 * time.Hour,
			output: "2 days and 1h0m0s",
		},
		{
			d:      31 * 24 * time.Hour,
			output: "31 days and 0s",
		},
		{
			d:      365 * 24 * time.Hour,
			output: "365 days and 0s",
		},
	} {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tc.output, durationFormat(tc.d))
		})
	}
}

func generateCertOrDie(notAfter time.Time) []byte {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour * 24 * 31),
		NotAfter:     notAfter,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, privateKey.Public(), privateKey)
	if err != nil {
		panic(err)
	}
	out := &bytes.Buffer{}
	err = pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	if err != nil {
		panic(err)
	}
	return out.Bytes()
}

func TestIsCertificateExpired(t *testing.T) {
	for _, tc := range []struct {
		csr         *certificates.CertificateSigningRequest
		gracePeriod time.Duration
		expired     bool
	}{
		{
			csr: &certificates.CertificateSigningRequest{
				Status: certificates.CertificateSigningRequestStatus{
					Certificate: generateCertOrDie(time.Now().Add(-time.Hour)),
				},
			},
			gracePeriod: time.Minute * 45,
			expired:     true,
		},
		{
			csr: &certificates.CertificateSigningRequest{
				Status: certificates.CertificateSigningRequestStatus{
					Certificate: generateCertOrDie(time.Now().Add(-time.Hour)),
				},
			},
			gracePeriod: time.Minute * 61,
			expired:     false,
		},
		{
			csr: &certificates.CertificateSigningRequest{
				Status: certificates.CertificateSigningRequestStatus{
					Certificate: generateCertOrDie(time.Now().Add(time.Hour)),
				},
			},
			gracePeriod: time.Minute * 1,
			expired:     false,
		},
	} {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tc.expired, IsCertificateExpired(tc.csr, tc.gracePeriod))
		})
	}
}
