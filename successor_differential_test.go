//nolint:staticcheck // These differential tests intentionally exercise deprecated compatibility facades.
package apiquery_test

//lint:file-ignore SA1019 These differential tests intentionally exercise deprecated compatibility facades.

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	apiquery "github.com/faustbrian/go-api-query"
	newhttp "github.com/faustbrian/go-api-query/adapters/http"
	newjsonapi "github.com/faustbrian/go-api-query/adapters/jsonapi"
	newjsonrpc "github.com/faustbrian/go-api-query/adapters/jsonrpc"
	newpostgres "github.com/faustbrian/go-api-query/adapters/postgres"
	newvalidation "github.com/faustbrian/go-api-query/adapters/validation"
	legacyhttp "github.com/faustbrian/go-api-query/apiqueryhttp"
	legacyjsonapi "github.com/faustbrian/go-api-query/apiqueryjsonapi"
	legacypostgres "github.com/faustbrian/go-api-query/apiquerypgx"
	legacyjsonrpc "github.com/faustbrian/go-api-query/apiqueryrpc"
	legacyvalidation "github.com/faustbrian/go-api-query/apiqueryvalidation"
	jsonapi "github.com/faustbrian/go-jsonapi"
	validation "github.com/faustbrian/go-validation"
)

func TestSuccessorSentinelsShareLegacyIdentity(t *testing.T) {
	t.Parallel()

	pairs := []struct {
		name         string
		legacy, next error
	}{
		{"http invalid", legacyhttp.ErrInvalid, newhttp.ErrInvalid},
		{"jsonapi invalid", legacyjsonapi.ErrInvalid, newjsonapi.ErrInvalid},
		{"jsonapi unsupported", legacyjsonapi.ErrUnsupported, newjsonapi.ErrUnsupported},
		{"postgres invalid", legacypostgres.ErrInvalid, newpostgres.ErrInvalid},
		{"jsonrpc invalid", legacyjsonrpc.ErrInvalid, newjsonrpc.ErrInvalid},
	}
	for _, pair := range pairs {
		t.Run(pair.name, func(t *testing.T) {
			t.Parallel()
			if pair.legacy != pair.next || !errors.Is(pair.legacy, pair.next) || //nolint:errorlint // Exact sentinel identity is the compatibility contract.
				!errors.Is(pair.next, pair.legacy) {
				t.Fatalf("legacy %p and successor %p do not share identity", pair.legacy, pair.next)
			}
		})
	}
}

func TestSuccessorNamedTypesOwnTheirPackageIdentity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, legacy, next, legacyPath, nextPath string
	}{
		{"jsonapi config", typeIdentity(legacyjsonapi.Config{}), typeIdentity(newjsonapi.Config{}), "github.com/faustbrian/go-api-query/apiqueryjsonapi.Config", "github.com/faustbrian/go-api-query/adapters/jsonapi.Config"},
		{"jsonapi filter decoder", typeIdentity(legacyjsonapi.FilterDecoder(nil)), typeIdentity(newjsonapi.FilterDecoder(nil)), "github.com/faustbrian/go-api-query/apiqueryjsonapi.FilterDecoder", "github.com/faustbrian/go-api-query/adapters/jsonapi.FilterDecoder"},
		{"jsonapi page decoder", typeIdentity(legacyjsonapi.PageDecoder(nil)), typeIdentity(newjsonapi.PageDecoder(nil)), "github.com/faustbrian/go-api-query/apiqueryjsonapi.PageDecoder", "github.com/faustbrian/go-api-query/adapters/jsonapi.PageDecoder"},
		{"postgres mapping", typeIdentity(legacypostgres.Mapping{}), typeIdentity(newpostgres.Mapping{}), "github.com/faustbrian/go-api-query/apiquerypgx.Mapping", "github.com/faustbrian/go-api-query/adapters/postgres.Mapping"},
		{"postgres compiler", typeIdentity(legacypostgres.Compiler{}), typeIdentity(newpostgres.Compiler{}), "github.com/faustbrian/go-api-query/apiquerypgx.Compiler", "github.com/faustbrian/go-api-query/adapters/postgres.Compiler"},
		{"postgres parts", typeIdentity(legacypostgres.QueryParts{}), typeIdentity(newpostgres.QueryParts{}), "github.com/faustbrian/go-api-query/apiquerypgx.QueryParts", "github.com/faustbrian/go-api-query/adapters/postgres.QueryParts"},
		{"jsonrpc params", typeIdentity(legacyjsonrpc.Params{}), typeIdentity(newjsonrpc.Params{}), "github.com/faustbrian/go-api-query/apiqueryrpc.Params", "github.com/faustbrian/go-api-query/adapters/jsonrpc.Params"},
		{"jsonrpc descriptor", typeIdentity(legacyjsonrpc.ContentDescriptor{}), typeIdentity(newjsonrpc.ContentDescriptor{}), "github.com/faustbrian/go-api-query/apiqueryrpc.ContentDescriptor", "github.com/faustbrian/go-api-query/adapters/jsonrpc.ContentDescriptor"},
	}
	for _, test := range tests {
		if test.legacy != test.legacyPath || test.next != test.nextPath || test.legacy == test.next {
			t.Fatalf("%s identities = %q and %q", test.name, test.legacy, test.next)
		}
	}
	panicMarker := errors.New("decoder panic")
	legacyPanic := capturePanic(func() {
		_, _ = legacyjsonapi.FromQuery(jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, legacyjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { panic(panicMarker) }})
	})
	newPanic := capturePanic(func() {
		_, _ = newjsonapi.FromQuery(jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, newjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { panic(panicMarker) }})
	})
	if legacyPanic != panicMarker || newPanic != panicMarker { //nolint:errorlint // Exact panic identity is the compatibility contract.
		t.Fatalf("callback panic = (%v, %v), want original marker", legacyPanic, newPanic)
	}
}

