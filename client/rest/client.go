// Package client is a client library for accessing a ledger's API.
//
// It provides append and read methods, as well as a method for checking server
// status. No state is held in the client so it's fine to make concurrent calls
// through the same client object.
package client

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/api/rest"
)

// Client is a ledger API client.
type Client struct {
	host    string
	options options
}

// New creates a new Client, talking to the ledger API found at host and with
// zero or more options changed from their defaults.
func New(host string, opt ...Option) *Client {
	c := Client{
		host:    host,
		options: defaultOptions,
	}
	for _, o := range opt {
		o(&c.options)
	}
	return &c
}

// genReadContextAndURL generates the context and URL for a read call.
func (c *Client) genReadContextAndURL(ctx context.Context, req *api.ReadRequest) (context.Context, string) {
	u, err := url.Parse(c.host)
	if err != nil {
		c.fatalf("Failed to parse host %q: %v", c.host, err)
	}
	u.Path += rest.URLPrefix
	u.Path += "/"
	u.Path += strconv.FormatInt(req.Index, 10)
	p := url.Values{}
	count := req.Count
	if count <= 0 {
		count = c.options.maxCount
	}
	p.Add("max_count", strconv.FormatInt(int64(count), 10))

	pollTimeout := c.options.pollTimeout
	deadline, ok := ctx.Deadline()
	if ok {
		// Calculate polling timeout from request deadline, setting aside half of it for network overhead.
		pollTimeout = deadline.Sub(time.Now()) / 2
	} else {
		// Set default timeout on context.
		ctx, _ = context.WithTimeout(ctx, c.options.pollTimeout+c.options.callTimeout)
	}
	p.Add("timeout", strconv.FormatInt(int64(pollTimeout), 10))
	u.RawQuery = p.Encode()

	return ctx, u.String()
}

// decodeAndVerifyNetworkSeed reads the received network seed from the header
// and compares verifies that it matches the expectation.
func decodeAndVerifyNetworkSeed(h http.Header, expected []byte) ([]byte, error) {
	seed, err := hex.DecodeString(h.Get(rest.SymbiontNetworkSeedHeader))
	if err != nil {
		return nil, fmt.Errorf("Failed to decode received network seed: %v", err)
	}
	if len(expected) > 0 && !bytes.Equal(expected, seed) {
		return nil, api.NetworkSeedMismatchError(seed)
	}
	return seed, nil
}

// newError creates an error based on HTTP status code.
func newError(code int, msg string, seed []byte) error {
	switch code {
	case http.StatusNotFound:
		return api.NotFoundError(msg)
	case http.StatusBadRequest:
		return api.BadRequestError(msg)
	case http.StatusPreconditionFailed:
		return api.NetworkSeedMismatchError(seed)
	case http.StatusInternalServerError:
		return api.ServerError(msg)
	default:
		return fmt.Errorf("Read request failed (%d): %s", code, msg)
	}
}

// ReadTransactions reads transactions from a ledger, starting at the provided
// index. If there's not yet any transaction at that index, but it's the next
// one up, the request will be held for up to the provided timeout before
// returning. If the requested index is further into the future an error is
// returned. If a network seed is provided, it will be checked against the
// ledger's, and an error returned in case of a mismatch.
//
// The function returns the server's network seed, an array of sequenced
// transactions with the index of the first matching the requested index, or
// potentially an error. If there's a network seed mismatch, the caller should,
// depending on use-case, report an error or start using the new network seed
// in future requests. Retries with a bad network seed will all fail.
func (c *Client) ReadTransactions(ctx context.Context, req *api.ReadRequest) (*api.ReadResult, error) {
	ctx, url := c.genReadContextAndURL(ctx, req)

	// Perform GET request.
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create GET request to %q: %v", c.host, err)
	}
	r = r.WithContext(ctx)
	r.Header.Add(rest.SymbiontNetworkSeedHeader, hex.EncodeToString(req.NetworkSeed))

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("Failed to send GET request to %q: %v", c.host, err)
	}
	defer resp.Body.Close()

	// Parse result.
	var res rest.ReadResult
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("Failed to decode response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		seed, _ := decodeAndVerifyNetworkSeed(resp.Header, nil)
		return nil, newError(resp.StatusCode, res.Error, seed)
	}
	seed, err := decodeAndVerifyNetworkSeed(resp.Header, req.NetworkSeed)
	if err != nil {
		return nil, err
	}
	// Verify returned seed and index, and decode received transactions.
	if res.FirstIndex != req.Index {
		return nil, fmt.Errorf("Unexpected \"first_index\" (got %d, expected %d)",
			res.FirstIndex, req.Index)
	}
	txs, err := rest.DecodeSequencedTransactions(res.Transactions)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode transactions: %v", err)
	}
	if len(txs) > 0 {
		if txs[0].Index != req.Index {
			return nil, fmt.Errorf("Unexpected index of first tx (got %d, expected %d)",
				txs[0].Index, req.Index)
		}
	}
	return &api.ReadResult{seed, txs}, nil
}

