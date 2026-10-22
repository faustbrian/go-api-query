package apiquery_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	apiquery "github.com/faustbrian/go-api-query/v5"
	apiqueryhttp "github.com/faustbrian/go-api-query/v5/adapters/http"
	apiqueryjsonapi "github.com/faustbrian/go-api-query/v5/adapters/jsonapi"
	apiqueryjsonrpc "github.com/faustbrian/go-api-query/v5/adapters/jsonrpc"
	apiquerypostgres "github.com/faustbrian/go-api-query/v5/adapters/postgres"
	apiqueryvalidation "github.com/faustbrian/go-api-query/v5/adapters/validation"
	jsonapi "github.com/faustbrian/go-jsonapi/v2"
	validation "github.com/faustbrian/go-validation/v3"
)

func TestHTTPAndJSONAPIReturnedCollectionsAreIndependent(t *testing.T) {
	t.Parallel()

	raw := "fields=id,status&include=customer&sort=-created_at,id"
	parsed, err := apiqueryhttp.Parse(raw, 256)
	if err != nil {
		t.Fatal(err)
	}
	mutateRequestCollections(t, &parsed)
	again, err := apiqueryhttp.Parse(raw, 256)
	if err != nil || requestCollectionsContain(t, again, "changed") {
		t.Fatalf("HTTP returned mutation was retained: %#v, %v", again, err)
	}

	query := completeJSONAPIQuery()
	composed, err := apiqueryjsonapi.FromQuery(query, jsonAPIConfig())
	if err != nil {
		t.Fatal(err)
	}
	mutateRequestCollections(t, &composed)
	recomposed, err := apiqueryjsonapi.FromQuery(query, jsonAPIConfig())
	if err != nil || requestCollectionsContain(t, recomposed, "changed") ||
		query.Fields["orders"][0] == "changed" || query.Include[0] == "changed" || query.Sort[0].Name == "changed" {
		t.Fatalf("JSON:API returned mutation was retained: %#v, %v, caller %#v", recomposed, err, query)
	}
}

func TestJSONAPICallbacksReceiveCopiesInOrder(t *testing.T) {
	t.Parallel()

	order := []string{}
	query := completeJSONAPIQuery()
	request, err := apiqueryjsonapi.FromQuery(query, apiqueryjsonapi.Config{
		Resource: "orders",
		DecodeFilter: func(family jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
			order = append(order, "filter")
			family["filter[status]"][0] = "mutated"
			return statusFilter("paid"), nil
		},
		DecodePage: func(family jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			order = append(order, "page")
			family["page[size]"][0] = "mutated"
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	})
	if err != nil || !reflect.DeepEqual(order, []string{"filter", "page"}) ||
		!reflect.DeepEqual(request.Filter, statusFilter("paid")) ||
		request.Page != (apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}) ||
		query.Filter["filter[status]"][0] != "paid" || query.Page["page[size]"][0] != "20" {
		t.Fatalf("FromQuery() = %#v, %v, order %v, caller %#v", request, err, order, query)
	}

	failure := errors.New("private decoder error")
	for _, test := range []struct {
		name    string
		query   jsonapi.Query
		config  apiqueryjsonapi.Config
		wantErr error
	}{
		{"missing resource", jsonapi.Query{}, apiqueryjsonapi.Config{}, apiqueryjsonapi.ErrInvalid},
		{"missing filter", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders"}, apiqueryjsonapi.ErrUnsupported},
		{"missing page", jsonapi.Query{Page: jsonapi.ParameterFamily{"page[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders"}, apiqueryjsonapi.ErrUnsupported},
		{"filter nil", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, nil }}, apiqueryjsonapi.ErrInvalid},
		{"filter error", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, failure }}, apiqueryjsonapi.ErrInvalid},
		{"page error", jsonapi.Query{Page: jsonapi.ParameterFamily{"page[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders", DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) { return apiquery.PageRequest{}, failure }}, apiqueryjsonapi.ErrInvalid},
	} {
		if _, err := apiqueryjsonapi.FromQuery(test.query, test.config); err != test.wantErr { //nolint:errorlint // The exact returned sentinel is the contract.
			t.Fatalf("%s: error = %v, want %v", test.name, err, test.wantErr)
		}
	}
}