func TestHTTPAdapterSuccessorMatchesLegacyMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, raw string
		max       int
		wantErr   error
	}{
		{"complete", "schema_revision=v1&fields=id,status&include=customer&sort=-created_at,id&page%5Bmode%5D=cursor&page%5Bsize%5D=20&page%5Bbefore%5D=opaque", 2048, nil},
		{"empty components", "fields=&include=&sort=&page%5Bmode%5D=offset&page%5Boffset%5D=4", 200, nil},
		{"empty", "", 1, nil},
		{"exact byte limit", "fields=id", len("fields=id"), nil},
		{"disabled limit", "", 0, legacyhttp.ErrInvalid},
		{"excess bytes", "fields=id", 2, legacyhttp.ErrInvalid},
		{"invalid raw UTF8", string([]byte{'f', 'i', 'e', 'l', 'd', 's', '=', 0xff}), 100, legacyhttp.ErrInvalid},
		{"malformed encoding", "fields=%zz", 100, legacyhttp.ErrInvalid},
		{"semicolon", "fields=id;sort=id", 100, legacyhttp.ErrInvalid},
		{"unknown", "sql=drop", 100, legacyhttp.ErrInvalid},
		{"duplicate", "fields=id&fields=status", 100, legacyhttp.ErrInvalid},
		{"invalid decoded UTF8", "fields=%FF", 100, legacyhttp.ErrInvalid},
		{"empty list member", "fields=id,,status", 100, legacyhttp.ErrInvalid},
		{"empty include member", "include=customer,", 100, legacyhttp.ErrInvalid},
		{"invalid filter", "filter=%7B", 100, legacyhttp.ErrInvalid},
		{"empty sort name", "sort=-", 100, legacyhttp.ErrInvalid},
		{"invalid page size", "page%5Bsize%5D=x", 100, legacyhttp.ErrInvalid},
		{"invalid page offset", "page%5Boffset%5D=x", 100, legacyhttp.ErrInvalid},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			legacyRequest, legacyErr := legacyhttp.Parse(test.raw, test.max)
			newRequest, newErr := newhttp.Parse(test.raw, test.max)
			if !reflect.DeepEqual(legacyRequest, newRequest) ||
				legacyErr != test.wantErr || newErr != test.wantErr { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
				t.Fatalf("legacy=(%#v,%v) successor=(%#v,%v)", legacyRequest, legacyErr, newRequest, newErr)
			}
		})
	}
}

func TestHTTPAndJSONAPIReturnedCollectionsAreIndependent(t *testing.T) {
	t.Parallel()

	raw := "fields=id,status&include=customer&sort=-created_at,id"
	legacyHTTP, legacyErr := legacyhttp.Parse(raw, 256)
	newHTTP, newErr := newhttp.Parse(raw, 256)
	if legacyErr != nil || newErr != nil {
		t.Fatalf("HTTP parse: legacy=%v successor=%v", legacyErr, newErr)
	}
	mutateRequestCollections(t, &legacyHTTP)
	mutateRequestCollections(t, &newHTTP)
	legacyHTTPAgain, legacyErr := legacyhttp.Parse(raw, 256)
	newHTTPAgain, newErr := newhttp.Parse(raw, 256)
	if legacyErr != nil || newErr != nil || !reflect.DeepEqual(legacyHTTPAgain, newHTTPAgain) ||
		requestCollectionsContain(t, legacyHTTPAgain, "changed") || requestCollectionsContain(t, newHTTPAgain, "changed") {
		t.Fatalf("HTTP returned mutation was retained: legacy=%#v successor=%#v", legacyHTTPAgain, newHTTPAgain)
	}

	legacyQuery := completeJSONAPIQuery()
	newQuery := completeJSONAPIQuery()
	legacyJSONAPI, legacyErr := legacyjsonapi.FromQuery(legacyQuery, legacyJSONAPIConfig())
	newJSONAPI, newErr := newjsonapi.FromQuery(newQuery, successorJSONAPIConfig())
	if legacyErr != nil || newErr != nil {
		t.Fatalf("JSON:API compose: legacy=%v successor=%v", legacyErr, newErr)
	}
	mutateRequestCollections(t, &legacyJSONAPI)
	mutateRequestCollections(t, &newJSONAPI)
	legacyJSONAPIAgain, legacyErr := legacyjsonapi.FromQuery(legacyQuery, legacyJSONAPIConfig())
	newJSONAPIAgain, newErr := newjsonapi.FromQuery(newQuery, successorJSONAPIConfig())
	if legacyErr != nil || newErr != nil || !reflect.DeepEqual(legacyJSONAPIAgain, newJSONAPIAgain) ||
		requestCollectionsContain(t, legacyJSONAPIAgain, "changed") || requestCollectionsContain(t, newJSONAPIAgain, "changed") ||
		legacyQuery.Fields["orders"][0] == "changed" || newQuery.Fields["orders"][0] == "changed" ||
		legacyQuery.Include[0] == "changed" || newQuery.Include[0] == "changed" ||
		legacyQuery.Sort[0].Name == "changed" || newQuery.Sort[0].Name == "changed" {
		t.Fatalf("JSON:API returned mutation was retained: legacy=%#v successor=%#v", legacyJSONAPIAgain, newJSONAPIAgain)
	}
}

