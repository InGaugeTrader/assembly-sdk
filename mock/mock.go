// Package mock is a mock ledger implementation. It implements the Ledger
// interface and has the append only semantics of a real ledger, but that's it.
// There's no networking and thus no BFT. Storage is in memory and is wiped on
// restart.
package mock

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"golang.org/x/net/context"
	"sync"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
)

// Ledger acts like a real (single-node) ledger. It holds all written
// transactions in memory.
type Ledger struct {
	data      []*api.SequencedTransaction
	seed      []byte
	stateHash []byte

	mu      sync.Mutex
	newData chan struct{}
}

// NewLedger create a new mock.Ledger object that implements api.Ledger.
func NewLedger() *Ledger {
	seed := make([]byte, 32)
	rand.Read(seed)

	l := Ledger{
		seed:    seed,
		newData: make(chan struct{}),
	}
	return &l
}

// verifySeed verifies that the provided seed matches that of the mock
// ledger. If no seed is provided, it assumes the client doesn't care about
// this protection and passes the verification.
func (l *Ledger) verifySeed(seed []byte) bool {
	if len(seed) > 0 && !bytes.Equal(seed, l.seed) {
		return false
	}
	return true
}

// ReadTransactions reads transactions from the in-memory storage of the mock
// ledger. If no new transactions are available it will wait for new ones until
// the provided timeout.
func (l *Ledger) ReadTransactions(ctx context.Context, req *api.ReadRequest) (*api.ReadResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.verifySeed(req.NetworkSeed) {
		return nil, api.NetworkSeedMismatchError(l.seed)
	}
	if req.Index > int64(len(l.data))+1 {
		return nil, api.NotFoundError("Requested index is too far in the future")
	} else if req.Index == int64(len(l.data))+1 {
		// Wait for new transactions to arrive.
		waitCh := l.newData // copy channel while holding mutex

		l.mu.Unlock()
		select {
		case <-waitCh:
		case <-ctx.Done():
		}
		l.mu.Lock()
	}
	return &api.ReadResult{l.seed, l.readData(req.Index, int(req.Count))}, nil
}

// readData reads a slice of transactions from the data array, starting at
// index and with length count. Indexes counts from 1 and length will be
// truncated to stay within bounds.
func (l *Ledger) readData(index int64, count int) []*api.SequencedTransaction {
	i := int(index - 1)
	if count > len(l.data)-i {
		count = len(l.data) - i
	}
	return l.data[i : i+count]
}

// AppendTransactions appends the provided array of transactions to the
// in-memory storage of the mock ledger and wakes any waiting readers.
func (l *Ledger) AppendTransactions(ctx context.Context, req *api.AppendRequest) (*api.AppendResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.verifySeed(req.NetworkSeed) {
		return nil, api.NetworkSeedMismatchError(l.seed)
	}

	index := int64(len(l.data) + 1)
	for _, tx := range req.Transactions {
		stateHash := sha256.Sum256(append(l.stateHash, tx.Hash...))
		l.data = append(l.data, &api.SequencedTransaction{
			Type:      tx.Type,
			Index:     index,
			Data:      tx.Data,
			Hash:      tx.Hash,
			StateHash: stateHash[:],
			Timestamp: time.Now().UnixNano(),
		})
		l.stateHash = stateHash[:]
		index++
	}

	// Signal arrival of new data to waiting readers.
	close(l.newData)
	l.newData = make(chan struct{})

	return &api.AppendResult{l.seed, int64(len(l.data))}, nil
}

// ServerStatus returns the status of the local node.
func (l *Ledger) ServerStatus(ctx context.Context, _ *api.Empty) (*api.ServerStatusResult, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	return &api.ServerStatusResult{
		NetworkType: "mock",
		NetworkSeed: l.seed,
		LastIndex:   int64(len(l.data)),
		ServerTime:  time.Now().UnixNano(),
		Ready:       true, // Mock ledger is always ready.
	}, nil
}
