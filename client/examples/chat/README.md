# Example of simple chat application

This is an example of a simple chat application intended to demonstrate how
easy it is to use the ledger for publishing and timestamping messages.

Usage
-----
Running two clients in parallel will look like this:

Client A:
```
$ go run client/examples/chat/chat.go --name alice
hi
2016-10-11T12:50:08Z: alice: hi
2016-10-11T12:50:23Z: bob: hey
```

Client B:
```
$ go run client/examples/chat/chat.go --name bob
2016-10-11T12:50:08Z: alice: hi
hey
2016-10-11T12:50:23Z: bob: hey
```
