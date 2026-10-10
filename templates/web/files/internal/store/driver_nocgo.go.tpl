// driver_nocgo.go
//go:build !cgo

package store

import _ "modernc.org/sqlite"

const driverName = "sqlite"
