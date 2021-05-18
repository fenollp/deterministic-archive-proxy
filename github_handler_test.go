package dap

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubHandler(t *testing.T) {
	type Arxiv struct {
		PreHash  string
		PreSize  int64
		PostHash string
		PostSize int64
	}

	for archive, arxiv := range map[string]Arxiv{
		"github.com/bazelbuild/rules_cc/archive/c612c9581b9e740a49ed4c006edb93912c8ab205.tar.gz": {
			PreHash:  "05073d6b8562d9f8913c274b1ec2624c5562b7077da69812b2cb4d7c9aa619ff",
			PreSize:  134294,
			PostHash: "69d0bc7275baaf689b5a53404ef6442a1d1abf3b7882afaacdb93f57464304f0",
			PostSize: 132905,
		},
		"github.com/mokacoding/symlinks/archive/65e46aa9c2fd85d93d97fa9acf2c1365d7157f42.tar.gz": {
			PreHash:  "333f863fd9bef5eaf026f50481bf612c693d2e52fe35d076f43ac83890fc73ec",
			PreSize:  529,
			PostHash: "c72e9ea918fa9743c58a73dc3462b9cf10ab7ff622601d6db12d63ff845b6ad0",
			PostSize: 419,
		},
		"github.com/bazelbuild/rules_java/archive/7cf3cefd652008d0a64a419c34c13bdca6c8f178.zip": {
			PreHash:  "bc81f1ba47ef5cc68ad32225c3d0e70b8c6f6077663835438da8d5733f917598",
			PreSize:  9422,
			PostHash: "5e925d87c1af8bd0af5be947598925e71555905f907b02c672069f79508f32d7",
			PostSize: 9788,
		},
	} {

		// Mock GitHub's archive

		// Turns out they set some metadata about commit hash that Go does not known how to write
		// so our deterministic archives _always_ differ from GitHub's, not just *might*.

		archiveN, archiveH, err := archive256(archive)
		assert.NoError(t, err)
		assert.Equal(t, arxiv.PreHash, archiveH)
		assert.Equal(t, arxiv.PreSize, archiveN)
		ct := "application/x-gzip"
		if strings.HasSuffix(archive, ".zip") {
			ct = "application/zip"
		}

		ts1 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Add(http.CanonicalHeaderKey("content-type"), ct)
			f, err := os.Open(filepath.Join("testdata", filepath.Base(archive)))
			if err != nil {
				panic(err)
			}
			if _, err := io.Copy(rw, f); err != nil {
				panic(err)
			}
		}))
		defer ts1.Close()

		handler1 := NewGitHubHandler(WithHTTPClient(ts1.Client()))
		req := httptest.NewRequest("GET", "https://my_server/"+archive, nil)
		w := httptest.NewRecorder()
		handler1.ServeHTTP(w, req)

		rep := w.Result()

		assert.Equal(t, "200 OK", rep.Status)
		assert.Equal(t, 200, rep.StatusCode)
		assert.Equal(t, ct, rep.Header.Get("Content-Type"))
		contents1, err := io.ReadAll(rep.Body)
		assert.NoError(t, err)
		defer rep.Body.Close()
		assert.Equal(t, arxiv.PostSize, int64(len(contents1)))
		h := sha256.New()
		_, err = io.Copy(h, bytes.NewReader(contents1))
		assert.NoError(t, err)
		assert.Equal(t, arxiv.PostHash, fmt.Sprintf("%x", h.Sum(nil)))

		// Roundtrip

		for i := 1; i <= 3; i++ {

			ts2 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				rw.Header().Add(http.CanonicalHeaderKey("content-type"), ct)
				if _, err := io.Copy(rw, bytes.NewReader(contents1)); err != nil {
					panic(err)
				}
			}))
			defer ts2.Close()

			handler2 := NewGitHubHandler(WithHTTPClient(ts2.Client()))
			req2 := httptest.NewRequest("GET", "https://my_server/"+archive, nil)
			w2 := httptest.NewRecorder()
			handler2.ServeHTTP(w2, req2)

			rep2 := w2.Result()

			assert.Equal(t, "200 OK", rep2.Status)
			assert.Equal(t, 200, rep2.StatusCode)
			assert.Equal(t, ct, rep2.Header.Get("Content-Type"))
			defer rep2.Body.Close()

			var w io.Writer
			h2 := sha256.New()
			var fw *os.File
			if i == 0 {
				tmp := arxiv.PostHash + filepath.Ext(archive)
				t.Logf("Writing temp output to %s", tmp)
				var err error
				fw, err = os.CreateTemp("", tmp)
				assert.NoError(t, err)
				w = io.MultiWriter(fw, h2)
			} else {
				w = h2
			}
			n, err := io.Copy(w, rep2.Body)
			assert.NoError(t, err)
			assert.Equal(t, arxiv.PostSize, n)
			assert.Equal(t, arxiv.PostHash, fmt.Sprintf("%x", h2.Sum(nil)))
			if fw != nil {
				err := fw.Close()
				assert.NoError(t, err)
			}
		}
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
