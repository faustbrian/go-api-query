// Package apiqueryjsonapi maps authoritative go-jsonapi query values into the
// transport-neutral request model. Calls are synchronous and stateless;
// callback inputs and returned slices preserve the compatibility path's copy
// boundaries. The package does not define JSON:API syntax, profiles, schema
// compilation, transport handling, or callback concurrency.
//
//nolint:staticcheck // This successor facade intentionally delegates to the deprecated compatibility implementation.
package apiqueryjsonapi

//lint:file-ignore SA1019 This successor facade intentionally delegates to the supported compatibility implementation.

import (
	apiquery "github.com/faustbrian/go-api-query"
	legacy "github.com/faustbrian/go-api-query/apiqueryjsonapi"
	jsonapi "github.com/faustbrian/go-jsonapi"
)

var (
	// ErrInvalid reports an invalid bridge configuration or callback result.
	ErrInvalid = legacy.ErrInvalid
	// ErrUnsupported reports a present family without an application decoder.
	ErrUnsupported = legacy.ErrUnsupported
)

// FilterDecoder decodes application-owned filter semantics.
type FilterDecoder func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error)

// PageDecoder decodes application-owned pagination semantics.
type PageDecoder func(jsonapi.ParameterFamily) (apiquery.PageRequest, error)

// Config binds one resource and its application-owned family decoders.
type Config struct {
	Resource     string
	DecodeFilter FilterDecoder
	DecodePage   PageDecoder
}

// FromQuery converts an already parsed JSON:API query.
func FromQuery(query jsonapi.Query, config Config) (apiquery.Request, error) {
	return legacy.FromQuery(query, legacy.Config{
		Resource:     config.Resource,
		DecodeFilter: legacy.FilterDecoder(config.DecodeFilter),
		DecodePage:   legacy.PageDecoder(config.DecodePage),
	})
}
