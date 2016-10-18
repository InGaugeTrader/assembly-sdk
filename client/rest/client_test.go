package client_test

import (
	"encoding/hex"
	"fmt"
	"github.com/gorilla/schema"
	"golang.org/x/net/context"
	"net/http"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
	"github.com/symbiont-io/assembly-sdk/api/rest"
	"github.com/symbiont-io/assembly-sdk/client/rest"

	"github.com/nbio/st"
	"net/http/httptest"
	"testing"
)

var schemaDecoder = schema.NewDecoder()

type readParams struct {
	MaxCount int           `schema:"max_count"`
	Timeout  time.Duration `schema:"timeout"`
}

type mockReadServer struct {
	lastPath string
	lastSeed string
	params   readParams
}

func (m *mockReadServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.lastPath = r.URL.Path
	m.lastSeed = r.Header.Get(rest.SymbiontNetworkSeedHeader)

	r.ParseForm()
	err := schemaDecoder.Decode(&m.params, r.Form)
	if err != nil {
		panic(err.Error())
	}

	if m.params.MaxCount == 13 { // bad luck triggering bad request code path.
		http.Error(w, `{"error": "bad luck"}`, http.StatusBadRequest)
		return
	}

	w.Header().Add(rest.SymbiontNetworkSeedHeader, "736F6D652073656564")
	fmt.Fprintf(w, `
		{
			"first_index":1,
			"transactions":[
				{
					"tx_index":1,
					"timestamp":1473418802676551328,
					"data":"c29tZSB0ZXh0",
					"hash":"b94f6f125c79e3a5ffaa826f584c10d52ada669e6762051b826b55776d05aed2",
					"state_hash":"71a8e55edefe53f703646a679e66799cfef657b98474ff2e4148c3a1ea43169c"
				},
				{
					"tx_index":2,
					"timestamp":1473418805474658817,
					"data":"c29tZSBvdGhlciB0ZXh0",
					"hash":"c08f2787f00de82881987c0b79f48be91fd0e123a3a4fc74344eb84ea52623ae",
					"state_hash":"39d9bd47d14b48a7dab2764fdd53ea7577cfc1d02571cbfa77444abad13c1dde"
				}
			],
			"last_index":2
		}
	`)
}

func TestClientRead(t *testing.T) {
	m := mockReadServer{}
	s := httptest.NewServer(&m)
	c := client.New(s.URL)

	seed := []byte("some seed")
	res, err := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: seed,
		Index:       1,
	})
	st.Assert(t, err, nil)
	st.Expect(t, m.lastSeed, hex.EncodeToString(seed))
	st.Expect(t, m.lastPath, "/transactions/1")

	st.Assert(t, len(res.Transactions), 2)
	st.Expect(t, res.Transactions[0].Data, []byte("some text"))
	st.Expect(t, res.Transactions[1].Data, []byte("some other text"))
}

func TestClientReadBadRequest(t *testing.T) {
	m := mockReadServer{}
	s := httptest.NewServer(&m)
	c := client.New(s.URL, client.WithMaxCount(13))

	_, err := c.ReadTransactions(context.Background(), &api.ReadRequest{Index: 1})
	st.Refute(t, err, nil)
	_, reject := err.(api.BadRequestError)
	st.Assert(t, reject, true)
	st.Expect(t, err.Error(), "bad luck")
}

func TestClientReadWrongIndex(t *testing.T) {
	m := mockReadServer{}
	s := httptest.NewServer(&m)
	c := client.New(s.URL)

	_, err := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: []byte("some seed"),
		Index:       123,
	})
	st.Refute(t, err, nil)
	st.Expect(t, m.lastPath, "/transactions/123")
}

func TestClientReadBadSeed(t *testing.T) {
	s := httptest.NewServer(&mockBadReadServer{})
	c := client.New(s.URL)

	_, err := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: []byte("some other seed"),
		Index:       1,
	})
	st.Refute(t, err, nil)
	seed, ok := err.(api.NetworkSeedMismatchError)
	st.Assert(t, ok, true)
	st.Expect(t, []byte(seed), []byte("some seed"))
}

