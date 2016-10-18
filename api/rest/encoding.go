package rest

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/symbiont-io/assembly-sdk/api"
)

func EncodeAppendRequest(in *api.AppendRequest) ([]byte, error) {
	data := new(bytes.Buffer)
	enc := json.NewEncoder(data)
	err := enc.Encode(&AppendRequest{
		EncodeUnsequencedTransactions(in.Transactions),
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to encode request: %v", err)
	}
	return data.Bytes(), nil
}

func DecodeAppendRequest(body io.Reader) (api.AppendRequest, error) {
	var out api.AppendRequest
	var err error
	var msg AppendRequest

	dec := json.NewDecoder(body)
	err = dec.Decode(&msg)
	if err != nil {
		return out, fmt.Errorf("Failed to decode request: %v", err)
	}
	out.Transactions, err = DecodeUnsequencedTransactions(msg.Transactions)
	if err != nil {
		return out, fmt.Errorf("Failed to decode transactions: %v", err)
	}
	return out, nil
}

func EncodeSequencedTransactions(in []*api.SequencedTransaction) []*EncodedSequencedTransaction {
	out := make([]*EncodedSequencedTransaction, 0, len(in))
	for _, tx := range in {
		out = append(out, &EncodedSequencedTransaction{
			Index:     tx.Index,
			Timestamp: tx.Timestamp,
			Data:      base64.StdEncoding.EncodeToString(tx.Data),
			Hash:      hex.EncodeToString(tx.Hash),
			StateHash: hex.EncodeToString(tx.StateHash),
			Type:      tx.Type,
		})
	}
	return out
}

func DecodeSequencedTransactions(in []*EncodedSequencedTransaction) ([]*api.SequencedTransaction, error) {
	out := make([]*api.SequencedTransaction, 0, len(in))
	for i, tx := range in {
		data, err := base64.StdEncoding.DecodeString(tx.Data)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode transaction %d: %v", i, err)
		}
		hash := sha256.Sum256(append([]byte(tx.Type), data...))
		if tx.Hash != hex.EncodeToString(hash[:]) {
			return nil, fmt.Errorf("Hash mismatch on transaction %d", i)
		}
		stateHash, err := hex.DecodeString(tx.StateHash)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode state hash for transaction %d: %v", i, err)
		}
		out = append(out, &api.SequencedTransaction{
			Index:     tx.Index,
			Timestamp: tx.Timestamp,
			Data:      data,
			Hash:      hash[:],
			StateHash: stateHash,
			Type:      tx.Type,
		})
	}
	return out, nil
}

func EncodeUnsequencedTransactions(in []*api.UnsequencedTransaction) []*EncodedUnsequencedTransaction {
	out := make([]*EncodedUnsequencedTransaction, 0, len(in))
	for _, tx := range in {
		if len(tx.Hash) == 0 {
			hash := sha256.Sum256(append([]byte(tx.Type), tx.Data...))
			tx.Hash = hash[:]
		}
		out = append(out, &EncodedUnsequencedTransaction{
			Data: base64.StdEncoding.EncodeToString(tx.Data),
			Hash: hex.EncodeToString(tx.Hash),
			Type: tx.Type,
		})
	}
	return out
}

func DecodeUnsequencedTransactions(in []*EncodedUnsequencedTransaction) ([]*api.UnsequencedTransaction, error) {
	out := make([]*api.UnsequencedTransaction, 0, len(in))
	for i, tx := range in {
		data, err := base64.StdEncoding.DecodeString(tx.Data)
		if err != nil {
			return nil, fmt.Errorf("Failed to decode transaction %d: %v", i, err)
		}
		hash := sha256.Sum256(append([]byte(tx.Type), data...))
		if tx.Hash != hex.EncodeToString(hash[:]) {
			return nil, fmt.Errorf("Hash mismatch on transaction %d", i)
		}
		out = append(out, &api.UnsequencedTransaction{
			Data: data,
			Hash: hash[:],
			Type: tx.Type,
		})
	}
	return out, nil
}

func EncodeServerStatus(in *api.ServerStatusResult) *ServerStatusResult {
	return &ServerStatusResult{
		NetworkType: in.NetworkType,
		NetworkSeed: hex.EncodeToString(in.NetworkSeed),
		LastIndex:   in.LastIndex,
		ServerTime:  in.ServerTime,
		Ready:       in.Ready,
		Version:     Version,
	}
}

func DecodeServerStatus(in *ServerStatusResult) (*api.ServerStatusResult, error) {
	seed, err := hex.DecodeString(in.NetworkSeed)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode network seed: %v", err)
	}
	return &api.ServerStatusResult{
		NetworkType: in.NetworkType,
		NetworkSeed: seed,
		LastIndex:   in.LastIndex,
		ServerTime:  in.ServerTime,
		Ready:       in.Ready,
	}, nil
}
