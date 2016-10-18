// Rest API server <-> client integration tests
package integration_test

import (
	"crypto/sha256"
	"github.com/jonboulle/clockwork"
	"golang.org/x/net/context"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/api/rest"
	"github.com/symbiont-io/assembly-sdk/client/rest"
	"github.com/symbiont-io/assembly-sdk/mock"

	"github.com/nbio/st"
	"github.com/symbiont-io/assembly-sdk/test/utils"
	"net/http/httptest"
	"testing"
)

func TestPollTimeoutDefault(t *testing.T) {
	fakeClock := clockwork.NewFakeClock()
	f := func(_ context.Context, to time.Duration) (context.Context, context.CancelFunc) {
		return utils.NewTestContextWithTimeout(fakeClock, to), func() {}
	}
	l := mock.NewLedger()
	s := httptest.NewServer(rest.NewServer(l, rest.WithTimeoutContextFactory(f)).Router())
	defer s.Close()
	c := client.New(s.URL, client.WithPollTimeout(10*time.Minute))

	ctx := context.Background()
	status, err := l.ServerStatus(ctx, nil)
	st.Assert(t, err, nil)

	fut := make(chan *api.ReadResult, 1)
	go func() {
		res, err := c.ReadTransactions(ctx, &api.ReadRequest{
			NetworkSeed: status.NetworkSeed,
			Index:       1,
		})
		st.Assert(t, err, nil)
		fut <- res
	}()

	fakeClock.BlockUntil(1) // Wait for the api to start its timer.
	fakeClock.Advance(9 * time.Minute)
	select {
	case <-fut:
		t.Fatal("Result ready too soon")
	case <-time.After(100 * time.Millisecond): // allow time for completing request.
	}

	fakeClock.Advance(1 * time.Minute)
	select {
	case res := <-fut:
		st.Expect(t, len(res.Transactions), 0)
	case <-time.After(1 * time.Second): // allow time for scheduling
		t.Error("Result not ready in time")
	}
}

func TestPollTimeoutContextOverride(t *testing.T) {
	fakeClock := clockwork.NewFakeClock()
	f := func(_ context.Context, to time.Duration) (context.Context, context.CancelFunc) {
		return utils.NewTestContextWithTimeout(fakeClock, to), func() {}
	}
	l := mock.NewLedger()
	s := httptest.NewServer(rest.NewServer(l, rest.WithTimeoutContextFactory(f)).Router())
	defer s.Close()
	c := client.New(s.URL)

	status, err := l.ServerStatus(context.Background(), nil)
	st.Assert(t, err, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	// Overriding the timeout with the context should result in the server
	// responding in half that time (leaving 50% room for network overhead).
	defer cancel()
	fut := make(chan *api.ReadResult, 1)
	go func() {
		res, err := c.ReadTransactions(ctx, &api.ReadRequest{
			NetworkSeed: status.NetworkSeed,
			Index:       1,
		})
		st.Assert(t, err, nil)
		fut <- res
	}()
	fakeClock.BlockUntil(1) // Wait for the api to start its timer.
	fakeClock.Advance(9 * time.Minute)
	select {
	case <-fut:
		t.Fatal("Result ready too soon")
	case <-time.After(100 * time.Millisecond): // allow time for completing request.
	}

	fakeClock.Advance(1 * time.Minute)
	select {
	case res := <-fut:
		st.Expect(t, len(res.Transactions), 0)
	case <-time.After(1 * time.Second): // allow time for scheduling
		t.Error("Result not ready in time")
	}
}

func TestAppendRead(t *testing.T) {
	l := mock.NewLedger()
	s := httptest.NewServer(rest.NewServer(l).Router())
	defer s.Close()
	c := client.New(s.URL)

	n := 100
	txs := utils.RandomUnsequencedTransactions(n, 100)

	ctx := context.Background()
	status, err := l.ServerStatus(ctx, nil)
	st.Assert(t, err, nil)
	appendRes, err := c.AppendTransactions(ctx, &api.AppendRequest{status.NetworkSeed, txs})
	st.Assert(t, err, nil)
	st.Expect(t, appendRes.LastIndex, int64(n))

	readRes, err := c.ReadTransactions(ctx, &api.ReadRequest{
		NetworkSeed: status.NetworkSeed,
		Index:       1,
	})
	st.Assert(t, err, nil)
	st.Expect(t, readRes.NetworkSeed, status.NetworkSeed)
	stateHash := make([]byte, 0)
	for i, tx := range readRes.Transactions {
		st.Expect(t, tx.Index, int64(i+1))
		st.Expect(t, tx.Data, txs[i].Data)

		hash := sha256.Sum256(append([]byte(tx.Type), tx.Data...))
		st.Expect(t, tx.Hash, hash[:])

		newStateHash := sha256.Sum256(append(stateHash, hash[:]...))
		st.Expect(t, tx.StateHash, newStateHash[:])
		stateHash = newStateHash[:]
	}
}

func TestAppendReadOffset(t *testing.T) {
	l := mock.NewLedger()
	s := httptest.NewServer(rest.NewServer(l).Router())
	defer s.Close()
	c := client.New(s.URL)

	n := 100
	txs := utils.RandomUnsequencedTransactions(n, 100)

	ctx := context.Background()
	status, err := l.ServerStatus(ctx, nil)
	st.Assert(t, err, nil)
	appendRes, err := c.AppendTransactions(ctx, &api.AppendRequest{status.NetworkSeed, txs})
	st.Assert(t, err, nil)
	st.Expect(t, appendRes.LastIndex, int64(n))

	readRes, err := c.ReadTransactions(ctx, &api.ReadRequest{
		NetworkSeed: status.NetworkSeed,
		Index:       20,
	})
	st.Assert(t, err, nil)
	st.Expect(t, readRes.NetworkSeed, status.NetworkSeed)
	st.Expect(t, len(readRes.Transactions), n-20+1)
	for i, tx := range readRes.Transactions {
		st.Expect(t, tx.Index, 20+int64(i))
		st.Expect(t, tx.Data, txs[20+i-1].Data)
	}
}

func TestBadNetworkSeed(t *testing.T) {
	l := mock.NewLedger()
	s := httptest.NewServer(rest.NewServer(l).Router())
	defer s.Close()
	c := client.New(s.URL)

	txs := utils.RandomUnsequencedTransactions(5, 100)

	ctx := context.Background()
	status, err := l.ServerStatus(ctx, nil)
	st.Assert(t, err, nil)
	_, err = c.AppendTransactions(ctx, &api.AppendRequest{[]byte("bad seed"), txs})
	st.Reject(t, err, nil)
	e, ok := err.(api.NetworkSeedMismatchError)
	st.Assert(t, ok, true)
	st.Expect(t, e.CorrectSeed(), status.NetworkSeed)

	_, err = c.ReadTransactions(ctx, &api.ReadRequest{
		NetworkSeed: []byte("also bad"),
		Index:       1,
	})
	st.Reject(t, err, nil)
	e, ok = err.(api.NetworkSeedMismatchError)
	st.Assert(t, ok, true)
	st.Expect(t, e.CorrectSeed(), status.NetworkSeed)
}
