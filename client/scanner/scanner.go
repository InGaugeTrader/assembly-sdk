// Package scanner is a wrapper of a client that preforms repeated reads and
// outputs a stream of transactions.
package scanner

import (
	"fmt"
	"golang.org/x/net/context"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
)

// Client is an interface describing the clients a Scanner can wrap.
// Essentially all that's needed is a Read method.
type Client interface {
	ReadTransactions(context.Context, *api.ReadRequest) (*api.ReadResult, error)
}

// Scanner wraps a client and outputs a stream of transactions.
type Scanner struct {
	err     error
	client  Client
	options options
}

// New creates a new scanner. Multiple scanners can share the same underlying
// Client.
func New(client Client, opt ...Option) *Scanner {
	s := Scanner{
		client:  client,
		options: defaultOptions,
	}
	for _, o := range opt {
		o(&s.options)
	}
	return &s
}

// Scan starts reading transactions through the underlying client, starting at
// index and with the provided network seed. It returns a channel that all
// received transactions will be output on. It keeps reading until an error
// occurs, which must be checked by calling Error(). Should not be called
// concurrently, but new Scan calls on the same Scanner are fine once the
// previous one has completed (channel closed).
func (s *Scanner) Scan(index int64, seed []byte) <-chan *api.SequencedTransaction {
	results := make(chan *api.SequencedTransaction)

	go func() {
		retried := 0
		for {
			res, err := s.client.ReadTransactions(context.Background(), &api.ReadRequest{
				NetworkSeed: seed,
				Index:       index,
			})
			if err != nil {
				if s.options.retries > retried || s.options.retries == InfiniteRetries {
					retried++
					s.infof("Request failed (%s), sleeping %s then retrying (%d/%d)",
						err, s.options.retryPeriod, retried, s.options.retries)
					time.Sleep(s.options.retryPeriod)
					continue
				}
				s.err = err
				break
			}
			retried = 0
			for _, tx := range res.Transactions {
				if tx.Index != index {
					s.err = fmt.Errorf("Unexpected transaction index (expected %d, got %d)",
						index, tx.Index)
					break
				}
				if !s.options.filter || tx.Type == s.options.transactionType {
					results <- tx
				}
				index++
			}
		}
		close(results)
	}()

	return results
}

// Error returns the error from the last Scan, or nil if no error occurred.
func (s *Scanner) Error() error {
	return s.err
}
