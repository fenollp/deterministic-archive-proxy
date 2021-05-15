package dap

import (
	"io"
	"net/http"
	"strings"
)

// NewGitHubHandler creates a new GitHubHandler
func NewGitHubHandler(opts ...Option) *GitHubHandler {
	return &GitHubHandler{newOptions(opts...)}
}

var _ http.Handler = (*GitHubHandler)(nil)

// GitHubHandler proxies requests for GitHub archives following the pattern
// https://github.com/<user>/<repo>/archive/<hash>.zip or
// https://github.com/<user>/<repo>/archive/<hash>.tar.gz
// It returns deterministic archives or GitHub's own server error.
type GitHubHandler struct {
	options
}

func (ghh *GitHubHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	uri := req.URL.Path[1:]
	if !githubURL(uri) {
		http.Error(rw, "400 Bad Request", http.StatusBadRequest)
		return
	}

	rep, err := ghh.options.client.Get("https://" + uri)
	if err != nil {
		http.Error(rw, rep.Status, rep.StatusCode)
		return
	}
	defer rep.Body.Close()

	for k, vv := range rep.Header {
		header := rw.Header()
		for _, v := range vv {
			header.Add(k, v)
		}
	}

	if _, err := io.Copy(rw, rep.Body); err != nil {
		panic(err)
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