func TestClientParams(t *testing.T) {
	m := mockReadServer{}
	s := httptest.NewServer(&m)
	c := client.New(s.URL,
		client.WithPollTimeout(123*time.Second),
		client.WithMaxCount(42))

	_, err := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: []byte("some seed"),
		Index:       1,
	})
	st.Assert(t, err, nil)

	st.Expect(t, m.lastPath, "/transactions/1")
	st.Expect(t, m.params.Timeout, 123*time.Second)
	st.Expect(t, m.params.MaxCount, 42)
}

type mockBadReadServer struct{}

func (m *mockBadReadServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add(rest.SymbiontNetworkSeedHeader, "736F6D652073656564")
	fmt.Fprintf(w, `
			{
				"first_index":1,
				"transactions":[
					{
						"tx_index":1,
						"timestamp":1473418802676551328,
						"data":"c29tZSB0ZXh0",
						"hash":"b94f6f125c79e3a5ffaa826f584c10d52ada669e6762051b826b55776d05aed3",
						"state_hash":"71a8e55edefe53f703646a679e66799cfef657b98474ff2e4148c3a1ea43169c"
					}
				],
				"last_index":2
			}
		`)
}

func TestClientReadBadHash(t *testing.T) {
	s := httptest.NewServer(&mockBadReadServer{})
	c := client.New(s.URL)

	_, err := c.ReadTransactions(context.Background(), &api.ReadRequest{
		NetworkSeed: []byte("some seed"),
		Index:       1,
	})
	st.Refute(t, err, nil)
}

type appendParams struct {
	Async bool `schema:"async"`
}

type mockAppendServer struct {
	lastPath string
	lastSeed string
	params   appendParams
}

func (m *mockAppendServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.lastPath = r.URL.Path
	m.lastSeed = r.Header.Get(rest.SymbiontNetworkSeedHeader)
	if m.lastSeed != hex.EncodeToString([]byte("some seed")) {
		http.Error(w, `{"error":"seed mismatch"}`, http.StatusBadRequest)
		return
	}
	r.ParseForm()
	err := schemaDecoder.Decode(&m.params, r.Form)
	if err != nil {
		panic(err.Error())
	}

	w.Header().Add(rest.SymbiontNetworkSeedHeader, "736F6D652073656564")
	fmt.Fprintf(w, `
		{
			"last_index":2,
			"status":"sequenced"
		}
	`)
}

func TestClientAppend(t *testing.T) {
	m := mockAppendServer{}
	s := httptest.NewServer(&m)
	c := client.New(s.URL)

	seed := []byte("some seed")
	res, err := c.AppendTransactions(context.Background(), &api.AppendRequest{
		seed,
		[]*api.UnsequencedTransaction{
			&api.UnsequencedTransaction{Data: []byte("some text")},
			&api.UnsequencedTransaction{Data: []byte("some other text")},
		},
	})
	st.Assert(t, err, nil)

	st.Expect(t, res.LastIndex, int64(2))
	st.Expect(t, m.lastPath, "/transactions")
	st.Expect(t, m.lastSeed, hex.EncodeToString(seed))
}

func TestClientAppendBadSeed(t *testing.T) {
	m := mockAppendServer{}
	s := httptest.NewServer(&m)
	c := client.New(s.URL)

	seed := []byte("bad seed")
	_, err := c.AppendTransactions(context.Background(), &api.AppendRequest{
		seed,
		[]*api.UnsequencedTransaction{
			&api.UnsequencedTransaction{Data: []byte("some text")},
		},
	})
	st.Refute(t, err, nil)
	_, reject := err.(api.BadRequestError)
	st.Assert(t, reject, true)
	st.Expect(t, err.Error(), "seed mismatch")
}