func TestJSONAPICallbackTermination(t *testing.T) {
	t.Parallel()

	marker := errors.New("callback failure")
	for _, test := range []struct {
		name, filter, page string
		wantOrder          []string
		wantErr            error
		wantPanic          any
	}{
		{name: "filter nil stops page", filter: "nil", wantOrder: []string{"filter"}, wantErr: apiqueryjsonapi.ErrInvalid},
		{name: "filter error stops page", filter: "error", wantOrder: []string{"filter"}, wantErr: apiqueryjsonapi.ErrInvalid},
		{name: "filter panic stops page", filter: "panic", wantOrder: []string{"filter"}, wantPanic: marker},
		{name: "page error after filter", page: "error", wantOrder: []string{"filter", "page"}, wantErr: apiqueryjsonapi.ErrInvalid},
		{name: "page panic after filter", page: "panic", wantOrder: []string{"filter", "page"}, wantPanic: marker},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			order, recovered, err := observeJSONAPICallbacks(test.filter, test.page, marker)
			if !reflect.DeepEqual(order, test.wantOrder) || err != test.wantErr || recovered != test.wantPanic { //nolint:errorlint // The exact returned sentinel is the contract.
				t.Fatalf("order=%v err=%v panic=%v", order, err, recovered)
			}
		})
	}
}

func TestPostgresCompilerSnapshotsMappingAndReturnsFreshArguments(t *testing.T) {
	t.Parallel()

	mapping := func() apiquerypostgres.Mapping {
		return apiquerypostgres.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}, Constraints: map[string]string{"tenant_id": "records.tenant_id"}}
	}
	borrowed := mapping()
	compiler, err := apiquerypostgres.NewCompiler(borrowed)
	if err != nil {
		t.Fatal(err)
	}
	pristine, err := apiquerypostgres.NewCompiler(mapping())
	if err != nil {
		t.Fatal(err)
	}
	borrowed.Fields["id"] = "changed.bad"
	borrowed.Filters["value"] = "changed.bad"
	borrowed.Sorts["id"] = "changed.bad"
	borrowed.Constraints["tenant_id"] = "changed.bad"

	plan := completePostgresPlan(t)
	got, err := compiler.Compile(plan)
	if err != nil {
		t.Fatal(err)
	}
	want, err := pristine.Compile(plan)
	if err != nil || got.Projection != want.Projection || got.Where != want.Where || got.OrderBy != want.OrderBy ||
		!reflect.DeepEqual(got.Arguments, want.Arguments) ||
		strings.Contains(got.Projection+got.Where+got.OrderBy, "changed") || len(got.Arguments) == 0 {
		t.Fatalf("compiled after caller mapping mutation = %#v; pristine = %#v, %v", got, want, err)
	}
	arguments := append([]apiquery.Value(nil), got.Arguments...)
	got.Arguments[0] = apiquery.StringValue("changed")
	again, err := compiler.Compile(plan)
	if err != nil || !reflect.DeepEqual(again.Arguments, arguments) {
		t.Fatalf("Compile retained returned argument mutation: %#v, %v", again, err)
	}

	if _, err := (*apiquerypostgres.Compiler)(nil).Compile(plan); err != apiquerypostgres.ErrInvalid { //nolint:errorlint // The exact returned sentinel is the contract.
		t.Fatalf("nil compiler error = %v", err)
	}
	var zero apiquerypostgres.Compiler
	if parts, err := zero.Compile(plan); err != apiquerypostgres.ErrInvalid || !reflect.DeepEqual(parts, apiquerypostgres.QueryParts{}) { //nolint:errorlint // The exact returned sentinel is the contract.
		t.Fatalf("zero compiler = %#v, %v", parts, err)
	}
	if _, err := compiler.Compile(nil); err != apiquerypostgres.ErrInvalid { //nolint:errorlint // The exact returned sentinel is the contract.
		t.Fatalf("nil plan error = %v", err)
	}
	for _, identifier := range []string{"", ".id", "a.b.c.d", "1table.id", "table.bad-name", "table.id;drop"} {
		if _, err := apiquerypostgres.NewCompiler(apiquerypostgres.Mapping{Fields: map[string]string{"id": identifier}}); err != apiquerypostgres.ErrInvalid { //nolint:errorlint // The exact returned sentinel is the contract.
			t.Fatalf("identifier %q error = %v", identifier, err)
		}
	}
	if _, err := apiquerypostgres.NewCompiler(apiquerypostgres.Mapping{Fields: map[string]string{"": "records.id"}}); err != apiquerypostgres.ErrInvalid { //nolint:errorlint // The exact returned sentinel is the contract.
		t.Fatalf("empty capability error = %v", err)
	}
	for _, missing := range []string{"field", "filter", "sort", "constraint"} {
		partial := mapping()
		switch missing {
		case "field":
			partial.Fields = nil
		case "filter":
			partial.Filters = nil
		case "sort":
			partial.Sorts = nil
		case "constraint":
			partial.Constraints = nil
		}
		incomplete, err := apiquerypostgres.NewCompiler(partial)
		if err != nil {
			t.Fatalf("missing %s constructor error = %v", missing, err)
		}
		if _, err := incomplete.Compile(plan); err != apiquerypostgres.ErrInvalid { //nolint:errorlint // The exact returned sentinel is the contract.
			t.Fatalf("missing %s mapping error = %v", missing, err)
		}
	}
}

