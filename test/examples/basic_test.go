package examples_test

import (
	"fmt"
	"golang.org/x/net/context"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/api/rest"
	"github.com/symbiont-io/assembly-sdk/client/rest"
	"github.com/symbiont-io/assembly-sdk/mock"

	"net/http/httptest"
)

// ExampleAppendRead appends 4 transactions to the ledger in sequence and
// read back selected indexes.
func ExampleAppendRead() {
	l := mock.NewLedger()
	s := httptest.NewServer(rest.NewServer(l).Router())
	defer s.Close()
	c := client.New(s.URL)

	status, _ := l.ServerStatus(context.Background(), nil)

	// Append 4 transactions to the ledger.
	for _, s := range []string{"alpha", "beta", "gamma", "delta"} {
		req := &api.AppendRequest{
			NetworkSeed: status.NetworkSeed,
			Transactions: []*api.UnsequencedTransaction{
				&api.UnsequencedTransaction{Data: []byte(s)},
			}}
		_, _ = c.AppendTransactions(context.Background(), req)
	}

	// Read from transaction 2, maximum 2 transactions.
	res1, _ := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: status.NetworkSeed,
		Index:       2,
		Count:       2,
	})
	for _, tx := range res1.Transactions {
		fmt.Println(string(tx.Data))
	}

	// Read from transaction 1, maximum 1 transactions.
	res2, _ := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: status.NetworkSeed,
		Index:       1,
		Count:       1,
	})
	for _, tx := range res2.Transactions {
		fmt.Println(string(tx.Data))
	}

	// Output:
	// beta
	// gamma
	// alpha
}
