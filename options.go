package dap

import (
	"net/http"
)

type options struct {
	client *http.Client
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
		opts.client = client
		return nil
	}
}