func TestJSONRPCParamsAndDescriptorAreIsolated(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		data []byte
		max  int
		want error
	}{
		{"complete", []byte(`{"schema_revision":"v1","fields":[],"includes":["customer"],"filter":{"predicate":{"name":"status","operator":"eq","values":[{"type":"string","value":"paid"}]}},"sorts":[{"name":"id","direction":"asc"}],"page":{"mode":"cursor","size":10}}`), 1000, nil},
		{"absent", []byte(`{}`), 10, nil}, {"empty", []byte(`{"fields":[],"includes":[],"sorts":[]}`), 100, nil},
		{"exact byte limit", []byte(`{"fields":[]}`), len(`{"fields":[]}`), nil},
		{"nil", nil, 100, apiqueryjsonrpc.ErrInvalid}, {"array", []byte(`[]`), 100, apiqueryjsonrpc.ErrInvalid}, {"unknown", []byte(`{"unknown":1}`), 100, apiqueryjsonrpc.ErrInvalid},
		{"duplicate", []byte(`{"fields":[],"fields":[]}`), 100, apiqueryjsonrpc.ErrInvalid}, {"trailing", []byte(`{} {}`), 100, apiqueryjsonrpc.ErrInvalid},
		{"nested duplicate", []byte(`{"filter":{"predicate":{"name":"first","name":"second"}}}`), 100, apiqueryjsonrpc.ErrInvalid},
		{"excess depth", deepJSONBytes(65), 512, apiqueryjsonrpc.ErrInvalid},
		{"invalid UTF8", []byte{0xff, 0x00}, 100, apiqueryjsonrpc.ErrInvalid},
		{"disabled bound", []byte(`{}`), 0, apiqueryjsonrpc.ErrInvalid}, {"excess bound", []byte(`{}`), 1, apiqueryjsonrpc.ErrInvalid},
	} {
		if _, err := apiqueryjsonrpc.Parse(test.data, test.max); err != test.want { //nolint:errorlint // The exact returned sentinel is the contract.
			t.Fatalf("%s: error = %v, want %v", test.name, err, test.want)
		}
	}

	descriptor := apiqueryjsonrpc.OpenRPCContentDescriptor()
	second := apiqueryjsonrpc.OpenRPCContentDescriptor()
	descriptor.Schema["type"] = "changed"
	descriptor.Schema["properties"].(map[string]any)["schema_revision"].(map[string]any)["type"] = "changed"
	if second.Schema["type"] != "object" || second.Schema["properties"].(map[string]any)["schema_revision"].(map[string]any)["type"] != "string" {
		t.Fatal("descriptor retained a mutable map")
	}

	complete, err := apiqueryjsonrpc.Parse([]byte(`{"fields":["id"],"filter":{"logic":"and","children":[{"predicate":{"name":"value","operator":"eq","values":[{"type":"string","value":"paid"}]}}]},"sorts":[{"name":"id","direction":"asc"}]}`), 512)
	if err != nil {
		t.Fatal(err)
	}
	first := complete.Request()
	firstFields, fieldsPresent := first.Fields.Value()
	firstSorts, sortsPresent := first.Sorts.Value()
	if !fieldsPresent || !sortsPresent || len(firstFields) != 1 || len(firstSorts) != 1 {
		t.Fatalf("first Request() = %#v", first)
	}
	firstFields[0] = "changed"
	firstSorts[0].Name = "changed"
	first.Filter.Children[0].Predicate.Values[0] = apiquery.StringValue("changed")
	again := complete.Request()
	againFields, _ := again.Fields.Value()
	againSorts, _ := again.Sorts.Value()
	if len(againFields) != 1 || len(againSorts) != 1 || againFields[0] != "id" || againSorts[0].Name != "id" ||
		again.Filter.Children[0].Predicate.Values[0].String() != "paid" {
		t.Fatalf("Request retained returned mutation: %#v", again)
	}
}

