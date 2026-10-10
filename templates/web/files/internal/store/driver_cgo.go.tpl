// driver_cgo.go
//go:build cgo

package store

import _ "github.com/mattn/go-sqlite3"

const driverName = "sqlite3"
