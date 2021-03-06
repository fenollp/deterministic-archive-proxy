package dap

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
)

// NewGitHubHandler creates a new GitHubHandler
func NewGitHubHandler(opts ...Option) *GitHubHandler {
	return &GitHubHandler{newOptions(opts...)}
}

var _ sendfiler = (*GitHubHandler)(nil)

// GitHubHandler proxies requests for GitHub archives following the pattern
// https://github.com/<user>/<repo>/archive/<hash>.zip or
// https://github.com/<user>/<repo>/archive/<hash>.tar.gz
// It returns deterministic archives or GitHub's own server error.
type GitHubHandler struct {
	options
}

func (ghh *GitHubHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	uri := r.URL.Path[1:]
	if !githubURL(uri) {
		log.Printf("unhandled URL %q", uri)
		http.Error(rw, "400 Bad Request", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://"+uri, nil)
	if err != nil {
		log.Println("failed preparing forwarded request:", err)
		http.Error(rw, "400 Bad Request", http.StatusBadRequest)
		return
	}
	if token := ghh.options.authToken; token != "" {
		req.Header.Add(hAuthorization, "token "+token)
	}

	rep, err := ghh.options.client.Do(req)
	if err != nil {
		http.Error(rw, "502 Bad Gateway", http.StatusBadGateway)
		return
	}
	defer rep.Body.Close()

	if rep.StatusCode != http.StatusOK {
		http.Error(rw, rep.Status, rep.StatusCode)
		return
	}

	if err := ctx.Err(); err != nil {
		log.Println("ctx done:", err)
		return
	}

	if ghh.sendfile == nil {
		ghh.sendfile = ghh.DefaultSendFile
	}
	if err := ghh.sendfile(ctx, uri, rw, rep); err != nil {
		log.Printf("error handling %q: %v", uri, err)
	}
}

// DefaultSendFile copies the response headers (except content-length, etag and authorization),
// then recomputes the archive so it is deterministic and streams the result.
func (ghh *GitHubHandler) DefaultSendFile(ctx context.Context, uri string, rw http.ResponseWriter, rep *http.Response) error {
	if header := rw.Header(); true {
		for k, vv := range rep.Header {
			if k == hContentLength || k == hETag || k == hAuthorization {
				continue
			}
			for _, v := range vv {
				header.Add(k, v)
			}
		}
	}

	switch {
	case strings.HasSuffix(uri, ".zip"):
		return githubZip(ctx, rw, rep.Body)
	case strings.HasSuffix(uri, ".tar.gz"):
		return githubTarGz(ctx, rw, rep.Body)
	default:
		_, err := io.Copy(rw, rep.Body)
		return err
	}
}

func githubURL(uri string) bool {
	return true &&
		strings.Count(uri, "/") == 4 &&
		strings.HasPrefix(uri, "github.com/") &&
		(false ||
			strings.HasSuffix(uri, ".zip") ||
			strings.HasSuffix(uri, ".tar.gz"))
}

func githubTarGz(ctx context.Context, rw http.ResponseWriter, srcReader io.Reader) error {
	gzipReader, err := gzip.NewReader(srcReader)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	// No Close()r

	type Contents struct {
		Header tar.Header
		Data   []byte
	}
	contents := make(map[string]Contents)
	var filenames []string
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeReg, tar.TypeSymlink:
			log.Println("reading tar.gz", header.Name)
			filenames = append(filenames, header.Name)
			data, err := io.ReadAll(tarReader)
			if err != nil {
				return err
			}
			contents[header.Name] = Contents{
				Header: *header,
				Data:   data,
			}
		case tar.TypeDir:
			continue
		case tar.TypeXGlobalHeader:
			continue // Ignore pax_global_header
		default:
			err := fmt.Errorf("unhandled type %c in %q", header.Typeflag, header.Name)
			return err
		}
	}

	gzipWriter := gzip.NewWriter(rw)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	sort.Strings(filenames)

	for _, fname := range filenames {
		if err := ctx.Err(); err != nil {
			return err
		}

		log.Println("writing tar.gz", fname)
		pair := contents[fname]
		if err := tarWriter.WriteHeader(&pair.Header); err != nil {
			return err
		}
		buffer := bytes.NewReader(pair.Data)
		if _, err := io.CopyN(tarWriter, buffer, pair.Header.Size); err != nil {
			return err
		}
	}

	return nil
}

func githubZip(ctx context.Context, rw http.ResponseWriter, srcReader io.Reader) error {
	// Download whole ZIP to memory, as a way to provide a ReaderAt for zip.NewReader
	whole, err := io.ReadAll(srcReader)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	srcReaderAt := bytes.NewReader(whole)
	zipReader, err := zip.NewReader(srcReaderAt, int64(len(whole)))
	if err != nil {
		return err
	}
	// No Close()r

	type Contents struct {
		Header zip.FileHeader
		Data   []byte
	}
	contents := make(map[string]Contents)
	var filenames []string
	for _, f := range zipReader.File {
		if err := ctx.Err(); err != nil {
			return err
		}

		log.Println("reading zip")
		filenames = append(filenames, f.Name)
		fr, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(fr)
		fr.Close()
		if err != nil {
			return err
		}
		contents[f.Name] = Contents{
			Header: f.FileHeader,
			Data:   data,
		}
	}

	zipWriter := zip.NewWriter(rw)

	sort.Strings(filenames)

	for _, fname := range filenames {
		if err := ctx.Err(); err != nil {
			return err
		}

		log.Println("writing zip", fname)
		pair := contents[fname]

		fw, err := zipWriter.CreateHeader(&pair.Header)
		if err != nil {
			return err
		}

		buffer := bytes.NewReader(pair.Data)
		if _, err := io.CopyN(fw, buffer, int64(len(pair.Data))); err != nil {
			return err
		}
	}

	return zipWriter.Close()
}