// genAppendContextAndURL generates the context and URL for an append call.
func (c *Client) genAppendContextAndURL(ctx context.Context) (context.Context, string) {
	u, err := url.Parse(c.host)
	if err != nil {
		c.fatalf("Failed to parse host %q: %v", c.host, err)
	}
	u.Path += rest.URLPrefix

	// Set default timeout if none is provided.
	_, ok := ctx.Deadline()
	if !ok {
		ctx, _ = context.WithTimeout(ctx, c.options.appendTimeout+c.options.callTimeout)
	}

	return ctx, u.String()
}

// AppendTransactions appends an array of transactions to the ledger. All
// provided transactions will be ordered after all transactions already in the
// ledger, but no particular order respective to each other is guaranteed. If a
// network seed is provided this will be checked against the ledger's and
// requests with a mismatching network seed will be rejected. Both successful
// requests and those rejected due to seed mismatch will return the server's
// network seed.
func (c *Client) AppendTransactions(ctx context.Context, req *api.AppendRequest) (*api.AppendResult, error) {
	// Encode transactions and calculate their hashes to protect against corruption.
	data, err := rest.EncodeAppendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to encode request: %v", err)
	}
	ctx, url := c.genAppendContextAndURL(ctx)

	// Post encoded transactions to the ledger.
	r, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Failed to create POST request to %q: %v", c.host, err)
	}
	r = r.WithContext(ctx)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add(rest.SymbiontNetworkSeedHeader, hex.EncodeToString(req.NetworkSeed))

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("Failed to send POST request to %q: %v", c.host, err)
	}
	defer resp.Body.Close()

	// Decode and check result.
	res := rest.AppendResult{}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("Failed to decode response (code %d): %v", resp.StatusCode, err)
	}
	seed, err := decodeAndVerifyNetworkSeed(resp.Header, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode network seed in response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newError(resp.StatusCode, res.Error, seed)
	}
	return &api.AppendResult{seed, res.LastIndex}, nil
}

// ServerStatus return the status of the node the client is connected to.
func (c *Client) ServerStatus(ctx context.Context, _ *api.Empty) (*api.ServerStatusResult, error) {
	// Set default timeout if none is provided.
	_, ok := ctx.Deadline()
	if !ok {
		ctx, _ = context.WithTimeout(ctx, c.options.callTimeout)
	}

	// Perform request.
	client := &http.Client{}
	r, err := http.NewRequest("GET", c.host, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to create GET request to %q: %v", c.host, err)
	}
	r = r.WithContext(ctx)
	resp, err := client.Do(r)
	if err != nil {
		return nil, fmt.Errorf("Failed to send get request to %q: %v", c.host, err)
	}
	defer resp.Body.Close()

	// Decode and check result.
	dec := json.NewDecoder(resp.Body)
	var res rest.ServerStatusResult
	if err := dec.Decode(&res); err != nil {
		return nil, fmt.Errorf("Failed to decode response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newError(resp.StatusCode, res.Error, nil)
	}
	return rest.DecodeServerStatus(&res)
}
