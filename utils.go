package dap

import (
	"context"
	"net/http"
)

var (
	hAuthorization = http.CanonicalHeaderKey("authorization")
	hContentLength = http.CanonicalHeaderKey("content-length")
	hETag          = http.CanonicalHeaderKey("etag")
)

// SendFiler writes the HTTP response.
type SendFiler func(ctx context.Context, uri string, rw http.ResponseWriter, rep *http.Response) error

type sendfiler interface {
	http.Handler
	// DefaultSendFile is a SendFiler that gets used if not overriden in options.
	DefaultSendFile(ctx context.Context, uri string, rw http.ResponseWriter, rep *http.Response) error
}
