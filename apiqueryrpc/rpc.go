// Package apiqueryrpc parses bounded JSON-RPC query parameters and describes
// their OpenRPC content without compiling or executing a query.
//
// Deprecated: use github.com/faustbrian/go-api-query/v4/adapters/jsonrpc. This package remains supported through the documented compatibility interval.
package apiqueryrpc

import (
	"errors"

	apiquery "github.com/faustbrian/go-api-query/v4"
	"github.com/faustbrian/go-api-query/v4/internal/strictjson"
)

// ErrInvalid is the sanitized JSON-RPC parameter failure.
var ErrInvalid = errors.New("API query RPC parameters are invalid")

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
	var params Params
	if err := strictjson.Decode(data, maxBytes, &params); err != nil {
		return Params{}, ErrInvalid
	}
	return params, nil
}

// Request returns a transport-neutral defensive snapshot.
func (p Params) Request() apiquery.Request {
	request := apiquery.Request{Filter: cloneFilter(p.Filter)}
	if p.SchemaRevision != nil {
		request.SchemaRevision = apiquery.Present(*p.SchemaRevision)
	}
	if p.Fields != nil {
		request.Fields = apiquery.Present(append([]string(nil), (*p.Fields)...))
	}
	if p.Includes != nil {
		request.Includes = apiquery.Present(append([]string(nil), (*p.Includes)...))
	}
	if p.Sorts != nil {
		request.Sorts = apiquery.Present(append([]apiquery.SortTerm(nil), (*p.Sorts)...))
	}
	if p.Page != nil {
		request.Page = *p.Page
	}
	return request
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

// OpenRPCContentDescriptor describes the query parameter object. The returned
// maps are newly allocated and safe for caller customization.
func OpenRPCContentDescriptor() ContentDescriptor {
	return ContentDescriptor{Name: "query", Required: false, Schema: map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"schema_revision": map[string]any{"type": "string"},
			"fields":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"includes":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"filter":          map[string]any{"type": "object"},
			"sorts":           map[string]any{"type": "array"},
			"page":            map[string]any{"type": "object"},
		},
	}}
}

func cloneFilter(filter *apiquery.FilterExpr) *apiquery.FilterExpr {
	if filter == nil {
		return nil
	}
	result := &apiquery.FilterExpr{Logic: filter.Logic}
	if filter.Predicate != nil {
		result.Predicate = &apiquery.Predicate{Name: filter.Predicate.Name,
			Operator: filter.Predicate.Operator,
			Values:   append([]apiquery.Value(nil), filter.Predicate.Values...)}
	}
	result.Children = make([]apiquery.FilterExpr, len(filter.Children))
	for index := range filter.Children {
		result.Children[index] = *cloneFilter(&filter.Children[index])
	}
	return result
}
