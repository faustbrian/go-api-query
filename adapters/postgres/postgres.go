// Package apiquerypostgres compiles reviewed plans into allowlisted PostgreSQL
// statement fragments. Compiler snapshots caller-owned mappings and Compile
// returns caller-owned arguments while sharing the compatibility path's
// sanitized error identity. It is safe for concurrent use after construction
// and never owns SQL execution, connections, transactions, joins, or schemas.
//
//nolint:staticcheck // This successor facade intentionally delegates to the deprecated compatibility implementation.
package apiquerypostgres

//lint:file-ignore SA1019 This successor facade intentionally delegates to the supported compatibility implementation.

import (
	apiquery "github.com/faustbrian/go-api-query"
	legacy "github.com/faustbrian/go-api-query/apiquerypgx"
)

// ErrInvalid reports an incomplete or unsafe mapping.
var ErrInvalid = legacy.ErrInvalid

// Mapping binds public capability names to reviewed PostgreSQL identifiers.
type Mapping struct {
	Fields      map[string]string
	Filters     map[string]string
	Sorts       map[string]string
	Constraints map[string]string
}

// Compiler is an immutable allowlisted PostgreSQL primitive compiler.
type Compiler struct {
	legacy *legacy.Compiler
}

// QueryParts contains fragments for an application-owned SQL statement.
type QueryParts struct {
	Projection string
	Where      string
	OrderBy    string
	Arguments  []apiquery.Value
}

// NewCompiler validates and snapshots every mapped identifier.
func NewCompiler(mapping Mapping) (*Compiler, error) {
	compiler, err := legacy.NewCompiler(legacy.Mapping{
		Fields:      mapping.Fields,
		Filters:     mapping.Filters,
		Sorts:       mapping.Sorts,
		Constraints: mapping.Constraints,
	})
	if err != nil {
		return nil, err
	}
	return &Compiler{legacy: compiler}, nil
}

// Compile translates a reviewed plan to PostgreSQL statement primitives.
func (compiler *Compiler) Compile(plan *apiquery.Plan) (QueryParts, error) {
	if compiler == nil {
		return QueryParts{}, ErrInvalid
	}
	parts, err := compiler.legacy.Compile(plan)
	if err != nil {
		return QueryParts{}, err
	}
	return QueryParts{
		Projection: parts.Projection,
		Where:      parts.Where,
		OrderBy:    parts.OrderBy,
		Arguments:  append([]apiquery.Value(nil), parts.Arguments...),
	}, nil
}
