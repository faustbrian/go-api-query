// Package apiqueryhttp parses the repository's bounded conventional HTTP query
// representation. Parse is stateless, performs no I/O, retains no input, and
// returns the same sanitized error identity as the compatibility path. It does
// not own server limits, authorization, schema compilation, or HTTP handling.
//
//nolint:staticcheck // This successor facade intentionally delegates to the deprecated compatibility implementation.
package apiqueryhttp

//lint:file-ignore SA1019 This successor facade intentionally delegates to the supported compatibility implementation.

import (
	apiquery "github.com/faustbrian/go-api-query"
	legacy "github.com/faustbrian/go-api-query/apiqueryhttp"
)

// ErrInvalid is the sanitized HTTP query failure.
var ErrInvalid = legacy.ErrInvalid

// Parse validates and converts an HTTP query string.
func Parse(rawQuery string, maxBytes int) (apiquery.Request, error) {
	return legacy.Parse(rawQuery, maxBytes)
}
