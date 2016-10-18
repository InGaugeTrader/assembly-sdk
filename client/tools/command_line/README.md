# Command line client application

A simple client intended to aid development.

Usage examples
--------------
Publish 2 transactions to an empty ledger:
```
$ go run client/tools/command_line/client.go --verbose publish "tx1 data" "tx2 data" "tx3 data" "tx4 data"
2016/09/20 17:42:19 Call completed. Time spent 1.329849ms
2016/09/20 17:42:19 Index of last value sequenced is 4
```

Scan starting scanning from transaction with index 2:
```
$ go run client/tools/command_line/client.go scan 2
       2 (2016-09-20 15:42:19.887294756 +0000 UTC): tx2 data
       3 (2016-09-20 15:42:19.887294756 +0000 UTC): tx3 data
       4 (2016-09-20 15:42:19.887294756 +0000 UTC): tx4 data
```