func TestJSONAPIAdapterSuccessorMatchesLegacyMatrix(t *testing.T) {
	t.Parallel()

	legacyOrder := []string{}
	newOrder := []string{}
	legacyQuery := completeJSONAPIQuery()
	newQuery := completeJSONAPIQuery()
	legacyRequest, legacyErr := legacyjsonapi.FromQuery(legacyQuery, legacyjsonapi.Config{
		Resource: "orders",
		DecodeFilter: func(family jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
			legacyOrder = append(legacyOrder, "filter")
			family["filter[status]"][0] = "mutated"
			return statusFilter("paid"), nil
		},
		DecodePage: func(family jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			legacyOrder = append(legacyOrder, "page")
			family["page[size]"][0] = "mutated"
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	})
	newRequest, newErr := newjsonapi.FromQuery(newQuery, newjsonapi.Config{
		Resource: "orders",
		DecodeFilter: func(family jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
			newOrder = append(newOrder, "filter")
			family["filter[status]"][0] = "mutated"
			return statusFilter("paid"), nil
		},
		DecodePage: func(family jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			newOrder = append(newOrder, "page")
			family["page[size]"][0] = "mutated"
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	})
	legacyFilter := legacyQuery.Filter["filter[status]"]
	newFilter := newQuery.Filter["filter[status]"]
	legacyPage := legacyQuery.Page["page[size]"]
	newPage := newQuery.Page["page[size]"]
	if legacyErr != nil || newErr != nil || !reflect.DeepEqual(legacyRequest, newRequest) ||
		!reflect.DeepEqual(legacyOrder, newOrder) ||
		len(legacyFilter) != 1 || len(newFilter) != 1 || legacyFilter[0] != "paid" || newFilter[0] != "paid" ||
		len(legacyPage) != 1 || len(newPage) != 1 || legacyPage[0] != "20" || newPage[0] != "20" {
		t.Fatalf("legacy=(%#v,%v,%v) successor=(%#v,%v,%v)", legacyRequest, legacyErr, legacyOrder, newRequest, newErr, newOrder)
	}

	failure := errors.New("private decoder error")
	tests := []struct {
		name    string
		query   jsonapi.Query
		legacy  legacyjsonapi.Config
		next    newjsonapi.Config
		wantErr error
	}{
		{"missing resource", jsonapi.Query{}, legacyjsonapi.Config{}, newjsonapi.Config{}, legacyjsonapi.ErrInvalid},
		{"missing filter", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, legacyjsonapi.Config{Resource: "orders"}, newjsonapi.Config{Resource: "orders"}, legacyjsonapi.ErrUnsupported},
		{"missing page", jsonapi.Query{Page: jsonapi.ParameterFamily{"page[x]": {"1"}}}, legacyjsonapi.Config{Resource: "orders"}, newjsonapi.Config{Resource: "orders"}, legacyjsonapi.ErrUnsupported},
		{"filter nil", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, legacyjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, nil }}, newjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, nil }}, legacyjsonapi.ErrInvalid},
		{"filter error", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, legacyjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, failure }}, newjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, failure }}, legacyjsonapi.ErrInvalid},
		{"page error", jsonapi.Query{Page: jsonapi.ParameterFamily{"page[x]": {"1"}}}, legacyjsonapi.Config{Resource: "orders", DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) { return apiquery.PageRequest{}, failure }}, newjsonapi.Config{Resource: "orders", DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) { return apiquery.PageRequest{}, failure }}, legacyjsonapi.ErrInvalid},
	}
	for _, test := range tests {
		_, legacyErr := legacyjsonapi.FromQuery(test.query, test.legacy)
		_, newErr := newjsonapi.FromQuery(test.query, test.next)
		if legacyErr != test.wantErr || newErr != test.wantErr { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
			t.Fatalf("%s: legacy=%v successor=%v", test.name, legacyErr, newErr)
		}
	}
}

