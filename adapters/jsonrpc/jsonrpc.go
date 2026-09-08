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
	apiquery "github.com/faustbrian/go-api-query"
	legacy "github.com/faustbrian/go-api-query/apiqueryrpc"
)

// ErrInvalid is the sanitized JSON-RPC parameter failure.
var ErrInvalid = legacy.ErrInvalid

// Params preserves absent versus explicitly empty JSON-RPC components.
type Params struct {
	SchemaRevision *string               `json:"schema_revision,omitempty"`
	Fields         *[]string             `json:"fields,omitempty"`
	Includes       *[]string             `json:"includes,omitempty"`
	Filter         *apiquery.FilterExpr  `json:"filter,omitempty"`
	Sorts          *[]apiquery.SortTerm  `json:"sorts,omitempty"`
	Page           *apiquery.PageRequest `json:"page,omitempty"`
}

// Parse strictly decodes one bounded JSON-RPC parameter object.
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
	Name     string         `json:"name"`
	Required bool           `json:"required"`
	Schema   map[string]any `json:"schema"`
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
