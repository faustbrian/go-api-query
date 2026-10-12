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
	apiquery "github.com/faustbrian/go-api-query/v4"
	legacy "github.com/faustbrian/go-api-query/v4/apiquerypgx"
)

// ErrInvalid reports an incomplete or unsafe mapping.
var ErrInvalid = legacy.ErrInvalid

// Mapping binds public capability names to reviewed PostgreSQL identifiers.
type Mapping struct {
	// Fields maps execution field names to trusted identifiers.
	Fields map[string]string
	// Filters maps filter names to trusted identifiers.
	Filters map[string]string
	// Sorts maps sort names to trusted identifiers.
	Sorts map[string]string
	// Constraints maps mandatory server predicate names to trusted identifiers.
	Constraints map[string]string
}

// Compiler is an immutable allowlisted PostgreSQL primitive compiler.
type Compiler struct {
	legacy *legacy.Compiler
}

// QueryParts contains fragments for an application-owned SQL statement.
type QueryParts struct {
	// Projection is a comma-separated execution field list without a SELECT keyword.
	Projection string
	// Where is the mandatory-and-client predicate fragment without a WHERE keyword.
	Where string
	// OrderBy is the ordered sort fragment without an ORDER BY keyword.
	OrderBy string
	// Arguments is a caller-owned positional slice of typed values; convert to driver
	// types and bind in order.
	Arguments []apiquery.Value
}

// NewCompiler validates and snapshots every mapped identifier. Maps must not be
// mutated during construction. Empty mappings are valid until a plan requires
// one; this does not prove schema coverage or authorize data exposure.
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

// Compile translates a plan to projection, predicates, order and typed positional
// arguments. All mandatory constraints are AND-composed regardless of Protected.
// String pattern operands escape LIKE metacharacters; comparisons/nulls otherwise
// follow PostgreSQL semantics. No statement is executed and no LIMIT, OFFSET,
// cursor seek predicate, join or include fetch is supplied. Applications own
// complete SQL, driver conversion, response masking and pagination. Missing
// mappings return zero QueryParts and ErrInvalid, not usable partial fragments.
func (compiler *Compiler) Compile(plan *apiquery.Plan) (QueryParts, error) {
	if compiler == nil {
		return QueryParts{}, ErrInvalid
	}
	compatibilityCompiler := compiler.legacy
	if compatibilityCompiler == nil {
		compatibilityCompiler = new(legacy.Compiler)
	}
	parts, err := compatibilityCompiler.Compile(plan)
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
