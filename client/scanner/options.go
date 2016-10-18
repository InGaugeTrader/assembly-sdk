package scanner

import "time"

// options holds the configurable options of a scanner. It is not meant to be
// used directly; the scanner initializes it with default values that are then
// modified by `With` lambdas passed to `scanner.New`.
type options struct {
	filter          bool
	transactionType string
	retries         int
	retryPeriod     time.Duration

	// logger is the logger used by the scanner.
	logger Logger
}

var defaultOptions = options{
	retryPeriod: 5 * time.Second,
}

type Option func(*options)

// WithTypeFilter sets a filter on the transaction type, only returning matching transactions.
func WithTypeFilter(t string) Option {
	return func(o *options) {
		o.filter = true
		o.transactionType = t
	}
}

// InfiniteRetries gives infinite retries if passed to WithRetries.
const InfiniteRetries = -1

// WithRetries sets the number of times to retry a failed request.
func WithRetries(count int) Option {
	return func(o *options) {
		o.retries = count
	}
}

// WithLogger sets a logger.
func WithLogger(l Logger) Option {
	return func(o *options) {
		o.logger = l
	}
}
