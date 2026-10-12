// Package apiqueryjsonrpc parses bounded JSON-RPC query parameters and exposes
// a caller-owned OpenRPC descriptor. Operations are stateless and safe for
// concurrent independent calls; returned requests and descriptor maps preserve
// compatibility copy boundaries. The package does not own methods, handlers,
// transport policy, authorization, schema compilation, or execution.
//
//nolint:staticcheck // This successor facade intentionally delegates to the deprecated compatibility implementation.
package apiqueryjsonrpc

//lint:file-ignore SA1019 This successor facade intentionally delegates to the supported compatibility implementation.

import (
	apiquery "github.com/faustbrian/go-api-query/v4"
	legacy "github.com/faustbrian/go-api-query/v4/apiqueryrpc"
)

// ErrInvalid is the sanitized JSON-RPC parameter failure.
var ErrInvalid = legacy.ErrInvalid

// Params preserves absent versus explicitly empty JSON-RPC components.
type Params struct {
	// SchemaRevision is nil when absent or JSON null.
	SchemaRevision *string `json:"schema_revision,omitempty"`
	// Fields preserves an explicit empty array; absent and JSON null both become nil.
	Fields *[]string `json:"fields,omitempty"`
	// Includes preserves an explicit empty array; absent and JSON null both become nil.
	Includes *[]string `json:"includes,omitempty"`
	// Filter is the parsed mutable expression, or nil for absent/null.
	Filter *apiquery.FilterExpr `json:"filter,omitempty"`
	// Sorts preserves an explicit empty array; absent and JSON null both become nil.
	Sorts *[]apiquery.SortTerm `json:"sorts,omitempty"`
	// Page is nil when absent or JSON null; a supplied zero page is distinct.
	Page *apiquery.PageRequest `json:"page,omitempty"`
}

// Parse decodes bounded JSON-RPC parameters with exact duplicate-name rejection
// and ordinary unknown-field rejection. Existing encoding/json limitations apply:
// null parameters decode as empty Params, null members collapse to absence, case
// aliases match field names, and invalid UTF-8 can be repaired. Value custom
// unmarshaling accepts nested extra members. Failure returns zero Params and
// ErrInvalid; semantic capabilities and pagination are validated by Compile.
func Parse(data []byte, maxBytes int) (Params, error) {
	params, err := legacy.Parse(data, maxBytes)
	if err != nil {
		return Params{}, err
	}
	return Params{
		SchemaRevision: params.SchemaRevision,
		Fields:         params.Fields,
		Includes:       params.Includes,
		Filter:         params.Filter,
		Sorts:          params.Sorts,
		Page:           params.Page,
	}, nil
}

// Request returns a transport-neutral defensive snapshot.
func (params Params) Request() apiquery.Request {
	return legacy.Params{
		SchemaRevision: params.SchemaRevision,
		Fields:         params.Fields,
		Includes:       params.Includes,
		Filter:         params.Filter,
		Sorts:          params.Sorts,
		Page:           params.Page,
	}.Request()
}

// ContentDescriptor is a minimal OpenRPC-compatible parameter descriptor.
type ContentDescriptor struct {
	// Name is the published parameter name, query.
	Name string `json:"name"`
	// Required is false for the optional query parameter.
	Required bool `json:"required"`
	// Schema is a fresh caller-owned minimal schema map, not a full capability or
	// authorization listing.
	Schema map[string]any `json:"schema"`
}

// OpenRPCContentDescriptor describes the query parameter object.
func OpenRPCContentDescriptor() ContentDescriptor {
	descriptor := legacy.OpenRPCContentDescriptor()
	return ContentDescriptor{
		Name:     descriptor.Name,
		Required: descriptor.Required,
		Schema:   descriptor.Schema,
	}
}
