package store

import "errors"

// Package-level errors for the store package.
var (
	// ErrUnsupportedDriver means the driver is not registered in the engine registry.
	ErrUnsupportedDriver = errors.New("unsupported engine")
	// ErrConnection indicates a failure establishing the connection.
	ErrConnection = errors.New("connection error")
	// ErrTableNotFound means the requested table does not exist in the schema.
	ErrTableNotFound = errors.New("table not found")
)
