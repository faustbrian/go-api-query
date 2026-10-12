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
	apiquery "github.com/faustbrian/go-api-query/v4"
	legacy "github.com/faustbrian/go-api-query/v4/apiqueryjsonapi"
	jsonapi "github.com/faustbrian/go-jsonapi/v2"
)

var (
	// ErrInvalid reports an invalid bridge configuration or callback result.
	ErrInvalid = legacy.ErrInvalid
	// ErrUnsupported reports a present family without an application decoder.
	ErrUnsupported = legacy.ErrUnsupported
)

// FilterDecoder synchronously decodes a copied parameter family. Its returned
// expression is borrowed by the request, not cloned; callbacks own reentrancy
// and concurrent-call safety. Errors become ErrInvalid without their causes.
type FilterDecoder func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error)

// PageDecoder synchronously decodes a copied parameter family into page state.
// It owns profile semantics and callback safety; errors become ErrInvalid.
type PageDecoder func(jsonapi.ParameterFamily) (apiquery.PageRequest, error)

// Config binds one resource and its application-owned family decoders.
type Config struct {
	// Resource selects the sparse fieldset resource and must be nonempty.
	Resource string
	// DecodeFilter is required for a present filter family; the returned expression is
	// borrowed, not cloned.
	DecodeFilter FilterDecoder
	// DecodePage is required for a present page family; its returned page is copied by
	// value.
	DecodePage PageDecoder
}

// FromQuery converts an already parsed JSON:API query without reparsing protocol
// syntax. Only the configured sparse-fieldset resource is selected; other
// fieldsets are ignored. Filter/page families require their respective decoders.
// Returned selections are copied, but a decoder filter expression remains shared.
// Callbacks have no context parameter, forced cancellation or panic recovery.
func FromQuery(query jsonapi.Query, config Config) (apiquery.Request, error) {
	return legacy.FromQuery(query, legacy.Config{
		Resource:     config.Resource,
		DecodeFilter: legacy.FilterDecoder(config.DecodeFilter),
		DecodePage:   legacy.PageDecoder(config.DecodePage),
	})
}