func TestJSONAPIAdapterCallbackTerminationMatchesLegacy(t *testing.T) {
	t.Parallel()

	marker := errors.New("callback failure")
	tests := []struct {
		name, filter, page string
		wantOrder          []string
		wantErr            error
		wantPanic          any
	}{
		{name: "filter nil stops page", filter: "nil", wantOrder: []string{"filter"}, wantErr: legacyjsonapi.ErrInvalid},
		{name: "filter error stops page", filter: "error", wantOrder: []string{"filter"}, wantErr: legacyjsonapi.ErrInvalid},
		{name: "filter panic stops page", filter: "panic", wantOrder: []string{"filter"}, wantPanic: marker},
		{name: "page error after filter", page: "error", wantOrder: []string{"filter", "page"}, wantErr: legacyjsonapi.ErrInvalid},
		{name: "page panic after filter", page: "panic", wantOrder: []string{"filter", "page"}, wantPanic: marker},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			legacyOrder, legacyPanic, legacyErr := observeLegacyJSONAPICallbacks(test.filter, test.page, marker)
			newOrder, newPanic, newErr := observeSuccessorJSONAPICallbacks(test.filter, test.page, marker)
			if !reflect.DeepEqual(legacyOrder, test.wantOrder) || !reflect.DeepEqual(newOrder, test.wantOrder) ||
				legacyErr != test.wantErr || newErr != test.wantErr || //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
				legacyPanic != test.wantPanic || newPanic != test.wantPanic {
				t.Fatalf("legacy=(%v,%v,%v) successor=(%v,%v,%v)", legacyOrder, legacyErr, legacyPanic, newOrder, newErr, newPanic)
			}
		})
	}
}

func TestPostgresAdapterSuccessorMatchesLegacyMatrix(t *testing.T) {
	t.Parallel()

	legacyMapping := legacypostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}
	newMapping := newpostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}
	legacyCompiler, legacyErr := legacypostgres.NewCompiler(legacyMapping)
	newCompiler, newErr := newpostgres.NewCompiler(newMapping)
	if legacyErr != nil || newErr != nil {
		t.Fatalf("constructors: legacy=%v successor=%v", legacyErr, newErr)
	}
	legacyMapping.Fields["id"] = "changed.bad"
	newMapping.Fields["id"] = "changed.bad"
	legacyMapping.Filters["value"] = "changed.bad"
	newMapping.Filters["value"] = "changed.bad"
	legacyMapping.Sorts["id"] = "changed.bad"
	newMapping.Sorts["id"] = "changed.bad"
	legacyMapping.Constraints["tenant_id"] = "changed.bad"
	newMapping.Constraints["tenant_id"] = "changed.bad"
	completePlan := differentialCompletePlan(t)
	legacyComplete, legacyCompleteErr := legacyCompiler.Compile(completePlan)
	newComplete, newCompleteErr := newCompiler.Compile(completePlan)
	if legacyCompleteErr != nil || newCompleteErr != nil || legacyComplete.Projection != newComplete.Projection ||
		legacyComplete.Where != newComplete.Where || legacyComplete.OrderBy != newComplete.OrderBy ||
		!reflect.DeepEqual(legacyComplete.Arguments, newComplete.Arguments) {
		t.Fatalf("complete plan: legacy=(%#v,%v) successor=(%#v,%v)", legacyComplete, legacyCompleteErr, newComplete, newCompleteErr)
	}
	legacyArguments := append([]apiquery.Value(nil), legacyComplete.Arguments...)
	newArguments := append([]apiquery.Value(nil), newComplete.Arguments...)
	if len(legacyComplete.Arguments) == 0 || len(newComplete.Arguments) == 0 {
		t.Fatal("complete plan has no arguments to mutation-test")
	}
	legacyComplete.Arguments[0] = apiquery.StringValue("changed")
	newComplete.Arguments[0] = apiquery.StringValue("changed")
	legacyCompleteAgain, legacyCompleteErr := legacyCompiler.Compile(completePlan)
	newCompleteAgain, newCompleteErr := newCompiler.Compile(completePlan)
	if legacyCompleteErr != nil || newCompleteErr != nil ||
		!reflect.DeepEqual(legacyCompleteAgain.Arguments, legacyArguments) ||
		!reflect.DeepEqual(newCompleteAgain.Arguments, newArguments) {
		t.Fatalf("complete plan retained returned argument mutation: legacy=(%#v,%v) successor=(%#v,%v)", legacyCompleteAgain, legacyCompleteErr, newCompleteAgain, newCompleteErr)
	}

	for _, expression := range allOperatorExpressions() {
		plan := differentialPlan(t, expression)
		legacyParts, legacyCompileErr := legacyCompiler.Compile(plan)
		newParts, newCompileErr := newCompiler.Compile(plan)
		if legacyCompileErr != nil || newCompileErr != nil ||
			legacyParts.Projection != newParts.Projection || legacyParts.Where != newParts.Where ||
			legacyParts.OrderBy != newParts.OrderBy || !reflect.DeepEqual(legacyParts.Arguments, newParts.Arguments) {
			t.Fatalf("expression %#v: legacy=(%#v,%v) successor=(%#v,%v)", expression, legacyParts, legacyCompileErr, newParts, newCompileErr)
		}
	}
	plan := differentialPlan(t, statusFilter("a"))
	_, legacyErr = (*legacypostgres.Compiler)(nil).Compile(plan)
	_, newErr = (*newpostgres.Compiler)(nil).Compile(plan)
	if legacyErr != legacypostgres.ErrInvalid || newErr != newpostgres.ErrInvalid { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
		t.Fatalf("nil compiler: legacy=%v successor=%v", legacyErr, newErr)
	}
	_, legacyErr = legacyCompiler.Compile(nil)
	_, newErr = newCompiler.Compile(nil)
	if legacyErr != legacypostgres.ErrInvalid || newErr != newpostgres.ErrInvalid { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
		t.Fatalf("nil plan: legacy=%v successor=%v", legacyErr, newErr)
	}
	for _, identifier := range []string{"", ".id", "a.b.c.d", "1table.id", "table.bad-name", "table.id;drop"} {
		_, legacyErr = legacypostgres.NewCompiler(legacypostgres.Mapping{Fields: map[string]string{"id": identifier}})
		_, newErr = newpostgres.NewCompiler(newpostgres.Mapping{Fields: map[string]string{"id": identifier}})
		if legacyErr != legacypostgres.ErrInvalid || newErr != newpostgres.ErrInvalid { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
			t.Fatalf("identifier %q: legacy=%v successor=%v", identifier, legacyErr, newErr)
		}
	}
	legacyZero, legacyErr := legacypostgres.NewCompiler(legacypostgres.Mapping{})
	newZero, newErr := newpostgres.NewCompiler(newpostgres.Mapping{})
	if legacyErr != nil || newErr != nil {
		t.Fatalf("zero mappings: legacy=%v successor=%v", legacyErr, newErr)
	}
	_, legacyErr = legacyZero.Compile(plan)
	_, newErr = newZero.Compile(plan)
	if legacyErr != legacypostgres.ErrInvalid || newErr != newpostgres.ErrInvalid { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
		t.Fatalf("zero mapping compile: legacy=%v successor=%v", legacyErr, newErr)
	}
	_, legacyErr = legacypostgres.NewCompiler(legacypostgres.Mapping{Fields: map[string]string{"": "records.id"}})
	_, newErr = newpostgres.NewCompiler(newpostgres.Mapping{Fields: map[string]string{"": "records.id"}})
	if legacyErr != legacypostgres.ErrInvalid || newErr != newpostgres.ErrInvalid { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
		t.Fatalf("empty capability: legacy=%v successor=%v", legacyErr, newErr)
	}

	missingMappings := []struct {
		name   string
		legacy legacypostgres.Mapping
		next   newpostgres.Mapping
	}{
		{name: "field", legacy: legacypostgres.Mapping{Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}, next: newpostgres.Mapping{Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}},
		{name: "filter", legacy: legacypostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Sorts: map[string]string{"id": "records.id"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}, next: newpostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Sorts: map[string]string{"id": "records.id"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}},
		{name: "sort", legacy: legacypostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}, next: newpostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}},
		{name: "constraint", legacy: legacypostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}}, next: newpostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}}},
	}
	for _, missing := range missingMappings {
		legacyMissing, legacyConstructErr := legacypostgres.NewCompiler(missing.legacy)
		newMissing, newConstructErr := newpostgres.NewCompiler(missing.next)
		if legacyConstructErr != nil || newConstructErr != nil {
			t.Fatalf("missing %s constructors: legacy=%v successor=%v", missing.name, legacyConstructErr, newConstructErr)
		}
		_, legacyErr = legacyMissing.Compile(completePlan)
		_, newErr = newMissing.Compile(completePlan)
		if legacyErr != legacypostgres.ErrInvalid || newErr != newpostgres.ErrInvalid { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
			t.Fatalf("missing %s mapping: legacy=%v successor=%v", missing.name, legacyErr, newErr)
		}
	}
}

