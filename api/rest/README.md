# ReSTful API for append-only distributed ledgers

The code in this folder (see [code outlay](#code-layout)) implements a RESTful
JSON-over-HTTP API for Assembly and other distributed ledgers with append-only
semantics.

## Read old or new transactions
### Request

`GET /transactions/<index:int>`
* `index` will normally be equal to `last_index` in the last response from the server + 1, or `1` in the initial request.
* Optional parameter: `max_count` <int> - max number of transactions to return.
* Optional parameter: `timeout` <int> - max time to wait for new transaction (in nanoseconds). If `timeout == 0`, the server will immediately send a (potentially empty) response.
* Optional parameter: `metadata_only` <bool> - actual transaction data is excluded from response (transaction array is empty).

(The server may have its own limits. Whichever is smaller is the one that will be used.)

Example: `GET /transactions/1?max_count=2`

### Response

If the requested index doesn't yet exist, the server is allowed to stall the request up to `timeout` before sending a response. If neither the requested index nor the index preceding it exist, the server will respond with an error.
The server may always send an empty response before the timeout has passed (perhaps due to a shorter server-side timeout).

Returns:
```
{
  "first_index": <int>
  "last_index": <int>,
  "transactions": [],
}
```

* `first_index` is equal to `index` in the request. If the response contains transactions, this will correspond to `tx_index` of the first returned transaction.
* `last_index` is the last index of the transactions returned here (`tx_index` of last transaction in response). If the response is empty, it will be equal to `index` in the request - 1.
* `transactions` is an array of transactions ordered by `tx_index`. The first transaction will have `tx_index == first_index`. The last will have `tx_index + 1 == next_index`. Not included in `metadata_only` responses (missing or empty array).

##### Transaction:
```
{
  "type": <string>,
  "tx_index": <int>,
  "timestamp": <int>,
  "data": <string:base64>,
  "hash": <string:hex>,
  "state_hash": <string:hex>,
}
```

* `type` is the type of the transaction. This is an arbitrary string set by the publisher.
* `tx_index` is the index the transaction has been assigned on the ledger. This is guaranteed to be the same on every node and never to change for a given transaction.
* `timestamp` is a timestamp assigned by the ledger. It is guaranteed to be the same on every node and never to change for a given transaction, as well as being non-decrementing with increasing `tx_index`. It's Unix Time with nanosecond resolution.
* `data` is the transaction itself, base64-encoded.
* `hash` is the hex-encoded SHA256 hash of the concatenation of `type` and the unencoded transaction data.
* `state_hash` is a hash based on this transaction and all previous ones and is intended to verify integrity of the server's database, as well as server and client implementation of this API. It's the hex-encoded SHA256 hash of the concatenation of the previous state hash and the hash of this transaction: `state_hash = hex_encode(sha256(previous_state_hash+hash))`, where `previous_state_hash` and `hash` are unencoded raw bytes. If this is the first transactions, `state_hash` is just the hex-encoded SHA256 hash of the unencoded `hash` field.

Example:
```
{
  "first_index": 1,
  "last_index": 2,
  "transactions": [
    {
      "type": "symbiont/example",
      "tx_index": 1,
      "timestamp": 1461614515676834000,
      "data": "dHgxIGRhdGE=",
      "hash": "a6aea047a8040359d315419484b62be02c3e481d985315245ef75597f77fdbfb",
      "state_hash": "2985804be2e6b1bd4454774e94a3d69fe2f88d3e5399a6a0906c7202f83bc8d6"
    },
    {
      "type": "symbiont/example",
      "tx_index": 2,
      "timestamp": 1461614515676834000,
      "data": "dHgyIGRhdGE=",
      "hash": "5998dd27ccd3b61afcac6e072370973a2768448df3124c1a4a4b2eee7aac55b6",
      "state_hash": "808dea6a1302434d66a7e0da0bb87d8d9e624630a48d3bad2c7d9a8db659a0eb"
    }
  ]
}
```

**Returns on error :**

Errors will have a HTTP status code different from `200`, as well as a descriptive error message in the body.

Possible status codes:
* `400 Bad Request` means there was an error with the request.
* `404 Not Found` means that the requested transaction was not found. This usually means that the `index` requested was too high.
* `412 Precondition Failed` means that this is a different ledger than the client was expecting, specifically the [Network Seed](#ledger-unique-network-seed) is not matching. The response will contain the server's seed in the `Symbiont-Network-Seed` header.
* `500 Internal Server Error` means that the server experienced an error. If retrying doesn't work, this should be reported.

Message body:
```
{
  "error": <string>
}
```
* `error` provides details about the error that occured.


## Publish / append new transactions
### Request

`POST /transactions`
* Optional parameter: `async` - Returns immediately, don't wait for transactions to be sequenced (assigned `tx_index`es)

One or more transactions can be sent in a request, held in an array in the body:
```
{
  "transactions": []
}
```

Transaction:
```
{
  "type": <string>,
  "data": <string:base64>,
  "hash": <string:hex>,
}
```
(same transaction format as in the [read API](#transaction), but only fields `type`, `data` and `hash` populated)

* If any hash mismatches are detected, the server will fail the whole request.
* No assumptions about timing and ordering, including relative to transactions from other sources, should be made (any such guarantees will be implementation specific).
* Duplicate transactions (`hash` equal to transaction seen before) may be rejected.

Example:
```
{
  "transactions": [
    {
      "type": "symbiont/example",
      "data": "dHgxIGRhdGE=", // "tx1 data"
      "hash": "a6aea047a8040359d315419484b62be02c3e481d985315245ef75597f77fdbfb"
    },
    {
      "type": "symbiont/example",
      "data": "dHgyIGRhdGE=", // "tx2 data"
      "hash": "5998dd27ccd3b61afcac6e072370973a2768448df3124c1a4a4b2eee7aac55b6"
    }
  ]
}
```

### Response

The response depends on the `async` parameter passed with the request. Servers are not required to support both modes. If an unsupported request is made, the server will fail with an appropriate error.

**Returns in `async` mode:**
```
{
  "status": "pending"
}
```
* Response will be almost instant.
* No assumptions can be made about the delay before transactions are permanently written to the ledger and sequenced, nor their order relative to simultaneous or following POST requests from this or other clients (any such guarantees will be implementation specific).
* The ledger does not guarantee that the transactions will ever be written. The client is responsible for retrying the request if some or all transactions are dropped.

**Returns in `sync` mode (no `async` parameter passed):**
```
{
  "status": "sequenced",
  "last_index": <int>
}
```
* No assumptions should be made about timing of this call (any such guarantees will be implementation specific).
* `last_index` is the highest `tx_index` assigned to any of the transactions in the request.
* All transactions in the request are permanently written to the ledger.
* Guarantees that all transactions in subsequent calls (from this or other clients) are sequenced after the ones in this request, with `tx_index`es strictly higher than `last_index`, and `timestamp`s being equal or higher.

**Returns on error :**

Errors will have a HTTP status code different from `200`, as well as a descriptive error message in the body.

Possible status codes:
* `400 Bad Request` means there was an error with the request.
* `412 Precondition Failed` means that this is a different ledger than the client was expecting, specifically the [Network Seed](#ledger-unique-network-seed) is not matching. The response will contain the server's seed in the `Symbiont-Network-Seed` header.
* `500 Internal Server Error` means that the server experienced an error. If retrying doesn't work, this should be reported.

```
{
  "error": <string>
}
```
* `error` provides details about the error that occured.

## Get server state
### Request

`GET /`

### Response

```
{
    "network_type": "mock",
    "network_seed": "3e6bed815e4604115d5f775b6157487022be80d93ba85217fb56529634f091f4",
    "last_index": 123,
    "server_time": 1473855891617613000,
    "ready": true,
    "version": "1.0.0"
}
```

* `network_type` is an arbitrary string describing the type of the ledger (eg. "production" or "testing").
* `network_seed` is a string identifying the ledger. It will always stay the same for a given ledger; if it has an unexpected value, the request was handled by a different (or potentially reset) ledger.
* `last_index`  is the last index written to the ledger. A low number indicates that the local node is behind the rest of the network.
* `server_time` is the time as seen by the local ledger node, in nanoseconds since Unix epoch.
* `ready` is a flag indicating if the local node deems itself ready to handle read and append requests. It can be false if the node is in the process of catching up to the rest of the network or is experiencing some other issue.
* `version` is the version of the ledger API.

**Returns on error :**

Errors will have a HTTP status code different from `200`, as well as a descriptive error message in the body.

Possible status codes:
* `500 Internal Server Error` means that the server experienced an error. If retrying doesn't work, this should be reported.

```
{
  "error": <string>
}
```
* `error` provides details about the error that occured.


## Ledger-unique network seed

Each ledger will have a unique seed associated with it. This is assigned when the ledger is created and is the same on every node. This has a double function, firstly it provides a check that a client is connected to the ledger it's expecting, and can allow either party to discard transactions not intended for it.

Secondly, it can allow non-production ledgers to reset their storage and communicate that to their clients, so that they can reset their own state and start reading transactions from the beginning.

The seed can be set in a HTTP header on both requests and responses, using the `Symbiont-Network-Seed` header. The semantics are as follows:

* If the header is present on a request, the ledger will verify the seed and reject requests with a unknown or outdated seed, returning a `417 Expectation Failed` status code.
* If the seed is not set on a request, the ledger will by-pass this check.
* Requests rejected for this reason (as well as successful ones) will have the correct seed set on the response.
* In production a client that has its request rejected will halt and report an error.
* In development systems a client may be configured to wipe its own state and start communicating with the ledger using the new seed.

Clients who wish to set this seed on their requests can obtain it by doing a server state request ('GET /') to the ledger.

## Code layout

* `encoding` handles encoding and decoding of the data structures being transmitted.
* `logging` provides short-hands to make logging more convenient.
* `options` defines options that can be provided when creating the API.
* `rest` is the RESTful API itself.
* `types` defines data structures used by the API.
* `version` defines the version of the API.
