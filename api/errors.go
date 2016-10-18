// Errors returned by the ledger.
//
// Implements the `net.Error` interface, meaning that a client can easily
// determine if the error is a timeout or temporary error. The ledger doesn't
// time out (the empty response to a long poll isn't considered an error), but
// temporary errors can occur, in which case the client can retry the same
// request.
package api

import "fmt"

// BadRequestError is the error returned when an request is rejected by the
// ledger.
type BadRequestError string

func (e BadRequestError) Error() string   { return string(e) }
func (e BadRequestError) Timeout() bool   { return false }
func (e BadRequestError) Temporary() bool { return false }

// NotFoundError is the error returned when the requested transactions don't
// exist.
type NotFoundError string

func (e NotFoundError) Error() string   { return string(e) }
func (e NotFoundError) Timeout() bool   { return false }
func (e NotFoundError) Temporary() bool { return true }

// NetworkSeedMismatchError is the error returned when the network seed in a
// request doesn't match that of the server.
type NetworkSeedMismatchError []byte

func (e NetworkSeedMismatchError) Error() string {
	return fmt.Sprintf("Network seed mismatch, correct seed: %x", []byte(e))
}
func (e NetworkSeedMismatchError) Timeout() bool   { return false }
func (e NetworkSeedMismatchError) Temporary() bool { return false }

// CorrectSeed returns the network seed used by the ledger. Depending on the
// use-case, the client may choose to continue operating against the ledger, in
// which case this is the network seed it must use going forward.
func (e NetworkSeedMismatchError) CorrectSeed() []byte {
	return []byte(e)
}

// ServerError is a general class of server errors that are assumed to be
// temporary.
type ServerError string

func (e ServerError) Error() string {
	return string(e)
}
func (e ServerError) Timeout() bool   { return false }
func (e ServerError) Temporary() bool { return true }