func TestJSONRPCAdapterSuccessorMatchesLegacyMatrix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
		max  int
		want error
	}{
		{"complete", []byte(`{"schema_revision":"v1","fields":[],"includes":["customer"],"filter":{"predicate":{"name":"status","operator":"eq","values":[{"type":"string","value":"paid"}]}},"sorts":[{"name":"id","direction":"asc"}],"page":{"mode":"cursor","size":10}}`), 1000, nil},
		{"absent", []byte(`{}`), 10, nil}, {"empty", []byte(`{"fields":[],"includes":[],"sorts":[]}`), 100, nil},
		{"exact byte limit", []byte(`{"fields":[]}`), len(`{"fields":[]}`), nil},
		{"nil", nil, 100, legacyjsonrpc.ErrInvalid}, {"array", []byte(`[]`), 100, legacyjsonrpc.ErrInvalid}, {"unknown", []byte(`{"unknown":1}`), 100, legacyjsonrpc.ErrInvalid},
		{"duplicate", []byte(`{"fields":[],"fields":[]}`), 100, legacyjsonrpc.ErrInvalid}, {"trailing", []byte(`{} {}`), 100, legacyjsonrpc.ErrInvalid},
		{"nested duplicate", []byte(`{"filter":{"predicate":{"name":"first","name":"second"}}}`), 100, legacyjsonrpc.ErrInvalid},
		{"excess depth", deepJSONBytes(65), 512, legacyjsonrpc.ErrInvalid},
		{"invalid UTF8", []byte{0xff, 0x00}, 100, legacyjsonrpc.ErrInvalid},
		{"disabled bound", []byte(`{}`), 0, legacyjsonrpc.ErrInvalid}, {"excess bound", []byte(`{}`), 1, legacyjsonrpc.ErrInvalid},
	}
	for _, test := range tests {
		legacyParams, legacyErr := legacyjsonrpc.Parse(test.data, test.max)
		newParams, newErr := newjsonrpc.Parse(test.data, test.max)
		if !reflect.DeepEqual(legacyParams.Request(), newParams.Request()) ||
			legacyErr != test.want || newErr != test.want { //nolint:errorlint // Exact returned sentinel identity is the compatibility contract.
			t.Fatalf("%s: legacy=(%#v,%v) successor=(%#v,%v)", test.name, legacyParams, legacyErr, newParams, newErr)
		}
	}
	legacyDescriptor := legacyjsonrpc.OpenRPCContentDescriptor()
	newDescriptor := newjsonrpc.OpenRPCContentDescriptor()
	if !reflect.DeepEqual(legacyDescriptor.Schema, newDescriptor.Schema) || legacyDescriptor.Name != newDescriptor.Name || legacyDescriptor.Required != newDescriptor.Required {
		t.Fatal("descriptor mismatch")
	}
	secondNew := newjsonrpc.OpenRPCContentDescriptor()
	newDescriptor.Schema["type"] = "changed"
	newProperties := newDescriptor.Schema["properties"].(map[string]any)
	newRevision := newProperties["schema_revision"].(map[string]any)
	newRevision["type"] = "changed"
	secondProperties := secondNew.Schema["properties"].(map[string]any)
	secondRevision := secondProperties["schema_revision"].(map[string]any)
	if secondNew.Schema["type"] != "object" || secondRevision["type"] != "string" {
		t.Fatal("successor descriptor retained mutable map")
	}
	complete, err := newjsonrpc.Parse([]byte(`{"fields":["id"],"filter":{"logic":"and","children":[{"predicate":{"name":"value","operator":"eq","values":[{"type":"string","value":"paid"}]}}]},"sorts":[{"name":"id","direction":"asc"}]}`), 512)
	if err != nil {
		t.Fatal(err)
	}
	first := complete.Request()
	firstFields, firstFieldsPresent := first.Fields.Value()
	firstSorts, firstSortsPresent := first.Sorts.Value()
	if !firstFieldsPresent || !firstSortsPresent || len(firstFields) != 1 || len(firstSorts) != 1 {
		t.Fatalf("first Request() = %#v", first)
	}
	firstFields[0] = "changed"
	firstSorts[0].Name = "changed"
	first.Filter.Children[0].Predicate.Values[0] = apiquery.StringValue("changed")
	second := complete.Request()
	secondFields, secondFieldsPresent := second.Fields.Value()
	secondSorts, secondSortsPresent := second.Sorts.Value()
	if !secondFieldsPresent || !secondSortsPresent || len(secondFields) != 1 || len(secondSorts) != 1 ||
		secondFields[0] != "id" || secondSorts[0].Name != "id" || second.Filter.Children[0].Predicate.Values[0].String() != "paid" {
		t.Fatalf("Request retained returned mutation: %#v", second)
	}
}

