// Package store provides SQLite storage access. The driver is selected automatically
// via build tags (see driver_cgo.go / driver_nocgo.go): github.com/mattn/go-sqlite3
// when CGO is enabled, modernc.org/sqlite otherwise. Business code is driver-agnostic.
package store

import "database/sql"

// DefaultDSN is the recommended DSN, using the _pragma parameter form common to both drivers.
const DefaultDSN = "file:app.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"

// Open opens a database connection with the automatically selected driver.
func Open(dsn string) (*sql.DB, error) {
	return sql.Open(driverName, dsn)
}
