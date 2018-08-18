package generate

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerator(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "kube-csr-tests-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	for _, tc := range []struct {
		conf        *Config
		expectedErr string
	}{
		{
			conf: &Config{
				Name:                 "test-1",
				Override:             false,
				CommonName:           "cn-1",
				Hosts:                nil,
				RSABits:              1024,
				PrivateKeyABSPath:    path.Join(tempDir, "1.private_key"),
				PrivateKeyPermission: 0600,
				CSRABSPath:           path.Join(tempDir, "1.csr"),
				CSRPermission:        0600,
			},
			expectedErr: "",
		},
		{
			conf: &Config{
				Name:                 "test-1",
				Override:             false,
				CommonName:           "cn-1",
				Hosts:                nil,
				RSABits:              1024,
				PrivateKeyABSPath:    path.Join(tempDir, "1.private_key"),
				PrivateKeyPermission: 0600,
				CSRABSPath:           path.Join(tempDir, "1.csr"),
				CSRPermission:        0600,
			},
			expectedErr: fmt.Sprintf("file exists %s/1.csr", tempDir),
		},
		{
			conf: &Config{
				Name:                 "test-1",
				Override:             true,
				CommonName:           "cn-1",
				Hosts:                nil,
				RSABits:              1024,
				PrivateKeyABSPath:    path.Join(tempDir, "1.private_key"),
				PrivateKeyPermission: 0600,
				CSRABSPath:           path.Join(tempDir, "1.csr"),
				CSRPermission:        0600,
			},
			expectedErr: "",
		},
		{
			conf: &Config{
				Name:       "test-2",
				Override:   false,
				CommonName: "cn-2",
				Hosts: []string{
					"example.com",
				},
				RSABits:              1024,
				PrivateKeyABSPath:    path.Join(tempDir, "2.private_key"),
				PrivateKeyPermission: 0600,
				CSRABSPath:           path.Join(tempDir, "2.csr"),
				CSRPermission:        0600,
			},
			expectedErr: "",
		},
		{
			conf: &Config{
				Name:       "test-3",
				Override:   false,
				CommonName: "cn-3",
				Hosts: []string{
					"192.168.1.1",
				},
				RSABits:              1024,
				PrivateKeyABSPath:    path.Join(tempDir, "3.private_key"),
				PrivateKeyPermission: 0600,
				CSRABSPath:           path.Join(tempDir, "3.csr"),
				CSRPermission:        0600,
			},
			expectedErr: "",
		},
		{
			conf: &Config{
				Name:       "test-4",
				Override:   false,
				CommonName: "cn-4",
				Hosts: []string{
					"example.com",
					"192.168.1.1",
				},
				RSABits:              1024,
				PrivateKeyABSPath:    path.Join(tempDir, "4.private_key"),
				PrivateKeyPermission: 0600,
				CSRABSPath:           path.Join(tempDir, "4.csr"),
				CSRPermission:        0600,
			},
			expectedErr: "",
		},
		{
			conf: &Config{
				Name:       "test-5",
				Override:   true,
				CommonName: "cn-5",
				Hosts: []string{
					"example.com",
					"192.168.1.1",
				},
				RSABits:              1024,
				LoadPrivateKey:       true,
				PrivateKeyABSPath:    path.Join(tempDir, "5.private_key"),
				PrivateKeyPermission: 0600,
				CSRABSPath:           path.Join(tempDir, "5.csr"),
				CSRPermission:        0600,
			},
			expectedErr: fmt.Sprintf("open %s/5.private_key: no such file or directory", tempDir),
		},
	} {
		t.Run(tc.conf.Name, func(t *testing.T) {
			g := NewGenerator(tc.conf)
			err := g.Generate()
			if tc.expectedErr == "" && err != nil {
				t.Errorf(err.Error())
			}
			if tc.expectedErr != "" {
				assert.Equal(t, tc.expectedErr, err.Error())
			}
		})
	}
}

func TestCategorizeHosts(t *testing.T) {
	for _, tc := range []struct {
		conf         *Config
		expectedFQDN []string
		expectedIPs  []net.IP
	}{
		{
			conf: &Config{
				Hosts: []string{},
			},
		},
		{
			conf: &Config{
				Hosts: []string{
					"example.com",
				},
			},
			expectedFQDN: []string{
				"example.com",
			},
		},
		{
			conf: &Config{
				Hosts: []string{
					"192.168.1.1",
				},
			},
			expectedIPs: []net.IP{
				net.ParseIP("192.168.1.1"),
			},
		},
		{
			conf: &Config{
				Hosts: []string{
					"192.168.1.1",
					"example.com",
				},
			},
			expectedFQDN: []string{
				"example.com",
			},
			expectedIPs: []net.IP{
				net.ParseIP("192.168.1.1"),
			},
		},
	} {
		t.Run("", func(t *testing.T) {
			g := NewGenerator(tc.conf)
			fqdns, ips, err := g.categorizeHosts()
			assert.Equal(t, nil, err)
			assert.Equal(t, tc.expectedFQDN, fqdns)
			assert.Equal(t, tc.expectedIPs, ips)
		})
	}
}