func TestValidationAdapterSuccessorMatchesLegacyMatrix(t *testing.T) {
	t.Parallel()

	limits := validation.DefaultLimits()
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "orders", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString}}})
	if err != nil {
		t.Fatal(err)
	}
	_, structured := apiquery.Compile(context.Background(), schema, apiquery.Request{Fields: apiquery.Present([]string{"unknown", "id", "id"})}, apiquery.CompileOptions{})
	nested := nestedValidationError(t)
	tests := []struct {
		name   string
		err    error
		limits validation.Limits
	}{{"nil", nil, limits}, {"generic", errors.New("unsafe details"), limits}, {"structured", structured, limits}, {"nested path", nested, limits}}
	truncated := limits
	truncated.MaxViolations = 1
	tests = append(tests, struct {
		name   string
		err    error
		limits validation.Limits
	}{"truncated", structured, truncated})
	for _, test := range tests {
		legacyReport := legacyvalidation.Report(test.err, test.limits)
		newReport := newvalidation.Report(test.err, test.limits)
		if test.name == "nested path" {
			legacyViolations := legacyReport.Violations()
			newViolations := newReport.Violations()
			if len(legacyViolations) != 1 || len(newViolations) != 1 ||
				legacyViolations[0].Path().String() != "filter.children[0]" ||
				newViolations[0].Path().String() != "filter.children[0]" ||
				legacyViolations[0].Code() != "invalid_element" || newViolations[0].Code() != "invalid_element" {
				t.Fatalf("nested reports: legacy=%#v successor=%#v", legacyViolations, newViolations)
			}
		}
		legacySnapshot := reportSnapshot(legacyReport)
		newSnapshot := reportSnapshot(newReport)
		legacyReturned := legacyReport.Violations()
		newReturned := newReport.Violations()
		if len(legacyReturned) > 0 {
			legacyReturned[0].Parameters()["changed"] = "yes"
			legacyReturned[0] = validation.NewViolation(validation.RootPath(), "changed", validation.Error, nil, nil)
		}
		if len(newReturned) > 0 {
			newReturned[0].Parameters()["changed"] = "yes"
			newReturned[0] = validation.NewViolation(validation.RootPath(), "changed", validation.Error, nil, nil)
		}
		if !reflect.DeepEqual(legacySnapshot, newSnapshot) || !reflect.DeepEqual(reportSnapshot(legacyReport), legacySnapshot) ||
			!reflect.DeepEqual(reportSnapshot(newReport), newSnapshot) || reportParametersContain(legacyReport, "changed") ||
			reportParametersContain(newReport, "changed") {
			t.Fatalf("%s: legacy=%#v successor=%#v", test.name, reportSnapshot(legacyReport), reportSnapshot(newReport))
		}
	}
}

