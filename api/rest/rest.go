// Package rest provides a REST API for the ledger. Data is exchanged encoded
// in JSON format.
package rest

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"net/http"
	"strconv"
	"time"

	"github.com/symbiont-io/assembly-sdk/api"
)

// URLPrefix is the prefix used for writing and reading transactions.
const URLPrefix = "/transactions"

// SymbiontNetworkSeedHeader is the name of the header used to hold the network
// seed. If this is set on request and don't match the one at the server, the
// request will be rejected. Responses will have the server's seed set on them.
const SymbiontNetworkSeedHeader = "Symbiont-Network-Seed"

// schemaDecoder is a HTTP parameter decoder from gorilla/schema.
var schemaDecoder = schema.NewDecoder()

// Server is a REST API server for the ledger api. It takes requests and
// forwards them to the provided ledger.
type Server struct {
	ledger  api.LedgerServer
	options options
	router  *mux.Router
}

// NewServer create a new Server backed by the provided ledger.
func NewServer(ledger api.LedgerServer, opt ...Option) *Server {
	s := &Server{
		ledger:  ledger,
		options: defaultOptions,
	}
	for _, o := range opt {
		o(&s.options)
	}

	r := mux.NewRouter()
	r.Methods("GET").Path(URLPrefix + "/{index:[0-9]+}").Handler(s.handler(s.readHandler))
	// Allow optional trailing slash on append requests.
	r.Methods("POST").Path(URLPrefix + `{_slash:\/?}`).Handler(s.handler(s.appendHandler))
	r.Methods("GET").Path("/").Handler(s.handler(s.statusHandler))
	s.router = r
	return s
}

// Router returns the http.Handler of the Server.
func (s *Server) Router() http.Handler {
	return s.router
}

type handleError struct {
	err  error
	msg  string
	code int
}

func (e handleError) Error() string {
	return fmt.Sprintf("%s: %v", e.msg, e.err)
}

// handler wraps a request handler with common logging and checks.
func (s *Server) handler(fn func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.infof("Handling request: %s %q", r.Method, r.URL.Path)

		w.Header().Add("Content-Type", "application/json")
		w.Header().Add("Symbiont-Ledger-Version", Version)

		err := fn(w, r)
		if err != nil {
			e, ok := err.(*handleError)
			if !ok {
				e = &handleError{err, "Failed to handle request", http.StatusInternalServerError}
			}
			w.WriteHeader(e.code)
			s.warnf("Status %d: %v", e.code, e)
			err := json.NewEncoder(w).Encode(&struct {
				Error string `json:"error"`
			}{e.Error()})
			if err != nil {
				// Fallback to plain-text error message.
				http.Error(w, fmt.Sprintf("Failed to encode message: %v", err),
					http.StatusInternalServerError)
			}
		}
	})
}

