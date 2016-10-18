package rest_test

import (
	"bytes"
	"encoding/hex"
	"golang.org/x/net/context"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/api/rest"

	"github.com/nbio/st"
	"github.com/symbiont-io/assembly-sdk/test/utils"
	"net/http/httptest"
	"testing"
)

type dummyLedger struct {
	lastReq      *api.ReadRequest
	lastDeadline time.Time
}

func (l *dummyLedger) ReadTransactions(ctx context.Context, req *api.ReadRequest) (*api.ReadResult, error) {
	l.lastReq = req
	l.lastDeadline, _ = ctx.Deadline()

	if req.Index < 100 {
		return &api.ReadResult{[]byte("seed"), []*api.SequencedTransaction{
			utils.MockSequencedTransaction(req.Index),
			utils.MockSequencedTransaction(req.Index + 1),
		}}, nil
	} else {
		return nil, api.NotFoundError("Requested index is too far in the future")
	}
}

func (l *dummyLedger) AppendTransactions(_ context.Context, req *api.AppendRequest) (*api.AppendResult, error) {
	serverSeed := []byte("right seed")
	if bytes.Equal(req.NetworkSeed, serverSeed) {
		return &api.AppendResult{serverSeed, int64(234)}, nil
	} else {
		return nil, api.NetworkSeedMismatchError(serverSeed)
	}
}

func (l *dummyLedger) ServerStatus(_ context.Context, _ *api.Empty) (*api.ServerStatusResult, error) {
	return nil, nil
}

func TestServerBadPaths(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())
	defer ts.Close()

	// Get without index
	{
		u, _ := url.Parse(ts.URL)
		u.Path += rest.URLPrefix + "/"
		resp, err := http.Get(u.String())
		st.Assert(t, err, nil)

		st.Expect(t, resp.StatusCode, http.StatusNotFound)
		resp.Body.Close()
	}

	// Get with bad index
	{
		u, _ := url.Parse(ts.URL)
		u.Path += rest.URLPrefix + "/1a"
		resp, err := http.Get(u.String())
		st.Assert(t, err, nil)

		st.Expect(t, resp.StatusCode, http.StatusNotFound)
		resp.Body.Close()
	}

	// Arbitrary path
	{
		u, _ := url.Parse(ts.URL)
		u.Path += "/fhsdjk/"
		resp, err := http.Get(u.String())
		st.Assert(t, err, nil)

		st.Expect(t, resp.StatusCode, http.StatusNotFound)
		resp.Body.Close()
	}
}

func TestServerReadTransactionInFuture(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix + "/123"
	resp, err := http.Get(u.String())
	st.Assert(t, err, nil)

	st.Expect(t, resp.StatusCode, http.StatusNotFound)

	resp.Body.Close()
	ts.Close()

	st.Expect(t, m.lastReq.Index, int64(123))
}

func TestServerReadOK(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix + "/10"
	before := time.Now()
	resp, err := http.Get(u.String())
	st.Assert(t, err, nil)
	after := time.Now()

	st.Expect(t, resp.StatusCode, http.StatusOK)

	resp.Body.Close()
	ts.Close()

	st.Expect(t, m.lastReq.Index, int64(10))
	st.Expect(t, m.lastReq.Count, int64(rest.DefaultCount))
	st.Expect(t, m.lastDeadline.After(after.Add(rest.DefaultPollTimeout)), false)
	st.Expect(t, m.lastDeadline.Before(before.Add(rest.DefaultPollTimeout)), false)
}

func TestServerReadParams(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix + "/1"
	p := url.Values{}
	p.Add("max_count", "42")
	p.Add("timeout", "12345")
	u.RawQuery = p.Encode()

	before := time.Now()
	resp, err := http.Get(u.String())
	st.Assert(t, err, nil)
	after := time.Now()

	st.Expect(t, resp.StatusCode, http.StatusOK)

	resp.Body.Close()
	ts.Close()

	st.Expect(t, m.lastReq.Index, int64(1))
	st.Expect(t, m.lastReq.Count, int64(42))
	st.Expect(t, m.lastDeadline.After(after.Add(time.Duration(12345))), false)
	st.Expect(t, m.lastDeadline.Before(before.Add(time.Duration(12345))), false)
}

