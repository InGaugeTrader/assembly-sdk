# Symbiont Assembly SDK

An SDK for Symbiont's distributed ledger: [Assembly](https://symbiont.io/technology/assembly)

This project provides a mock server that implements the [API of the full ledger](https://github.com/symbiont-io/assembly-sdk/tree/master/api/rest), but has no network, no BFT and no persistent storage. It's intended for demonstration and development purposes only.


Install
-------
Prerequisites:
* Go 1.7 or newer (https://golang.org/)
* Git (https://git-scm.com/)
* Godep for dependency handling: `$ go get github.com/tools/godep`

Download:
`$ go get github.com/symbiont-io/assembly-sdk`

Fetch dependencies:
`$ cd $GOPATH/src/github.com/symbiont-io/assembly-sdk`
`$ godep restore`

Run tests:
`$ go test ./...`


Usage
-----
`$ go run server.go` or `$ assembly-sdk`

The [API](https://github.com/symbiont-io/assembly-sdk/tree/master/api/rest) is by default exposed on port 4000 and only to local clients, but this can be changed with the `--listen` flag. Eg. `$ go run server.go --listen :4000` will make it available to everyone on your computer's network.

Code layout
-----------

* `api` - specification and implementation of the API.
* `client` - client libraries, tools and example applications.
* `mock` - mock implementation of a distributed ledger.
* `test` - integration tests and usage examples.