// readHandler parses read requests and forwards them to the ledger API.
func (s *Server) readHandler(w http.ResponseWriter, r *http.Request) error {
	// Parse read parameters.
	r.ParseForm()
	p := struct {
		MaxCount     int64         `schema:"max_count"`
		MetadataOnly bool          `schema:"metadata_only"`
		Timeout      time.Duration `schema:"timeout"`
	}{s.options.defaultCount, false, s.options.defaultPollTimeout}
	err := schemaDecoder.Decode(&p, r.Form)
	if err != nil {
		return &handleError{err, "Failed to decode form", http.StatusBadRequest}
	}
	seed, err := hex.DecodeString(r.Header.Get(SymbiontNetworkSeedHeader))
	if err != nil {
		return &handleError{err, "Failed to parse network seed", http.StatusBadRequest}
	}
	index, err := strconv.ParseInt(mux.Vars(r)["index"], 10, 64)
	if err != nil {
		return &handleError{err, "Failed to parse index", http.StatusBadRequest}
	}
	if p.MaxCount > s.options.maxCount {
		p.MaxCount = s.options.maxCount
	}

	// Perform read request with timeout set. This timeout is shorter than the
	// HTTP request timeout, allowing us to send an empty response rather than
	// just timing out.
	ctx, cancel := s.options.contextWithTimeout(r.Context(), p.Timeout)
	defer cancel()

	res, err := s.ledger.ReadTransactions(ctx, &api.ReadRequest{seed, index, p.MaxCount})
	if err != nil {
		switch err := err.(type) {
		case api.BadRequestError:
			return &handleError{err, "Bad request", http.StatusBadRequest}
		case api.NotFoundError:
			return &handleError{err, "Transaction not found", http.StatusNotFound}
		case api.NetworkSeedMismatchError:
			w.Header().Add(SymbiontNetworkSeedHeader, hex.EncodeToString(err.CorrectSeed()))
			return &handleError{err, "Network seed mismatch", http.StatusPreconditionFailed}
		default:
			return err
		}
	}

	// Format read response.
	limit := index + int64(len(res.Transactions))
	out := ReadResult{
		FirstIndex: index,
		LastIndex:  limit - 1,
	}
	if !p.MetadataOnly {
		out.Transactions = EncodeSequencedTransactions(res.Transactions)
	}
	s.infof("Returning %d transactions, indexes [%d, %d)", len(res.Transactions), index, limit)
	w.Header().Add(SymbiontNetworkSeedHeader, hex.EncodeToString(res.NetworkSeed))
	return json.NewEncoder(w).Encode(&out)
}

// appendHandler parses append requests and forwards them to the ledger API.
func (s *Server) appendHandler(w http.ResponseWriter, r *http.Request) error {
	// Parse request and parameters.
	r.ParseForm()
	p := struct{ Async bool }{}
	err := schemaDecoder.Decode(&p, r.Form)
	if err != nil {
		return &handleError{err, "Failed to decode form", http.StatusBadRequest}
	}
	seed, err := hex.DecodeString(r.Header.Get(SymbiontNetworkSeedHeader))
	if err != nil {
		return &handleError{err, "Failed to parse network seed", http.StatusBadRequest}
	}

	req, err := DecodeAppendRequest(r.Body)
	if err != nil {
		return &handleError{err, "Failed to parse body", http.StatusBadRequest}
	}
	req.NetworkSeed = seed

	// Perform append request.
	s.debugf("Appending %d transactions", len(req.Transactions))
	if p.Async {
		// In the asynchronous case we don't wait for the append to complete.
		go s.ledger.AppendTransactions(r.Context(), &req) // ignore returned values.
		return json.NewEncoder(w).Encode(&AppendResult{Status: appendStatusPending})
	}
	res, err := s.ledger.AppendTransactions(r.Context(), &req)
	if err != nil {
		switch err := err.(type) {
		case api.BadRequestError:
			return &handleError{err, "Refused to append transactions", http.StatusBadRequest}
		case api.NetworkSeedMismatchError:
			w.Header().Add(SymbiontNetworkSeedHeader, hex.EncodeToString(err.CorrectSeed()))
			return &handleError{err, "Network seed mismatch", http.StatusPreconditionFailed}
		default:
			return err
		}
	}

	// Format append response.
	s.infof("Transactions appended with indexes starting from %d", res.LastIndex)
	w.Header().Add(SymbiontNetworkSeedHeader, hex.EncodeToString(res.NetworkSeed))
	return json.NewEncoder(w).Encode(&AppendResult{
		LastIndex: res.LastIndex,
		Status:    appendStatusSequenced,
	})
}

// statusHandler handles status requests and response with the status of the
// local node.
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) error {
	status, err := s.ledger.ServerStatus(r.Context(), nil)
	if err != nil {
		return err
	}
	writeNetworkSeed(w, status.NetworkSeed)
	return json.NewEncoder(w).Encode(EncodeServerStatus(status))
}

func writeNetworkSeed(w http.ResponseWriter, seed []byte) {
	w.Header().Add(SymbiontNetworkSeedHeader, hex.EncodeToString(seed))
}
