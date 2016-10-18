package rest

import (
	"golang.org/x/net/context"
	"time"
)

const (
	DefaultCount       = 100
	DefaultMaxCount    = 1000
	DefaultPollTimeout = 5 * time.Second
)

type timeoutContextFactory func(context.Context, time.Duration) (context.Context, context.CancelFunc)

type options struct {
	defaultCount       int64
	maxCount           int64
	defaultPollTimeout time.Duration
	logger             Logger
	contextWithTimeout timeoutContextFactory
}

var defaultOptions = options{
	defaultCount:       DefaultCount,
	maxCount:           DefaultMaxCount,
	defaultPollTimeout: DefaultPollTimeout,
	contextWithTimeout: func(parent context.Context, to time.Duration) (
		context.Context, context.CancelFunc) {
		return context.WithTimeout(parent, to)
	},
}

type Option func(*options)

func WithDefaultCount(c int64) Option {
	return func(o *options) {
		o.defaultCount = c
	}
}

func WithMaxCount(c int64) Option {
	return func(o *options) {
		o.maxCount = c
	}
}

func WithDefaultPollTimeout(to time.Duration) Option {
	return func(o *options) {
		o.defaultPollTimeout = to
	}
}

func WithLogger(l Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}

func WithTimeoutContextFactory(f timeoutContextFactory) Option {
	return func(o *options) {
		o.contextWithTimeout = f
	}
}
