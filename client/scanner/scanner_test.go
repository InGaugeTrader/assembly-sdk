package scanner_test

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/net/context"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/client/scanner"

	"github.com/nbio/st"
	"github.com/symbiont-io/assembly-sdk/test/utils"
	"testing"
)

type mockClient struct {
	results []*api.ReadResult
}

func (mc *mockClient) ReadTransactions(_ context.Context, req *api.ReadRequest) (*api.ReadResult, error) {
	if len(mc.results) > 0 {
		// pop front
		r := mc.results[0]
		mc.results = mc.results[1:]
		if !bytes.Equal(r.NetworkSeed, req.NetworkSeed) {
			return r, fmt.Errorf("Network seed mismatch")
		}
		return r, nil
	} else {
		return nil, fmt.Errorf("done")
	}
}

func TestScannerOK(t *testing.T) {
	s := scanner.New(&mockClient{
		[]*api.ReadResult{
			&api.ReadResult{
				NetworkSeed: []byte("some seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(1),
					utils.MockSequencedTransaction(2),
				},
			},
			&api.ReadResult{
				NetworkSeed: []byte("some seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(3),
				},
			},
		},
	})
	txs := s.Scan(1, []byte("some seed"))
	i := int64(1)
	for tx := range txs {
		st.Expect(t, tx.Index, i)
		i++
	}
	st.Expect(t, i, int64(4))
	st.Expect(t, s.Error(), errors.New("done"))
}

func TestScannerOffset(t *testing.T) {
	s := scanner.New(&mockClient{
		[]*api.ReadResult{
			&api.ReadResult{
				NetworkSeed: []byte("some seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(100),
					utils.MockSequencedTransaction(101),
				},
			},
			&api.ReadResult{
				NetworkSeed: []byte("some seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(102),
				},
			},
		},
	})
	txs := s.Scan(100, []byte("some seed"))
	i := int64(100)
	for tx := range txs {
		st.Expect(t, tx.Index, i)
		i++
	}
	st.Expect(t, i, int64(103))
	st.Expect(t, s.Error(), errors.New("done"))
}

func TestScannerBadIndex(t *testing.T) {
	s := scanner.New(&mockClient{
		[]*api.ReadResult{
			&api.ReadResult{
				NetworkSeed: []byte("some seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(2),
					utils.MockSequencedTransaction(3),
				},
			},
		},
	})
	txs := s.Scan(1, []byte("some seed"))
	for _ = range txs { // Exhaust output
	}
	st.Reject(t, s.Error(), nil)
}

func TestScannerIndexJump(t *testing.T) {
	s := scanner.New(&mockClient{
		[]*api.ReadResult{
			&api.ReadResult{
				NetworkSeed: []byte("some seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(1),
					utils.MockSequencedTransaction(2),
				},
			},
			&api.ReadResult{
				NetworkSeed: []byte("some seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(4),
				},
			},
		},
	})
	txs := s.Scan(1, []byte("some seed"))
	for _ = range txs { // Exhaust output
	}
	st.Reject(t, s.Error(), nil)
}

func TestScannerBadSeed(t *testing.T) {
	s := scanner.New(&mockClient{
		[]*api.ReadResult{
			&api.ReadResult{
				NetworkSeed: []byte("bad seed"),
				Transactions: []*api.SequencedTransaction{
					utils.MockSequencedTransaction(1),
					utils.MockSequencedTransaction(2),
				},
			},
		},
	})
	txs := s.Scan(1, []byte("some seed"))
	for _ = range txs { // Exhaust output
	}
	st.Reject(t, s.Error(), nil)
}

func TestScannerWithType(t *testing.T) {
	s := scanner.New(&mockClient{
		[]*api.ReadResult{
			&api.ReadResult{
				Transactions: []*api.SequencedTransaction{
					utils.MockTypedSequencedTransaction("a", 1),
					utils.MockTypedSequencedTransaction("b", 2),
					utils.MockTypedSequencedTransaction("", 3),
					utils.MockTypedSequencedTransaction("a", 4),
				},
			},
		},
	})
	txs := s.Scan(1, nil)
	i := int64(1)
	for tx := range txs {
		st.Expect(t, tx.Index, i)
		i++
	}
	st.Expect(t, i, int64(5))
	st.Expect(t, s.Error(), errors.New("done"))
}

func TestScannerWithFilter(t *testing.T) {
	s := scanner.New(&mockClient{
		[]*api.ReadResult{
			&api.ReadResult{
				Transactions: []*api.SequencedTransaction{
					utils.MockTypedSequencedTransaction("a", 1),
					utils.MockTypedSequencedTransaction("b", 2),
					utils.MockTypedSequencedTransaction("a", 3),
				},
			},
		},
	}, scanner.WithTypeFilter("a"))
	txs := s.Scan(1, nil)
	count := 0
	for tx := range txs {
		st.Expect(t, tx.Type, "a")
		count++
	}
	st.Expect(t, count, 2)
	st.Expect(t, s.Error(), errors.New("done"))
}
