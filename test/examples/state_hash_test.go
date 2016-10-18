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

// ExampleVerifyStateHash appends 2 transactions to the ledger, reads them back
// and verify that the state hash is correct.
func ExampleVerifyStateHash() {
	l := mock.NewLedger()
	s := httptest.NewServer(rest.NewServer(l).Router())
	defer s.Close()
	c := client.New(s.URL)

	status, _ := l.ServerStatus(context.Background(), nil)

	// Append 2 transactions to the ledger.
	for _, s := range []string{"semper", "fidelis"} {
		req := &api.AppendRequest{
			NetworkSeed: status.NetworkSeed,
			Transactions: []*api.UnsequencedTransaction{
				&api.UnsequencedTransaction{Data: []byte(s)},
			}}
		_, _ = c.AppendTransactions(context.Background(), req)
	}

	// Read them back.
	res, _ := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: status.NetworkSeed,
		Index:       1,
	})

	var stateHash []byte
	for _, tx := range res.Transactions {
		fmt.Printf("Transaction %d:\n", tx.Index)
		fmt.Printf("  Data:       %q\n", string(tx.Data))
		fmt.Printf("  Hash:       SHA256(%q) = %x\n", string(tx.Data), tx.Hash)
		fmt.Printf("  State hash: SHA256(%x%x) = %x\n", stateHash, tx.Hash, tx.StateHash)
		stateHash = tx.StateHash
	}

	// Output:
	// Transaction 1:
	//   Data:       "semper"
	//   Hash:       SHA256("semper") = 59aa9f6655437abe7a29fd1093641131b61d9f5c827141476a42169a64ecc04f
	//   State hash: SHA256(59aa9f6655437abe7a29fd1093641131b61d9f5c827141476a42169a64ecc04f) = 123a8b99fcce348cc9fa531ed641eccf9d8afb233e4ba8dbfacbd6a34035edfe
	// Transaction 2:
	//   Data:       "fidelis"
	//   Hash:       SHA256("fidelis") = 61b8fff6ea436d38e5f9641942f70eca2d40f2cfcbfc7945078b92a05250bda6
	//   State hash: SHA256(123a8b99fcce348cc9fa531ed641eccf9d8afb233e4ba8dbfacbd6a34035edfe61b8fff6ea436d38e5f9641942f70eca2d40f2cfcbfc7945078b92a05250bda6) = 5a4613a6187bd0198279afaa89741bccf6273e25f80f3bdcae86c4bd5d59a95c
}