func TestServerAppendOK(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())
	defer ts.Close()

	data := bytes.NewBufferString(`
		{
			"transactions":[
				{
					"data":"QQ==",
					"hash":"559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd"
				}
			]
		}`)

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix

	req, err := http.NewRequest("POST", u.String(), data)
	st.Assert(t, err, nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(rest.SymbiontNetworkSeedHeader, "72696768742073656564")

	client := &http.Client{}
	resp, err := client.Do(req)
	st.Assert(t, err, nil)
	defer resp.Body.Close()

	st.Expect(t, resp.StatusCode, http.StatusOK)
	msg, _ := ioutil.ReadAll(resp.Body)
	st.Expect(t, strings.TrimSpace(string(msg)),
		`{"last_index":234,"status":"sequenced"}`)
}

func TestServerAppendBadHash(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())
	defer ts.Close()

	data := bytes.NewBufferString(`
		{
			"transactions":[
				{
					"data":"QQ==",
					"hash":"559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffe"
				}
			]
		}`)

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix
	resp, err := http.Post(u.String(), "application/json", data)
	st.Assert(t, err, nil)
	defer resp.Body.Close()

	st.Expect(t, resp.StatusCode, http.StatusBadRequest)
}

func TestServerAppendBadSeed(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())
	defer ts.Close()

	data := bytes.NewBufferString(`
		{
			"transactions":[
				{
					"data":"QQ==",
					"hash":"559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd"
				}
			]
		}`)

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix
	req, err := http.NewRequest("POST", u.String(), data)
	st.Assert(t, err, nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(rest.SymbiontNetworkSeedHeader, "72696768742073656563")

	client := &http.Client{}
	resp, err := client.Do(req)
	st.Assert(t, err, nil)
	defer resp.Body.Close()

	st.Expect(t, resp.StatusCode, http.StatusPreconditionFailed)
	st.Expect(t, resp.Header.Get(rest.SymbiontNetworkSeedHeader), hex.EncodeToString([]byte("right seed")))
}

func TestServerAppendUnparseableSeed(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())
	defer ts.Close()

	data := bytes.NewBufferString(`
		{
			"transactions":[
				{
					"data":"QQ==",
					"hash":"559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd"
				}
			]
		}`)

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix
	req, err := http.NewRequest("POST", u.String(), data)
	st.Assert(t, err, nil)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(rest.SymbiontNetworkSeedHeader, "XX")

	client := &http.Client{}
	resp, err := client.Do(req)
	st.Assert(t, err, nil)
	defer resp.Body.Close()

	st.Expect(t, resp.StatusCode, http.StatusBadRequest)
}

func TestServerAppendBadBase64(t *testing.T) {
	m := dummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())
	defer ts.Close()

	data := bytes.NewBufferString(`
		{
			"transactions":[
				{
					"data":"QQ%=",
					"hash":"559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd"
				}
			]
		}`)

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix
	resp, err := http.Post(u.String(), "application/json", data)
	st.Assert(t, err, nil)
	defer resp.Body.Close()

	st.Expect(t, resp.StatusCode, http.StatusBadRequest)
}

type badDummyLedger struct {
}

func (l *badDummyLedger) ReadTransactions(_ context.Context, _ *api.ReadRequest) (*api.ReadResult, error) {
	return nil, nil
}

func (l *badDummyLedger) AppendTransactions(_ context.Context, _ *api.AppendRequest) (*api.AppendResult, error) {
	var block chan struct{}
	<-block // block forever reading from nil channel
	return nil, nil
}

func (l *badDummyLedger) ServerStatus(_ context.Context, _ *api.Empty) (*api.ServerStatusResult, error) {
	return nil, nil
}

func TestServerAsyncWrite(t *testing.T) {
	m := badDummyLedger{}
	ts := httptest.NewServer(rest.NewServer(&m).Router())
	defer ts.Close()

	data := bytes.NewBufferString(`
		{
			"transactions":[
				{
					"data":"QQ==",
					"hash":"559aead08264d5795d3909718cdd05abd49572e84fe55590eef31a88a08fdffd"
				}
			]
		}`)

	u, _ := url.Parse(ts.URL)
	u.Path += rest.URLPrefix
	p := url.Values{}
	p.Add("async", "true") // async requests will not wait for ledger append to complete.
	u.RawQuery = p.Encode()
	resp, err := http.Post(u.String(), "application/json", data)
	st.Assert(t, err, nil)
	defer resp.Body.Close()

	st.Expect(t, resp.StatusCode, http.StatusOK)
	msg, _ := ioutil.ReadAll(resp.Body)
	st.Expect(t, strings.TrimSpace(string(msg)), `{"status":"pending"}`)
}
