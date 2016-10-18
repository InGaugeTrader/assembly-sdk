# Distributed ledger mock implementation

Implements the Ledger interface and has the append only semantics of a real ledger, but that's it. There's no networking and thus no BFT. Storage is in memory and is wiped on restart.
