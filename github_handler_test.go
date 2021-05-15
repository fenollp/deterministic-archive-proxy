package dap

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubHandler(t *testing.T) {
	handler := NewGitHubHandler()

	archive := "https://my_server/github.com/bazelbuild/rules_cc/archive/c612c9581b9e740a49ed4c006edb93912c8ab205.tar.gz"
	req := httptest.NewRequest("GET", archive, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	rep := w.Result()

	assert.Equal(t, "200 OK", rep.Status)
	assert.Equal(t, 200, rep.StatusCode)
	assert.Equal(t, "application/x-gzip", rep.Header.Get("Content-Type"))
	{
		h := sha256.New()
		_, err := io.Copy(h, rep.Body)
		assert.NoError(t, err)
		assert.Equal(t, "05073d6b8562d9f8913c274b1ec2624c5562b7077da69812b2cb4d7c9aa619ff", fmt.Sprintf("%x", h.Sum(nil)))
	}
}

func TestGitHubHandlerURLs(t *testing.T) {
	for uri, expected := range map[string]bool{
		"github.com/bazelbuild/rules_java/archive/7cf3cefd652008d0a64a419c34c13bdca6c8f178.zip":                        true,
		"mirror.bazel.build/github.com/bazelbuild/rules_proto/archive/7e4afce6fe62dbff0a4a03450143146f9f2d7488.tar.gz": false,
		"github.com/bazelbuild/rules_proto/archive/7e4afce6fe62dbff0a4a03450143146f9f2d7488.tar.gz":                    true,
		"github.com/bboe/deterministic_zip":      false,
		"github.com/rust-lang/cargo/issues/2948": false,
	} {
		assert.Equal(t, expected, githubURL(uri), uri)
	}
}
