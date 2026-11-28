// Package quack is a Go client for DuckDB's quack wire protocol, the quack
// extension's HTTP RPC: message encoding and decoding (wire.go,
// protocol.go), one Session per server-assigned connection (session.go),
// and SEND_DATA bulk loads for the driver's appenders (senddata.go,
// append.go, merge.go).
//
// It knows nothing about database/sql or the duckdb subprocess. The parent
// driver package supervises the process, bootstraps the listener, and adapts
// Sessions to database/sql, including restart-on-crash handling.
package quack
