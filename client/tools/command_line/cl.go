// Package main of a simple ledger client command-line tool. It has two
// functions, `publish` which writes a number of strings as separate
// transactions to the ledger, and `scan` which continuously reads transactions
// from the ledger and prints their content.
package main

import (
	"fmt"
	"golang.org/x/net/context"
	"log"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/client/rest"
	"github.com/symbiont-io/assembly-sdk/client/scanner"
)

func main() {
	var rootCmd = &cobra.Command{Use: "client"}

	var verbose bool
	var host string

	var tx_type string
	var cmdPublish = &cobra.Command{
		Use:   "publish [first string to publish] [second ..] [..]",
		Short: "Publish arbitrary strings to ledger",
		RunE: func(cmd *cobra.Command, args []string) error {
			c := client.New(host)
			status, err := c.ServerStatus(context.Background(), nil)
			if err != nil {
				log.Fatalf("Failed to get server status: %v", err)
			}

			txs := []*api.UnsequencedTransaction{}
			for _, arg := range args {
				txs = append(txs, &api.UnsequencedTransaction{
					Type: tx_type,
					Data: []byte(arg),
				})
			}
			before := time.Now()
			res, err := c.AppendTransactions(context.Background(), &api.AppendRequest{status.NetworkSeed, txs})
			if err != nil {
				return err
			}
			if verbose {
				log.Printf("Call completed. Time spent %v", time.Since(before))
				log.Printf("Index of last value sequenced is %d", res.LastIndex)
			}
			return nil
		},
	}

	var cmdScan = &cobra.Command{
		Use:   "scan <index>",
		Short: "Scan transactions from the ledger starting at <index>",
		RunE: func(cmd *cobra.Command, args []string) error {
			index, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("Failed to parse <index>")
			}

			c := client.New(host)
			status, err := c.ServerStatus(context.Background(), nil)
			if err != nil {
				log.Fatalf("Failed to get server status: %v", err)
			}
			s := scanner.New(c)
			txs := s.Scan(index, status.NetworkSeed)
			for tx := range txs {
				fmt.Printf("% 8d (%v)[%s] %s\n", tx.Index, time.Unix(0, tx.Timestamp).UTC(), tx.Type, string(tx.Data))
			}
			if s.Error() != nil {
				log.Fatalf("Failed to scan transactions: %v", s.Error())
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&host, "host", "", "http://localhost:4000", "ledger to connect to")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "output more information")

	rootCmd.AddCommand(cmdPublish)
	cmdPublish.PersistentFlags().StringVarP(&tx_type, "type", "t", "", "type of the transactions being published")

	rootCmd.AddCommand(cmdScan)
	rootCmd.Execute()
}
