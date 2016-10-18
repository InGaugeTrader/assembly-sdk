package client

import (
	"time"
)

// options holds the configurable options of a client. It is not meant to be
// used directly; the client initializes it with default values that are then
// modified by `With` lambdas passed to `client.New`.
type options struct {
	// maxCount is the maximum number of transactions the client desires in a
	// single response from the server. This can be overridden by setting the
	// value in the read request.
	maxCount int64

	// callTimeout is the client side timeout set on network calls. This is
	// meant to account for network overhead, so pollTimeout or appendTimeout is
	// added to the timeout for read or append requests respectively.
	callTimeout time.Duration

	// pollTimeout is the maximum duration the client wants the server to delay
	// a empty response while waiting for more data to become available. This
	// can be overridden by setting the value in the read request.
	pollTimeout time.Duration

	// appendTimeout is the maximum duration the clients allows the server for
	// completing an append request.
	appendTimeout time.Duration

	// logger is the logger used by the client.
	logger Logger
}

var defaultOptions = options{
	maxCount:      100,
	pollTimeout:   10 * time.Second,
	appendTimeout: 10 * time.Second,
	callTimeout:   2 * time.Second,
}

type Option func(*options)

// WithMaxCount changes maxCount from the default value.
func WithMaxCount(c int64) Option {
	return func(o *options) {
		o.maxCount = c
	}
}

// WithPollTimeout changes pollTimeout from the default value.
func WithPollTimeout(t time.Duration) Option {
	return func(o *options) {
		o.pollTimeout = t
	}
}

// WithAppendTimeout changes appendTimeout from the default value.
func WithAppendTimeout(t time.Duration) Option {
	return func(o *options) {
		o.appendTimeout = t
	}
}

// WithCallTimeout changes callTimeout from the default value.
func WithCallTimeout(t time.Duration) Option {
	return func(o *options) {
		o.callTimeout = t
	}
}

// WithLogger sets a logger.
func WithLogger(l Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}
