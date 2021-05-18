package dap

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func archive256(archive string) (int64, string, error) {
	f, err := os.Open(filepath.Join("testdata", filepath.Base(archive)))
	if err != nil {
		return 0, "", err
	}

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return 0, "", err
	}
	return n, fmt.Sprintf("%x", h.Sum(nil)), nil
}
