package pemio

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestWriteFile(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "kube-csr-tests-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = WriteFile([]byte("123"), path.Join(tempDir, "a"), 0600, false)
	assert.NoError(t, err)
	err = WriteFile([]byte("123"), path.Join(tempDir, "a"), 0600, false)
	require.Error(t, err)
	assert.Equal(t, err.Error(), fmt.Sprintf("file exists %s/a", tempDir))
	err = WriteFile([]byte("123"), path.Join(tempDir, "a"), 0600, true)
	assert.NoError(t, err)
}

func TestWritePem(t *testing.T) {
	tempDir, err := ioutil.TempDir(os.TempDir(), "kube-csr-tests-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	err = WritePem([]byte("123"), "CERTIFICATE", path.Join(tempDir, "a"), 0600, false)
	assert.NoError(t, err)
	err = WritePem([]byte("123"), "CERTIFICATE", path.Join(tempDir, "a"), 0600, false)
	require.Error(t, err)
	assert.Equal(t, err.Error(), fmt.Sprintf("file exists %s/a", tempDir))
	err = WritePem([]byte("123"), "CERTIFICATE", path.Join(tempDir, "a"), 0600, true)
	assert.NoError(t, err)
}