func observeLegacyJSONAPICallbacks(filterBehavior, pageBehavior string, marker error) (order []string, panicValue any, resultErr error) {
	defer func() { panicValue = recover() }()
	_, resultErr = legacyjsonapi.FromQuery(completeJSONAPIQuery(), legacyjsonapi.Config{
		Resource: "orders",
		DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
			order = append(order, "filter")
			switch filterBehavior {
			case "nil":
				return nil, nil
			case "error":
				return nil, marker
			case "panic":
				panic(marker)
			default:
				return statusFilter("paid"), nil
			}
		},
		DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			order = append(order, "page")
			if pageBehavior == "error" {
				return apiquery.PageRequest{}, marker
			}
			if pageBehavior == "panic" {
				panic(marker)
			}
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	})
	return order, nil, resultErr
}

func observeSuccessorJSONAPICallbacks(filterBehavior, pageBehavior string, marker error) (order []string, panicValue any, resultErr error) {
	defer func() { panicValue = recover() }()
	_, resultErr = newjsonapi.FromQuery(completeJSONAPIQuery(), newjsonapi.Config{
		Resource: "orders",
		DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
			order = append(order, "filter")
			switch filterBehavior {
			case "nil":
				return nil, nil
			case "error":
				return nil, marker
			case "panic":
				panic(marker)
			default:
				return statusFilter("paid"), nil
			}
		},
		DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			order = append(order, "page")
			if pageBehavior == "error" {
				return apiquery.PageRequest{}, marker
			}
			if pageBehavior == "panic" {
				panic(marker)
			}
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	})
	return order, nil, resultErr
}

func legacyJSONAPIConfig() legacyjsonapi.Config {
	return legacyjsonapi.Config{
		Resource:     "orders",
		DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return statusFilter("paid"), nil },
		DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	}
}

func successorJSONAPIConfig() newjsonapi.Config {
	return newjsonapi.Config{
		Resource:     "orders",
		DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return statusFilter("paid"), nil },
		DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	}
}

func mutateRequestCollections(t *testing.T, request *apiquery.Request) {
	t.Helper()
	fields, fieldsPresent := request.Fields.Value()
	includes, includesPresent := request.Includes.Value()
	sorts, sortsPresent := request.Sorts.Value()
	if !fieldsPresent || !includesPresent || !sortsPresent || len(fields) == 0 || len(includes) == 0 || len(sorts) == 0 {
		t.Fatalf("request lacks mutable collections: %#v", request)
	}
	fields[0] = "changed"
	includes[0] = "changed"
	sorts[0].Name = "changed"
}

func requestCollectionsContain(t *testing.T, request apiquery.Request, value string) bool {
	t.Helper()
	fields, _ := request.Fields.Value()
	includes, _ := request.Includes.Value()
	sorts, _ := request.Sorts.Value()
	return len(fields) > 0 && fields[0] == value || len(includes) > 0 && includes[0] == value ||
		len(sorts) > 0 && sorts[0].Name == value
}

func deepJSONBytes(depth int) []byte {
	return []byte(`{"filter":` + strings.Repeat(`[`, depth) + `null` + strings.Repeat(`]`, depth) + `}`)
}

func nestedValidationError(t *testing.T) error {
	t.Helper()
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{
		Resource: "orders", Revision: "v1",
		Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString}},
		Filters: []apiquery.FilterDefinition{{Name: "status", Type: apiquery.TypeString,
			Operators: []apiquery.Operator{apiquery.OpEqual}}},
		AllowedLogic: []apiquery.Logic{apiquery.LogicAnd},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, compileErr := apiquery.Compile(context.Background(), schema, apiquery.Request{Filter: &apiquery.FilterExpr{
		Logic: apiquery.LogicAnd,
		Children: []apiquery.FilterExpr{{Predicate: &apiquery.Predicate{
			Name: "status", Operator: apiquery.OpEqual,
			Values: []apiquery.Value{apiquery.IntValue(1)},
		}}},
	}}, apiquery.CompileOptions{})
	if compileErr == nil {
		t.Fatal("nested invalid filter compiled without an error")
	}
	return compileErr
}

func reportParametersContain(report validation.Report, key string) bool {
	violations := report.Violations()
	return len(violations) > 0 && violations[0].Parameters()[key] != ""
}

