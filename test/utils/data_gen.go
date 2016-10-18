package utils

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
)

func RandomData(n, size int) [][]byte {
	data := make([][]byte, n)
	for i, _ := range data {
		data[i] = make([]byte, size)
		_, err := rand.Read(data[i])
		if err != nil {
			panic(err.Error())
		}
	}
	return data
}

func RandomUnsequencedTransactions(n, size int) []*api.UnsequencedTransaction {
	data := RandomData(n, size)
	var txs []*api.UnsequencedTransaction
	for _, d := range data {
		txs = append(txs, &api.UnsequencedTransaction{Data: d})
	}
	return txs
}

func MockTypedSequencedTransaction(t string, index int64) *api.SequencedTransaction {
	data := []byte(fmt.Sprintf("data %d", index))
	hash := sha256.Sum256(append([]byte(t), data...))
	return &api.SequencedTransaction{
		t,
		index,
		index * int64(time.Second),
		data,
		hash[:],
		[]byte("mock-state-hash"),
	}
}

func MockSequencedTransaction(index int64) *api.SequencedTransaction {
	return MockTypedSequencedTransaction("", index)
}
