package rest

//
// Read route (GET "/transactions/:index")
//

// ReadResult is the result of a read from the ledger.
type ReadResult struct {
	// FirstIndex is the index of the first transaction returned, or the index
	// used in the request if no transactions are returned.
	FirstIndex int64 `json:"first_index"`

	// Transactions is an array of transactions read from the ledger.
	Transactions []*EncodedSequencedTransaction `json:"transactions"`

	// LastIndex is the index of the last transaction returned, or one less
	// than the index in the request if no transactions are returned.
	LastIndex int64 `json:"last_index"`

	// Error is set if an error happened while executing the request.
	Error string `json:"error,omitempty"`
}

// EncodedSequencedTransaction is the encoded represetation of a transaction
// written to the ledger (sequenced).
type EncodedSequencedTransaction struct {
	// Type is the transaction type. This field allows clients to filter out
	// transactions they don't care about.
	Type string `json:"type"`

	// Index is the index assigned to the transaction when writing it to the
	// ledger. Uniquely identifies this transaction on the ledger.
	Index int64 `json:"tx_index"`

	// Timestamp is the time this transaction was written to the ledger, in
	// nanoseconds since the unix epoch. Monotonically increasing with Index.
	// Accuracy is implementation specific.
	Timestamp int64 `json:"timestamp"`

	// Data is the payload of the transaction, base64 encoded.
	Data string `json:"data"`

	// Hash is the SHA256 hash of the concatenation of Type and the unencoded
	// Data, hex encoded.
	Hash string `json:"hash"`

	// StateHash is the hex encoded SHA256 hash of the concatenation of the
	// previous state hash and the hash of this transaction (both as raw,
	// unencoded bytes). For the first transaction it will be simply its hash.
	// This provides paranoid clients with a mechanism for verifying ledger
	// correctness.
	//
	// `StateHash = hex.EncodeToString(
	//      sha256.Sum256(append(rawPreviousStateHash, rawHash...)))`
	StateHash string `json:"state_hash"`
}

//
// Append route (POST "/transactions/")
//

// AppendRequest is a request to append one or more transactions to the ledger.
type AppendRequest struct {
	// Transactions is an array of transactions to be written to the ledger in
	// an unspecified order.
	Transactions []*EncodedUnsequencedTransaction `json:"transactions"`
}

// AppendResult is the result of an append operation.
type AppendResult struct {
	// LastIndex is the index assigned to the transaction in the request
	// written to the ledger last. Will be 0 or absent for request with the
	// `async=true` flag.
	LastIndex int64 `json:"last_index,omitempty"`

	// Status is the status of the request. Will be `sequenced` for normal
	// requests and `pending` if `async=true` is set.
	Status string `json:"status"`

	// Error indicates that an error occured while executing the request.
	Error string `json:"error,omitempty"`
}

const (
	// appendStatusPending is the AppendResult.Status value for requests that
	// still haven't been sequenced.
	appendStatusPending = "pending"

	// appendStatusSequenced is the AppendResult.Status value for requests that
	// have been sequenced / written to the ledger.
	appendStatusSequenced = "sequenced"
)

// EncodedUnsequencedTransaction is an encoded unsequenced (not yet written to
// the ledger) transaction.
type EncodedUnsequencedTransaction struct {
	// Type is the transaction type. This field allows clients to filter out
	// transactions they don't care about. It can also be used to pass
	// administration messages to ledger nodes.
	Type string `json:"type"`

	// Data is the payload of the transaction, base64 encoded.
	Data string `json:"data"`

	// Hash is the SHA256 hash of the concatenation of Type and the unencoded
	// Data, hex encoded.
	Hash string `json:"hash"`
}

//
// Status route (GET "/")
//

// ServerStatus is the status of the local ledger node.
type ServerStatusResult struct {
	// NetworkType is an arbitrary string describing the type of the ledger
	// (eg. "production" or "testing").
	NetworkType string `json:"network_type"`

	// NetworkSeed is a random number identifying the ledger. It will always
	// stay the same for a given ledger; if it has a surprising value,
	// transactions were read from a different (or potentially reset) ledger.
	// Clients should send this with append requests and check it on read
	// results.
	NetworkSeed string `json:"network_seed"`

	// LastIndex is the last index written to the ledger. A low number
	// indicates that the local node is behind the rest of the network.
	LastIndex int64 `json:"last_index"`

	// ServerTime is the time as seen by the local ledger node, in nanoseconds
	// since unix-epoch.
	ServerTime int64 `json:"server_time"`

	// Ready is a flag indicating if the local node deems itself ready to
	// handle read and append requests. It can be false if the node is in the
	// process of catching up to the rest of the network or is experiencing
	// some other issue.
	Ready bool `json:"ready"`

	// Version indicates the version of the ledger API.
	Version string `json:"version"`

	// Error is set if an error happened while executing the request.
	Error string `json:"error,omitempty"`
}
