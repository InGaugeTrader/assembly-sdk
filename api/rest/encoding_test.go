package rest_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/api/rest"

	"github.com/nbio/st"
	"github.com/symbiont-io/assembly-sdk/test/utils"
	"testing"
)

func newEncodedSequencedTx(i, ts int64, data, prevSH []byte) (*rest.EncodedSequencedTransaction, []byte) {
	hash := sha256.Sum256(data) // type == ""
	sh := sha256.Sum256(append(prevSH, hash[:]...))
	return &rest.EncodedSequencedTransaction{
		Index:     i,
		Timestamp: ts,
		Data:      base64.StdEncoding.EncodeToString(data),
		Hash:      hex.EncodeToString(hash[:]),
		StateHash: hex.EncodeToString(sh[:]),
	}, sh[:]
}

func TestSequencedTransactionDecoding(t *testing.T) {
	v := []struct {
		data      []byte
		timestamp int64
	}{{
		[]byte("some text"),
		1473412023099563000, // UTC 2016-09-09T09:07:03.099563
	}, {
		[]byte("some other text"),
		1473412029138744600,
	}}
	var encoded []*rest.EncodedSequencedTransaction
	var stateHash []byte
	for i := 0; i < len(v); i++ {
		tx, newStateHash := newEncodedSequencedTx(int64(i+1), v[i].timestamp, v[i].data, stateHash)
		encoded = append(encoded, tx)
		stateHash = newStateHash
	}

	decoded, err := rest.DecodeSequencedTransactions(encoded)
	st.Assert(t, err, nil)

	st.Assert(t, len(decoded), 2)
	st.Expect(t, decoded[0].Index, int64(1))

	expectedTime, err := time.Parse(time.RFC3339, "2016-09-09T09:07:03.099563+00:00")
	st.Assert(t, err, nil)
	st.Expect(t, decoded[0].Timestamp, expectedTime.UnixNano())

	st.Expect(t, decoded[0].Data, v[0].data)

	decodedStateHash, err := hex.DecodeString(encoded[0].StateHash)
	st.Assert(t, err, nil)
	st.Expect(t, decoded[0].StateHash, decodedStateHash)

	st.Expect(t, decoded[1].Index, int64(2))
	st.Expect(t, decoded[1].Data, v[1].data)
}

func TestSequencedTransactionDecodingBadHash(t *testing.T) {
	tx, _ := newEncodedSequencedTx(int64(1), 1473412023099563000, []byte("some text"), nil)
	_, err := rest.DecodeSequencedTransactions([]*rest.EncodedSequencedTransaction{tx})
	st.Assert(t, err, nil)
	tx.Hash = "b94f6f125c79e3a5ffaa826f584c10d52ada669e6762051b826b55776d05aed3" // wrong hash
	_, err = rest.DecodeSequencedTransactions([]*rest.EncodedSequencedTransaction{tx})
	st.Refute(t, err, nil)
}

func TestSequencedTransactionEncodeDecode(t *testing.T) {
	data := utils.RandomData(100, 100)
	now := time.Now().UnixNano()

	var in []*api.SequencedTransaction
	for i, d := range data {
		hash := sha256.Sum256(d)
		in = append(in, &api.SequencedTransaction{
			Index:     int64(i),
			Timestamp: now + int64(i),
			Data:      d,
			Hash:      hash[:],
			StateHash: hash[:], // We don't care about state hash correctness at this level.
		})
	}

	encoded := rest.EncodeSequencedTransactions(in)
	decoded, err := rest.DecodeSequencedTransactions(encoded)
	st.Assert(t, err, nil)

	for i, tx := range decoded {
		st.Expect(t, tx.Index, int64(i))
		st.Expect(t, tx.Timestamp, now+int64(i))
		st.Expect(t, tx.Data, data[i])
	}
}

func newEncodedUnsequencedTx(data []byte) *rest.EncodedUnsequencedTransaction {
	hash := sha256.Sum256(data)
	return &rest.EncodedUnsequencedTransaction{
		Data: base64.StdEncoding.EncodeToString(data),
		Hash: hex.EncodeToString(hash[:]),
	}
}

func TestUnsequencedTransactionDecoding(t *testing.T) {
	data := utils.RandomData(50, 100)
	var encoded []*rest.EncodedUnsequencedTransaction
	for _, d := range data {
		encoded = append(encoded, newEncodedUnsequencedTx(d))
	}

	decoded, err := rest.DecodeUnsequencedTransactions(encoded)
	st.Assert(t, err, nil)

	st.Assert(t, len(decoded), 50)
	for i, d := range data {
		st.Expect(t, decoded[i].Data, d)
	}
}

func TestUnsequencedTransactionDecodingBadHash(t *testing.T) {
	n := 5
	for i := 0; i < n; i++ {
		data := utils.RandomData(n, 100)
		var encoded []*rest.EncodedUnsequencedTransaction
		for _, d := range data {
			encoded = append(encoded, newEncodedUnsequencedTx(d))
		}

		// Replace the hash of transaction i with random data.
		badHash := utils.RandomData(1, 100)
		encoded[i].Hash = hex.EncodeToString(badHash[0])

		_, err := rest.DecodeUnsequencedTransactions(encoded)
		st.Refute(t, err, nil)
	}
}

func TestUnhashedUnsequencedTransactionEncodeDecode(t *testing.T) {
	data := utils.RandomData(100, 100)

	var in []*api.UnsequencedTransaction
	for _, d := range data {
		in = append(in, &api.UnsequencedTransaction{Data: d})
	}

	encoded := rest.EncodeUnsequencedTransactions(in)
	decoded, err := rest.DecodeUnsequencedTransactions(encoded)
	st.Assert(t, err, nil)

	for i, tx := range decoded {
		st.Expect(t, tx.Data, data[i])
		hash := sha256.Sum256(data[i])
		st.Expect(t, tx.Hash, hash[:])
	}
}

func TestHashedUnsequencedTransactionEncodeDecode(t *testing.T) {
	data := utils.RandomData(100, 100)

	var in []*api.UnsequencedTransaction
	for _, d := range data {
		hash := sha256.Sum256(d)
		in = append(in, &api.UnsequencedTransaction{
			Data: d,
			Hash: hash[:],
		})
	}

	encoded := rest.EncodeUnsequencedTransactions(in)
	decoded, err := rest.DecodeUnsequencedTransactions(encoded)
	st.Assert(t, err, nil)

	for i, tx := range decoded {
		st.Expect(t, tx.Data, data[i])
	}
}

func TestServerStatusEncodeDecode(t *testing.T) {
	now := time.Now().UnixNano()
	seed := utils.RandomData(1, 100)
	in := &api.ServerStatusResult{seed[0], "test", 1234, now, true}

	encoded := rest.EncodeServerStatus(in)
	decoded, err := rest.DecodeServerStatus(encoded)
	st.Assert(t, err, nil)

	st.Expect(t, decoded.NetworkType, "test")
	st.Expect(t, decoded.NetworkSeed, seed[0])
	st.Expect(t, decoded.ServerTime, now)
	st.Expect(t, decoded.LastIndex, int64(1234))
	st.Expect(t, decoded.Ready, true)
}
