// This is a simple chat client that demonstrate usage of the client library.
package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/context"
	"log"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/client/rest"
	"github.com/symbiont-io/assembly-sdk/client/scanner"
)

var host = flag.String("host", "http://localhost:4000", "address to connect to")
var name = flag.String("name", "Assembly user", "name to associate with published messages")

func main() {
	flag.Parse()

	c := client.New(*host)

	// Read out messages from the ledger as they are published.
	go func() {
		s := scanner.New(c, scanner.WithTypeFilter("example/chat"))
		txs := s.Scan(1, nil)
		for tx := range txs {
			fmt.Printf("%v: %s\n", time.Unix(0, tx.Timestamp).UTC().Format(time.RFC3339), string(tx.Data))
		}
		if s.Error() != nil {
			log.Fatalf("Failed to scan transactions: %v", s.Error())
		}
	}()

	// Read input from stdin and publish it to the ledger, with the name of the
	// user prepended.
	for {
		var line string
		_, err := fmt.Scanln(&line)
		if err != nil {
			log.Fatalf("Failed to read user input: %v", err)
		}
		_, _ = c.AppendTransactions(context.Background(), &api.AppendRequest{nil, []*api.UnsequencedTransaction{
			&api.UnsequencedTransaction{Type: "example/chat", Data: []byte(*name + ": " + line)},
		}})
	}
}