func TestValidationReportProjectsPathsTruncatesAndCopies(t *testing.T) {
	t.Parallel()

	limits := validation.DefaultLimits()
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "orders", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString}}})
	if err != nil {
		t.Fatal(err)
	}
	_, structured := apiquery.Compile(context.Background(), schema, apiquery.Request{Fields: apiquery.Present([]string{"unknown", "id", "id"})}, apiquery.CompileOptions{})
	if full := apiqueryvalidation.Report(structured, limits); len(full.Violations()) < 2 || full.Truncated() {
		t.Fatalf("structured report = %v, truncated %v", full.Violations(), full.Truncated())
	}
	truncatedLimits := limits
	truncatedLimits.MaxViolations = 1
	if truncated := apiqueryvalidation.Report(structured, truncatedLimits); len(truncated.Violations()) != 1 || !truncated.Truncated() {
		t.Fatalf("truncated report = %v, truncated %v", truncated.Violations(), truncated.Truncated())
	}
	nested := apiqueryvalidation.Report(nestedValidationError(t), limits).Violations()
	if len(nested) != 1 || nested[0].Path().String() != "filter.children[0]" || nested[0].Code() != "invalid_element" {
		t.Fatalf("nested report = %#v", nested)
	}

	for _, input := range []error{nil, errors.New("unsafe details"), structured} {
		report := apiqueryvalidation.Report(input, limits)
		snapshot := reportSnapshot(report)
		if returned := report.Violations(); len(returned) > 0 {
			returned[0].Parameters()["changed"] = "yes"
			returned[0] = validation.NewViolation(validation.RootPath(), "changed", validation.Error, nil, nil)
		}
		if !reflect.DeepEqual(reportSnapshot(report), snapshot) || reportParametersContain(report, "changed") {
			t.Fatalf("%v: report retained returned mutation: %v", input, reportSnapshot(report))
		}
	}
}

func observeJSONAPICallbacks(filterBehavior, pageBehavior string, marker error) (order []string, panicValue any, resultErr error) {
	defer func() { panicValue = recover() }()
	_, resultErr = apiqueryjsonapi.FromQuery(completeJSONAPIQuery(), apiqueryjsonapi.Config{
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

func jsonAPIConfig() apiqueryjsonapi.Config {
	return apiqueryjsonapi.Config{
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

func completePostgresPlan(t *testing.T) *apiquery.Plan {
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