func typeIdentity(value any) string {
	typeOf := reflect.TypeOf(value)
	return typeOf.PkgPath() + "." + typeOf.Name()
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func completeJSONAPIQuery() jsonapi.Query {
	return jsonapi.Query{Fields: map[string][]string{"orders": {"status"}}, IncludePresent: true, Include: []string{"customer.address"}, SortPresent: true, Sort: []jsonapi.SortField{{Name: "created_at", Descending: true}, {Name: "id"}}, Filter: jsonapi.ParameterFamily{"filter[status]": {"paid"}}, Page: jsonapi.ParameterFamily{"page[size]": {"20"}}}
}

func statusFilter(value string) *apiquery.FilterExpr {
	return &apiquery.FilterExpr{Predicate: &apiquery.Predicate{Name: "value", Operator: apiquery.OpEqual, Values: []apiquery.Value{apiquery.StringValue(value)}}}
}

func differentialPlan(t *testing.T, expression *apiquery.FilterExpr) *apiquery.Plan {
	t.Helper()
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "records", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString, Required: true}}, Filters: []apiquery.FilterDefinition{{Name: "value", Type: apiquery.TypeString, Nullable: true, AllowEmpty: true, Operators: []apiquery.Operator{apiquery.OpEqual, apiquery.OpNotEqual, apiquery.OpLess, apiquery.OpLessOrEqual, apiquery.OpGreater, apiquery.OpGreaterOrEqual, apiquery.OpIn, apiquery.OpNotIn, apiquery.OpBetween, apiquery.OpIsNull, apiquery.OpContains, apiquery.OpStartsWith, apiquery.OpEndsWith}}}, AllowedLogic: []apiquery.Logic{apiquery.LogicAnd, apiquery.LogicOr, apiquery.LogicNot}})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := apiquery.Compile(context.Background(), schema, apiquery.Request{Filter: expression}, apiquery.CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func differentialCompletePlan(t *testing.T) *apiquery.Plan {
	t.Helper()
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{
		Resource: "records", Revision: "v1",
		Fields:  []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString, Required: true}},
		Filters: []apiquery.FilterDefinition{{Name: "value", Type: apiquery.TypeString, Operators: []apiquery.Operator{apiquery.OpEqual}}},
		Sorts:   []apiquery.SortDefinition{{Name: "id", Type: apiquery.TypeString, TieBreaker: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := apiquery.Compile(context.Background(), schema, apiquery.Request{
		Filter: statusFilter("paid"),
		Sorts:  apiquery.Present([]apiquery.SortTerm{{Name: "id", Direction: apiquery.Ascending}}),
	}, apiquery.CompileOptions{MandatoryConstraints: []apiquery.Constraint{{Name: "tenant_id", Value: apiquery.StringValue("tenant-42"), Protected: true}}})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func capturePanic(operation func()) (recovered any) {
	defer func() { recovered = recover() }()
	operation()
	return nil
}

func allOperatorExpressions() []*apiquery.FilterExpr {
	leaf := func(operator apiquery.Operator, values ...string) *apiquery.FilterExpr {
		typed := make([]apiquery.Value, len(values))
		for index, value := range values {
			typed[index] = apiquery.StringValue(value)
		}
		return &apiquery.FilterExpr{Predicate: &apiquery.Predicate{Name: "value", Operator: operator, Values: typed}}
	}
	group := func(logic apiquery.Logic, children ...*apiquery.FilterExpr) *apiquery.FilterExpr {
		values := make([]apiquery.FilterExpr, len(children))
		for index, child := range children {
			values[index] = *child
		}
		return &apiquery.FilterExpr{Logic: logic, Children: values}
	}
	return []*apiquery.FilterExpr{leaf(apiquery.OpEqual, "a"), leaf(apiquery.OpNotEqual, "a"), leaf(apiquery.OpLess, "a"), leaf(apiquery.OpLessOrEqual, "a"), leaf(apiquery.OpGreater, "a"), leaf(apiquery.OpGreaterOrEqual, "a"), leaf(apiquery.OpIn, "a", "b"), leaf(apiquery.OpNotIn, "a", "b"), leaf(apiquery.OpBetween, "a", "z"), leaf(apiquery.OpIsNull), leaf(apiquery.OpContains, `a%b_c\d`), leaf(apiquery.OpStartsWith, "a"), leaf(apiquery.OpEndsWith, "z"), group(apiquery.LogicAnd, leaf(apiquery.OpGreater, "a"), leaf(apiquery.OpLess, "z")), group(apiquery.LogicOr, leaf(apiquery.OpEqual, "a"), leaf(apiquery.OpNotEqual, "z")), group(apiquery.LogicNot, leaf(apiquery.OpEqual, "a"))}
}

func reportSnapshot(report validation.Report) []string {
	result := []string{reflect.ValueOf(report.ContextError()).String(), errorText(report.Err())}
	for _, violation := range report.Violations() {
		result = append(result, violation.Path().String()+"|"+violation.Code()+"|"+violation.String()+"|"+errorText(violation.Cause()))
	}
	return append(result, boolText(report.Empty()), boolText(report.Truncated()), boolText(report.HasErrors()))
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
