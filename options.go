package dap

import (
	"errors"
	"net/http"
)

type options struct {
	client    *http.Client
	sendfile  SendFiler
	authToken string
}

func newOptions(opts ...Option) options {
	o := &options{
		client: http.DefaultClient,
	}
	for _, opt := range opts {
		if err := opt(o); err != nil {
			panic(err)
		}
	}
	return *o
}

// Option specializes a handler.
type Option func(*options) error

// WithHTTPClient sets the HTTP client to use when requesting the original archive.
// By default, http.DefaultClient is used which is safe for concurrent use by multiple goroutines.
func WithHTTPClient(client *http.Client) Option {
	return func(opts *options) error {
		if client == nil {
			return errors.New("proxy option: given HTTP client is nil")
		}
		opts.client = client
		return nil
	}
}

// WithSendFiler controls how the HTTP response is written.
// Override this to use e.g. caching.
func WithSendFiler(fn SendFiler) Option {
	return func(opts *options) error {
		if fn == nil {
			return errors.New("proxy option: given SendFiler is nil")
		}
		opts.sendfile = fn
		return nil
	}
}

// WithAuthToken sets the token to use when requesting tarballs from upstream.
func WithAuthToken(token string) Option {
	return func(opts *options) error {
		if token == "" {
			return errors.New("proxy option: given auth token is empty")
		}
		opts.authToken = token
		return nil
	}
}
