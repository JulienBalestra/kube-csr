package pemio

import (
	"bufio"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/golang/glog"
)

// WriteFile write the passed bytes to the created file. Override allows to truncate the existing file
func WriteFile(b []byte, absPath string, perm os.FileMode, override bool) error {
	_, err := os.Stat(absPath)
	if err == nil {
		if !override {
			glog.Errorf("Cannot override existing %s", absPath)
			return fmt.Errorf("file exists %s", absPath)
		}
		glog.V(0).Infof("Override existing: %s", absPath)
	}
	fd, err := os.OpenFile(absPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, perm)
	if err != nil {
		glog.Errorf("Unexpected error during file creation: %v", err)
		return err
	}
	defer fd.Close()
	fdW := bufio.NewWriter(fd)
	fdW.Write(b)
	return fdW.Flush()
}

// WritePem write the passed pemBlock bytes to the created file. pemType represents the HEADER of the pem file.
// Override allows to truncate the existing file
func WritePem(b []byte, pemType string, absPath string, perm os.FileMode, override bool) error {
	glog.V(2).Infof("Creating file %s for %s", absPath, pemType)
	_, err := os.Stat(absPath)
	if err == nil {
		if !override {
			glog.Errorf("Cannot override existing %s", absPath)
			return fmt.Errorf("file exists %s", absPath)
		}
		glog.V(0).Infof("Override existing: %s", absPath)
	}
	fd, err := os.OpenFile(absPath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, perm)
	if err != nil {
		glog.Errorf("Unexpected error during %s file creation: %v", pemType, err)
		return err
	}
	defer fd.Close()
	fdW := bufio.NewWriter(fd)
	err = pem.Encode(fdW, &pem.Block{Type: pemType, Bytes: b})
	if err != nil {
		glog.Errorf("Fail to write pem to %s: %v", absPath, err)
		return err
	}
	glog.V(0).Infof("Wrote %s to %s", pemType, absPath)
	return fdW.Flush()
}
