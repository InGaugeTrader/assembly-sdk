package mock_test

import (
	"github.com/jonboulle/clockwork"
	"golang.org/x/net/context"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/mock"

	"github.com/nbio/st"
	"github.com/symbiont-io/assembly-sdk/test/utils"
	"testing"
)

func TestAppendAndRead(t *testing.T) {
	l := mock.NewLedger()
	n := 100
	txs := utils.RandomUnsequencedTransactions(n, 100)

	ctx := context.Background()
	status, _ := l.ServerStatus(ctx, nil)
	appendRes, err := l.AppendTransactions(ctx, &api.AppendRequest{status.NetworkSeed, txs})
	st.Assert(t, err, nil)
	st.Expect(t, appendRes.LastIndex, int64(n))

	fut := make(chan *api.ReadResult, 1)
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		res, err := l.ReadTransactions(ctx, &api.ReadRequest{nil, 20, 1000})
		st.Assert(t, err, nil)
		fut <- res
	}()
	select {
	case res := <-fut:
		st.Expect(t, res.NetworkSeed, status.NetworkSeed)
		st.Expect(t, len(res.Transactions), n-20+1)
		for i, tx := range res.Transactions {
			st.Expect(t, tx.Index, 20+int64(i))
			st.Expect(t, tx.Data, txs[20+int64(i)-1].Data)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("Timed out reading transactions")
	}
}

func TestReadTooFarAhead(t *testing.T) {
	l := mock.NewLedger()
	n := 100
	txs := utils.RandomUnsequencedTransactions(n, 100)

	ctx := context.Background()
	status, _ := l.ServerStatus(ctx, nil)
	appendRes, err := l.AppendTransactions(ctx, &api.AppendRequest{status.NetworkSeed, txs})
	st.Assert(t, err, nil)
	st.Assert(t, appendRes.LastIndex, int64(n))

	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	_, err = l.ReadTransactions(ctx, &api.ReadRequest{nil, 102, 1})
	st.Refute(t, err, nil)
	_, ok := err.(api.NotFoundError)
	st.Assert(t, ok, true)
}

func TestBadNetworkSeed(t *testing.T) {
	l := mock.NewLedger()
	txs := utils.RandomUnsequencedTransactions(5, 100)

	_, err := l.AppendTransactions(context.Background(), &api.AppendRequest{[]byte("bad seed"), txs})
	_, rejected := err.(api.NetworkSeedMismatchError)
	st.Assert(t, rejected, true)
}

func TestPollTimeout(t *testing.T) {
	fakeClock := clockwork.NewFakeClock()
	l := mock.NewLedger()

	fut := make(chan *api.ReadResult, 1)
	go func() {
		ctx := utils.NewTestContextWithTimeout(fakeClock, 5*time.Minute)
		res, err := l.ReadTransactions(ctx, &api.ReadRequest{nil, 1, 1})
		st.Assert(t, err, nil)
		fut <- res
	}()
	fakeClock.BlockUntil(1) // Wait for the mock to start its timer.
	select {
	case <-fut:
		t.Error("Result ready too soon")
	default:
	}

	fakeClock.Advance(4 * time.Minute)
	select {
	case <-fut:
		t.Error("Result still ready too soon")
	default:
	}

	fakeClock.Advance(1 * time.Minute)
	select {
	case res := <-fut:
		st.Expect(t, len(res.Transactions), 0)
	case <-time.After(1 * time.Second): // allow time for scheduling
		t.Error("Result not ready txs time")
	}
}

func TestRealClock(t *testing.T) {
	l := mock.NewLedger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	res, err := l.ReadTransactions(ctx, &api.ReadRequest{nil, 1, 1})
	st.Assert(t, err, nil)
	st.Expect(t, len(res.Transactions), 0)
}
