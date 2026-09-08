// Command api-query-v1-oracle emits a deterministic observation of the
// released API Query v1 adapter behavior. Run it only from an external module.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"time"

	apiquery "github.com/faustbrian/go-api-query"
	"github.com/faustbrian/go-api-query/apiqueryhttp"
	"github.com/faustbrian/go-api-query/apiqueryjsonapi"
	"github.com/faustbrian/go-api-query/apiquerypgx"
	"github.com/faustbrian/go-api-query/apiqueryrpc"
	"github.com/faustbrian/go-api-query/apiqueryvalidation"
	"github.com/faustbrian/go-api-query/cursor"
	jsonapi "github.com/faustbrian/go-jsonapi"
	validation "github.com/faustbrian/go-validation"
	"github.com/jackc/pgx/v5"
)

const (
	maxInputBytes = 64 << 10
	maxJSONDepth  = 64
	httpCaseCount = 18
	operatorCount = 16
)

type input struct {
	Schema            string      `json:"schema"`
	HTTP              []httpInput `json:"http"`
	PostgresOperators []string    `json:"postgres_operators"`
}

type httpInput struct {
	Name     string `json:"name"`
	Raw      string `json:"raw"`
	MaxBytes int    `json:"max_bytes"`
}

type record struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

type errorObservation struct {
	IsInvalid        bool   `json:"is_invalid"`
	IsUnsupported    bool   `json:"is_unsupported,omitempty"`
	ExactInvalid     bool   `json:"exact_invalid"`
	ExactUnsupported bool   `json:"exact_unsupported,omitempty"`
	Text             string `json:"text,omitempty"`
}

func main() {
	if len(os.Args) != 2 {
		panic("usage: api-query-v1-oracle INPUT")
	}
	inputBytes, err := readBoundedFile(os.Args[1], maxInputBytes)
	must(err)
	var spec input
	must(decodeInput(inputBytes, &spec))
	if spec.Schema != "api-query-v1-oracle-input-v1" {
		panic("unexpected input schema")
	}
	if len(spec.HTTP) != httpCaseCount || len(spec.PostgresOperators) != operatorCount {
		panic("unexpected input matrix")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	records := []record{{Name: "input-schema", Value: spec.Schema}}
	records = append(records, observeSentinels()...)
	records = append(records, observeCursorSentinels()...)
	records = append(records, observeHTTP(spec.HTTP)...)
	records = append(records, observeJSONAPI()...)
	records = append(records, observePostgres(ctx, spec.PostgresOperators)...)
	records = append(records, observeJSONRPC()...)
	records = append(records, observeValidation(ctx)...)
	records = append(records, observeStatelessResources(ctx))
	output, err := json.Marshal(struct {
		Schema  string   `json:"schema"`
		Records []record `json:"records"`
	}{Schema: "api-query-v1-oracle-output-v1", Records: records})
	must(err)
	_, err = os.Stdout.Write(append(output, '\n'))
	must(err)
}

func decodeInput(data []byte, target *input) error {
	if err := rejectDuplicateJSONMembers(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("input contains trailing JSON")
	}
	return nil
}

func rejectDuplicateJSONMembers(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("input contains trailing JSON")
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, depth int) error {
	if depth > maxJSONDepth {
		return errors.New("input exceeds maximum JSON depth")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, composite := token.(json.Delim)
	if !composite {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]struct{}{}
		for decoder.More() {
			member, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := member.(string)
			if !ok {
				return errors.New("object member is not a string")
			}
			if _, duplicate := seen[name]; duplicate {
				return fmt.Errorf("duplicate object member %q", name)
			}
			seen[name] = struct{}{}
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	_, err = decoder.Token()
	return err
}

func readBoundedFile(path string, maximum int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maximum+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maximum {
		return nil, errors.New("input exceeds maximum bytes")
	}
	return data, nil
}

func observeSentinels() []record {
	return []record{
		{Name: "sentinel-http", Value: sentinel(apiqueryhttp.ErrInvalid)},
		{Name: "sentinel-jsonapi-invalid", Value: sentinel(apiqueryjsonapi.ErrInvalid)},
		{Name: "sentinel-jsonapi-unsupported", Value: sentinel(apiqueryjsonapi.ErrUnsupported)},
		{Name: "sentinel-postgres", Value: sentinel(apiquerypgx.ErrInvalid)},
		{Name: "sentinel-jsonrpc", Value: sentinel(apiqueryrpc.ErrInvalid)},
		{Name: "named-types", Value: map[string]string{
			"jsonapi.Config":            reflect.TypeOf(apiqueryjsonapi.Config{}).PkgPath(),
			"jsonapi.FilterDecoder":     reflect.TypeOf(apiqueryjsonapi.FilterDecoder(nil)).PkgPath(),
			"jsonapi.PageDecoder":       reflect.TypeOf(apiqueryjsonapi.PageDecoder(nil)).PkgPath(),
			"postgres.Mapping":          reflect.TypeOf(apiquerypgx.Mapping{}).PkgPath(),
			"postgres.Compiler":         reflect.TypeOf(apiquerypgx.Compiler{}).PkgPath(),
			"postgres.QueryParts":       reflect.TypeOf(apiquerypgx.QueryParts{}).PkgPath(),
			"jsonrpc.Params":            reflect.TypeOf(apiqueryrpc.Params{}).PkgPath(),
			"jsonrpc.ContentDescriptor": reflect.TypeOf(apiqueryrpc.ContentDescriptor{}).PkgPath(),
		}},
	}
}

func sentinel(err error) map[string]any {
	return map[string]any{"non_nil": err != nil, "self_identity": errors.Is(err, err), "text": err.Error()}
}

func observeCursorSentinels() []record {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	keyring, err := cursor.NewKeyring(cursor.Key{ID: "key", Secret: bytes.Repeat([]byte{0x42}, 32)})
	must(err)
	newCodec := func(replay cursor.ReplayGuard) *cursor.Codec {
		codec, codecErr := cursor.NewCodec(cursor.Config{
			Version: "v1", Keys: keyring, MaxEncodedBytes: 2048, MaxPositions: 4,
			MaxStringBytes: 64, MaxTTL: time.Minute, Clock: func() time.Time { return now },
			ReplayGuard: replay, Random: bytes.NewReader(bytes.Repeat([]byte{0x24}, 64)),
		})
		must(codecErr)
		return codec
	}
	sorts := []apiquery.SortTerm{{Name: "id", Direction: apiquery.Ascending}}
	payload := cursor.Payload{SchemaRevision: "orders-v1", Direction: cursor.Forward, Sorts: sorts,
		Positions: []apiquery.Value{apiquery.StringValue("order-42")}, ExpiresAt: now.Add(time.Minute), Policy: "stable"}
	codec := newCodec(nil)
	token, err := codec.Encode(payload)
	must(err)
	_, invalidErr := codec.Decode("malformed", payload.SchemaRevision, sorts)
	_, versionErr := codec.Decode("v2.key.invalid", payload.SchemaRevision, sorts)
	_, schemaErr := codec.Decode(token, "orders-v2", sorts)
	_, sortErr := codec.Decode(token, payload.SchemaRevision, []apiquery.SortTerm{{Name: "created_at", Direction: apiquery.Descending}})
	replayCodec := newCodec(func([32]byte, time.Time) bool { return false })
	replayToken, err := replayCodec.Encode(payload)
	must(err)
	_, replayErr := replayCodec.Decode(replayToken, payload.SchemaRevision, sorts)
	expiredCodec := newCodec(nil)
	expiringPayload := payload
	expiringPayload.ExpiresAt = now.Add(time.Second)
	expiredToken, err := expiredCodec.Encode(expiringPayload)
	must(err)
	expiredCodec.SetClock(func() time.Time { return now.Add(2 * time.Second) })
	_, expiredErr := expiredCodec.Decode(expiredToken, payload.SchemaRevision, sorts)
	return []record{
		cursorSentinelRecord("cursor-sentinel-invalid", cursor.ErrInvalid, invalidErr),
		cursorSentinelRecord("cursor-sentinel-expired", cursor.ErrExpired, expiredErr),
		cursorSentinelRecord("cursor-sentinel-version", cursor.ErrVersion, versionErr),
		cursorSentinelRecord("cursor-sentinel-schema", cursor.ErrSchema, schemaErr),
		cursorSentinelRecord("cursor-sentinel-sort", cursor.ErrSort, sortErr),
		cursorSentinelRecord("cursor-sentinel-replay", cursor.ErrReplay, replayErr),
	}
}

func cursorSentinelRecord(name string, sentinelError, observed error) record {
	return record{Name: name, Value: map[string]any{
		"sentinel":   sentinel(sentinelError),
		"normal_use": map[string]any{"is": errors.Is(observed, sentinelError), "exact": observed == sentinelError, "text": observed.Error()},
	}}
}

func observeHTTP(inputs []httpInput) []record {
	result := make([]record, 0, len(inputs))
	for _, item := range inputs {
		raw := item.Raw
		if item.Name == "invalid-raw-utf8" {
			raw = string([]byte{'f', 'i', 'e', 'l', 'd', 's', '=', 0xff})
		}
		request, err := apiqueryhttp.Parse(raw, item.MaxBytes)
		result = append(result, record{Name: "http/" + item.Name, Value: map[string]any{
			"error": errorState(err, apiqueryhttp.ErrInvalid, nil), "request": requestState(request),
		}})
	}
	return result
}

func observeJSONAPI() []record {
	result := []record{}
	query := jsonapi.Query{
		Fields: map[string][]string{"orders": {"status"}}, IncludePresent: true,
		Include: []string{"customer.address"}, SortPresent: true,
		Sort:   []jsonapi.SortField{{Name: "created_at", Descending: true}, {Name: "id"}},
		Filter: jsonapi.ParameterFamily{"filter[status]": {"paid"}},
		Page:   jsonapi.ParameterFamily{"page[size]": {"20"}},
	}
	order := []string{}
	request, err := apiqueryjsonapi.FromQuery(query, apiqueryjsonapi.Config{
		Resource: "orders",
		DecodeFilter: func(family jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
			order = append(order, "filter")
			family["filter[status]"][0] = "callback-mutated"
			return &apiquery.FilterExpr{Predicate: &apiquery.Predicate{Name: "status", Operator: apiquery.OpEqual,
				Values: []apiquery.Value{apiquery.StringValue("paid")}}}, nil
		},
		DecodePage: func(family jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			order = append(order, "page")
			family["page[size]"][0] = "callback-mutated"
			return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 20}, nil
		},
	})
	query.Fields["orders"][0] = "source-mutated"
	query.Include[0] = "source-mutated"
	result = append(result, record{Name: "jsonapi/complete-copy-order", Value: map[string]any{
		"error": errorState(err, apiqueryjsonapi.ErrInvalid, apiqueryjsonapi.ErrUnsupported),
		"order": order, "request": requestState(request),
		"filter_source": query.Filter["filter[status]"][0], "page_source": query.Page["page[size]"][0],
	}})
	absentCalls := 0
	request, err = apiqueryjsonapi.FromQuery(jsonapi.Query{}, apiqueryjsonapi.Config{Resource: "orders",
		DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { absentCalls++; return nil, nil },
		DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			absentCalls++
			return apiquery.PageRequest{}, nil
		},
	})
	result = append(result, record{Name: "jsonapi/absent-families", Value: map[string]any{
		"calls": absentCalls, "error": errorState(err, apiqueryjsonapi.ErrInvalid, apiqueryjsonapi.ErrUnsupported),
		"request": requestState(request),
	}})
	cases := []struct {
		name   string
		query  jsonapi.Query
		config apiqueryjsonapi.Config
	}{
		{"missing-resource", jsonapi.Query{}, apiqueryjsonapi.Config{}},
		{"missing-filter-decoder", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders"}},
		{"missing-page-decoder", jsonapi.Query{Page: jsonapi.ParameterFamily{"page[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders"}},
		{"filter-nil", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, nil }}},
		{"filter-error", jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { return nil, errors.New("private") }}},
		{"page-error", jsonapi.Query{Page: jsonapi.ParameterFamily{"page[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders", DecodePage: func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
			return apiquery.PageRequest{}, errors.New("private")
		}}},
	}
	for _, item := range cases {
		_, caseErr := apiqueryjsonapi.FromQuery(item.query, item.config)
		result = append(result, record{Name: "jsonapi/" + item.name, Value: errorState(caseErr, apiqueryjsonapi.ErrInvalid, apiqueryjsonapi.ErrUnsupported)})
	}
	panicValue := ""
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicValue = fmt.Sprint(recovered)
			}
		}()
		_, _ = apiqueryjsonapi.FromQuery(jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}},
			apiqueryjsonapi.Config{Resource: "orders", DecodeFilter: func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) { panic("callback-panic") }})
	}()
	result = append(result, record{Name: "jsonapi/callback-panic", Value: panicValue})
	return result
}

func observePostgres(ctx context.Context, operators []string) []record {
	result := []record{}
	mapping := apiquerypgx.Mapping{Fields: map[string]string{"id": "records.id"}, Filters: map[string]string{"value": "records.value"}, Sorts: map[string]string{"id": "records.id"}}
	compiler, err := apiquerypgx.NewCompiler(mapping)
	must(err)
	mapping.Fields["id"] = "changed.bad"
	mapping.Filters["value"] = "changed.bad"
	mapping.Sorts["id"] = "changed.bad"
	for _, name := range operators {
		expr := operatorExpression(name)
		parts, compileErr := compiler.Compile(operatorPlan(ctx, expr))
		result = append(result, record{Name: "postgres/operator/" + name, Value: map[string]any{
			"error": errorState(compileErr, apiquerypgx.ErrInvalid, nil), "parts": partsState(parts),
		}})
	}
	invalidIdentifiers := []string{"", ".id", "a.b.c.d", "1table.id", "table.bad-name", "table.id;drop"}
	for _, identifier := range invalidIdentifiers {
		_, invalidErr := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Fields: map[string]string{"id": identifier}})
		result = append(result, record{Name: "postgres/invalid-identifier/" + identifier, Value: errorState(invalidErr, apiquerypgx.ErrInvalid, nil)})
	}
	_, emptyNameErr := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Fields: map[string]string{"": "records.id"}})
	result = append(result,
		record{Name: "postgres/empty-capability", Value: errorState(emptyNameErr, apiquerypgx.ErrInvalid, nil)},
		record{Name: "postgres/nil-compiler", Value: compileError((*apiquerypgx.Compiler)(nil), operatorPlan(ctx, operatorExpression("eq")))},
		record{Name: "postgres/nil-plan", Value: compileError(compiler, nil)},
	)
	result = append(result, observePostgresSnapshot(ctx))
	result = append(result, observePostgresFailures(ctx)...)
	result = append(result, observePostgresRuntime(ctx))
	return result
}

func observePostgresSnapshot(ctx context.Context) record {
	mapping := apiquerypgx.Mapping{
		Fields:      map[string]string{"id": "orders.id", "status": "orders.status"},
		Filters:     map[string]string{"status": "orders.status"},
		Sorts:       map[string]string{"id": "orders.id"},
		Constraints: map[string]string{"tenant_id": "orders.tenant_id"},
	}
	compiler, err := apiquerypgx.NewCompiler(mapping)
	must(err)
	mapping.Fields["id"] = "mutated.id"
	mapping.Filters["status"] = "mutated.status"
	mapping.Sorts["id"] = "mutated.id"
	mapping.Constraints["tenant_id"] = "mutated.tenant"
	parts, err := compiler.Compile(databasePlan(ctx, "paid"))
	must(err)
	return record{Name: "postgres/mapping-snapshot-and-constraint-precedence", Value: partsState(parts)}
}

func observePostgresFailures(ctx context.Context) []record {
	plan := operatorPlan(ctx, operatorExpression("eq"))
	missingField, _ := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Filters: map[string]string{"value": "records.value"}})
	missingFilter, _ := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Fields: map[string]string{"id": "records.id"}})
	sorted := sortedPlan(ctx)
	missingSort, _ := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Fields: map[string]string{"id": "records.id"}})
	constraintPlan := databasePlan(ctx, "paid")
	missingConstraint, _ := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Fields: map[string]string{"id": "orders.id", "status": "orders.status"}, Filters: map[string]string{"status": "orders.status"}, Sorts: map[string]string{"id": "orders.id"}})
	return []record{
		{Name: "postgres/missing-field", Value: compileError(missingField, plan)},
		{Name: "postgres/missing-filter", Value: compileError(missingFilter, plan)},
		{Name: "postgres/missing-sort", Value: compileError(missingSort, sorted)},
		{Name: "postgres/missing-constraint", Value: compileError(missingConstraint, constraintPlan)},
	}
}

func observePostgresRuntime(ctx context.Context) record {
	dsn := os.Getenv("APIQUERY_TEST_DATABASE_URL")
	if dsn == "" {
		panic("APIQUERY_TEST_DATABASE_URL is required")
	}
	connection, err := pgx.Connect(ctx, dsn)
	must(err)
	defer connection.Close(ctx)
	_, err = connection.Exec(ctx, `CREATE TEMP TABLE orders (id text PRIMARY KEY, tenant_id text NOT NULL, status text NOT NULL, created_at timestamptz NOT NULL)`)
	must(err)
	_, err = connection.Exec(ctx, `INSERT INTO orders (id, tenant_id, status, created_at) VALUES ('a','tenant-42','paid','2026-01-02T00:00:00Z'),('b','tenant-42','paid','2026-01-02T00:00:00Z'),('foreign','tenant-other','paid','2026-01-03T00:00:00Z')`)
	must(err)
	compiler, err := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Fields: map[string]string{"id": "orders.id", "status": "orders.status"}, Filters: map[string]string{"status": "orders.status"}, Sorts: map[string]string{"id": "orders.id"}, Constraints: map[string]string{"tenant_id": "orders.tenant_id"}})
	must(err)
	parts, err := compiler.Compile(databasePlan(ctx, "paid' OR true --"))
	must(err)
	rows, err := connection.Query(ctx, "SELECT "+parts.Projection+" FROM orders WHERE "+parts.Where+" ORDER BY "+parts.OrderBy, valueArguments(parts.Arguments)...)
	must(err)
	injectionIDs := collectIDs(rows)
	parts, err = compiler.Compile(databasePlan(ctx, "paid"))
	must(err)
	rows, err = connection.Query(ctx, "SELECT "+parts.Projection+" FROM orders WHERE "+parts.Where+" ORDER BY "+parts.OrderBy, valueArguments(parts.Arguments)...)
	must(err)
	paidIDs := collectIDs(rows)
	var count int
	must(connection.QueryRow(ctx, "SELECT count(*) FROM orders").Scan(&count))
	return record{Name: "postgres/real-database", Value: map[string]any{"injection_ids": injectionIDs, "paid_ids": paidIDs, "row_count": count}}
}

func observeJSONRPC() []record {
	result := []record{}
	data := []byte(`{"schema_revision":"v1","fields":[],"includes":["customer"],"filter":{"predicate":{"name":"status","operator":"eq","values":[{"type":"string","value":"paid"}]}},"sorts":[{"name":"id","direction":"asc"}],"page":{"mode":"cursor","size":10,"after":"opaque"}}`)
	params, err := apiqueryrpc.Parse(data, 2048)
	request := params.Request()
	if params.Fields != nil {
		*params.Fields = append((*params.Fields)[:0], "mutated")
	}
	if params.Includes != nil {
		(*params.Includes)[0] = "mutated"
	}
	if params.Filter != nil {
		params.Filter.Predicate.Values[0] = apiquery.StringValue("mutated")
	}
	if params.Sorts != nil {
		(*params.Sorts)[0].Name = "mutated"
	}
	result = append(result, record{Name: "jsonrpc/complete-copy", Value: map[string]any{"error": errorState(err, apiqueryrpc.ErrInvalid, nil), "request": requestState(request)}})
	absent, err := apiqueryrpc.Parse([]byte(`{}`), 32)
	empty, emptyErr := apiqueryrpc.Parse([]byte(`{"fields":[],"includes":[],"sorts":[]}`), 128)
	exactData := []byte(`{"fields":[]}`)
	exact, exactErr := apiqueryrpc.Parse(exactData, len(exactData))
	result = append(result,
		record{Name: "jsonrpc/absent", Value: map[string]any{"error": errorState(err, apiqueryrpc.ErrInvalid, nil), "request": requestState(absent.Request())}},
		record{Name: "jsonrpc/explicit-empty", Value: map[string]any{"error": errorState(emptyErr, apiqueryrpc.ErrInvalid, nil), "request": requestState(empty.Request())}},
		record{Name: "jsonrpc/exact-byte-limit", Value: map[string]any{"error": errorState(exactErr, apiqueryrpc.ErrInvalid, nil), "request": requestState(exact.Request())}},
	)
	invalids := map[string]struct {
		data []byte
		max  int
	}{
		"nil": {nil, 100}, "array": {[]byte(`[]`), 100},
		"unknown":          {[]byte(`{"unknown":1}`), 100},
		"duplicate":        {[]byte(`{"fields":[],"fields":[]}`), 100},
		"nested-duplicate": {[]byte(`{"filter":{"predicate":{"name":"first","name":"second"}}}`), 100},
		"excess-depth":     {deepJSON(65), 512},
		"trailing":         {[]byte(`{} {}`), 100}, "invalid-utf8": {[]byte{0xff, 0x00}, 100},
		"disabled-bound": {[]byte(`{}`), 0}, "excess-bound": {[]byte(`{}`), 1},
	}
	for name, invalid := range invalids {
		_, invalidErr := apiqueryrpc.Parse(invalid.data, invalid.max)
		result = append(result, record{Name: "jsonrpc/invalid/" + name, Value: errorState(invalidErr, apiqueryrpc.ErrInvalid, nil)})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	first := apiqueryrpc.OpenRPCContentDescriptor()
	second := apiqueryrpc.OpenRPCContentDescriptor()
	first.Schema["type"] = "mutated"
	firstProperties := first.Schema["properties"].(map[string]any)
	firstSchemaRevision := firstProperties["schema_revision"].(map[string]any)
	firstSchemaRevision["type"] = "mutated"
	secondProperties := second.Schema["properties"].(map[string]any)
	secondSchemaRevision := secondProperties["schema_revision"].(map[string]any)
	result = append(result, record{Name: "jsonrpc/descriptor-copy", Value: map[string]any{
		"first_type": first.Schema["type"], "first_nested_type": firstSchemaRevision["type"],
		"second_type": second.Schema["type"], "second_nested_type": secondSchemaRevision["type"],
	}})
	return result
}

func deepJSON(depth int) []byte {
	return []byte(`{"filter":` + strings.Repeat(`[`, depth) + `null` + strings.Repeat(`]`, depth) + `}`)
}

func observeValidation(ctx context.Context) []record {
	limits := validation.DefaultLimits()
	result := []record{{Name: "validation/nil", Value: reportState(apiqueryvalidation.Report(nil, limits))}}
	result = append(result, record{Name: "validation/generic", Value: reportState(apiqueryvalidation.Report(errors.New("unsafe details"), limits))})
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "orders", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString}}})
	must(err)
	_, compileErr := apiquery.Compile(ctx, schema, apiquery.Request{Fields: apiquery.Present([]string{"unknown", "id", "id"})}, apiquery.CompileOptions{})
	report := apiqueryvalidation.Report(compileErr, limits)
	violations := report.Violations()
	if len(violations) > 0 {
		copyParameters := violations[0].Parameters()
		copyParameters["mutated"] = "yes"
		violations[0] = validation.NewViolation(validation.RootPath(), "mutated", validation.Error, nil, nil)
	}
	result = append(result, record{Name: "validation/structured-copy", Value: reportState(report)})
	nestedSchema, err := apiquery.NewSchema(apiquery.SchemaConfig{
		Resource: "orders", Revision: "v1",
		Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString}},
		Filters: []apiquery.FilterDefinition{{Name: "status", Type: apiquery.TypeString,
			Operators: []apiquery.Operator{apiquery.OpEqual}}},
		AllowedLogic: []apiquery.Logic{apiquery.LogicAnd},
	})
	must(err)
	_, nestedErr := apiquery.Compile(ctx, nestedSchema, apiquery.Request{Filter: &apiquery.FilterExpr{
		Logic: apiquery.LogicAnd,
		Children: []apiquery.FilterExpr{{Predicate: &apiquery.Predicate{
			Name: "status", Operator: apiquery.OpEqual,
			Values: []apiquery.Value{apiquery.IntValue(1)},
		}}},
	}}, apiquery.CompileOptions{})
	result = append(result, record{Name: "validation/nested-path", Value: reportState(apiqueryvalidation.Report(nestedErr, limits))})
	limits.MaxViolations = 1
	result = append(result, record{Name: "validation/truncated", Value: reportState(apiqueryvalidation.Report(compileErr, limits))})
	return result
}

func observeStatelessResources(ctx context.Context) record {
	beforeGoroutines := runtime.NumGoroutine()
	beforeDescriptors := descriptorCount(ctx)
	httpRequest, err := apiqueryhttp.Parse("fields=id", 64)
	must(err)
	rpcParams, err := apiqueryrpc.Parse([]byte(`{"fields":["id"]}`), 64)
	must(err)
	jsonapiRequest, err := apiqueryjsonapi.FromQuery(jsonapi.Query{}, apiqueryjsonapi.Config{Resource: "orders"})
	must(err)
	compiler, err := apiquerypgx.NewCompiler(apiquerypgx.Mapping{Fields: map[string]string{"id": "records.id"}})
	must(err)
	parts, err := compiler.Compile(fieldOnlyPlan(ctx))
	must(err)
	report := apiqueryvalidation.Report(nil, validation.DefaultLimits())
	returnedOwners := countReturnedResourceOwners([]any{
		httpRequest, rpcParams, jsonapiRequest, compiler, parts, report,
	})
	if returnedOwners != 0 {
		panic("adapter returned an unexpected resource owner")
	}
	runtime.GC()
	if runtime.NumGoroutine() != beforeGoroutines {
		panic("adapter execution changed goroutine ownership")
	}
	if descriptorCount(ctx) != beforeDescriptors {
		panic("adapter execution changed file or connection descriptor ownership")
	}
	return record{Name: "resources/observed-stateless", Value: map[string]any{
		"goroutine_delta_zero":                     true,
		"file_or_connection_descriptor_delta_zero": true,
		"returned_resource_owners":                 returnedOwners,
	}}
}

func countReturnedResourceOwners(values []any) int {
	count := 0
	for _, value := range values {
		if _, ok := value.(interface{ Close() error }); ok {
			count++
		}
		if _, ok := value.(pgx.Rows); ok {
			count++
		}
		if _, ok := value.(pgx.Tx); ok {
			count++
		}
	}
	return count
}

func descriptorCount(ctx context.Context) int {
	output, err := exec.CommandContext(ctx, "/usr/sbin/lsof", "-p", fmt.Sprint(os.Getpid()), "-Fn").Output()
	must(err)
	return strings.Count(string(output), "\nn")
}

func fieldOnlyPlan(ctx context.Context) *apiquery.Plan {
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "records", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString, Required: true}}})
	must(err)
	plan, err := apiquery.Compile(ctx, schema, apiquery.Request{}, apiquery.CompileOptions{})
	must(err)
	return plan
}

func requestState(request apiquery.Request) map[string]any {
	fields, fieldsPresent := request.Fields.Value()
	includes, includesPresent := request.Includes.Value()
	sorts, sortsPresent := request.Sorts.Value()
	revision, revisionPresent := request.SchemaRevision.Value()
	return map[string]any{"schema_revision": revision, "schema_revision_present": revisionPresent,
		"fields": fields, "fields_present": fieldsPresent, "includes": includes,
		"includes_present": includesPresent, "filter": request.Filter, "sorts": sorts,
		"sorts_present": sortsPresent, "page": request.Page}
}

func reportState(report validation.Report) map[string]any {
	items := []map[string]any{}
	for _, violation := range report.Violations() {
		items = append(items, map[string]any{"path": violation.Path().String(), "code": violation.Code(), "severity": violation.Severity(), "parameters": violation.Parameters(), "cause": fmt.Sprint(violation.Cause())})
	}
	return map[string]any{"empty": report.Empty(), "len": report.Len(), "truncated": report.Truncated(), "has_errors": report.HasErrors(), "context_error": fmt.Sprint(report.ContextError()), "violations": items}
}

func errorState(err, invalid, unsupported error) errorObservation {
	if err == nil {
		return errorObservation{}
	}
	return errorObservation{
		IsInvalid: errors.Is(err, invalid), IsUnsupported: unsupported != nil && errors.Is(err, unsupported),
		ExactInvalid: err == invalid, ExactUnsupported: unsupported != nil && err == unsupported,
		Text: err.Error(),
	}
}

func compileError(compiler *apiquerypgx.Compiler, plan *apiquery.Plan) errorObservation {
	_, err := compiler.Compile(plan)
	return errorState(err, apiquerypgx.ErrInvalid, nil)
}

func partsState(parts apiquerypgx.QueryParts) map[string]any {
	args := make([]map[string]string, len(parts.Arguments))
	for index, value := range parts.Arguments {
		args[index] = map[string]string{"type": string(value.Type()), "value": value.String()}
	}
	return map[string]any{"projection": parts.Projection, "where": parts.Where, "order_by": parts.OrderBy, "arguments": args}
}

func operatorExpression(name string) *apiquery.FilterExpr {
	leaf := func(operator apiquery.Operator, values ...string) *apiquery.FilterExpr {
		typed := make([]apiquery.Value, len(values))
		for index, value := range values {
			typed[index] = apiquery.StringValue(value)
		}
		return &apiquery.FilterExpr{Predicate: &apiquery.Predicate{Name: "value", Operator: operator, Values: typed}}
	}
	group := func(logic apiquery.Logic, children ...*apiquery.FilterExpr) *apiquery.FilterExpr {
		values := make([]apiquery.FilterExpr, len(children))
		for i, child := range children {
			values[i] = *child
		}
		return &apiquery.FilterExpr{Logic: logic, Children: values}
	}
	switch name {
	case "eq":
		return leaf(apiquery.OpEqual, "a")
	case "neq":
		return leaf(apiquery.OpNotEqual, "a")
	case "lt":
		return leaf(apiquery.OpLess, "a")
	case "lte":
		return leaf(apiquery.OpLessOrEqual, "a")
	case "gt":
		return leaf(apiquery.OpGreater, "a")
	case "gte":
		return leaf(apiquery.OpGreaterOrEqual, "a")
	case "in":
		return leaf(apiquery.OpIn, "a", "b")
	case "not_in":
		return leaf(apiquery.OpNotIn, "a", "b")
	case "between":
		return leaf(apiquery.OpBetween, "a", "z")
	case "is_null":
		return leaf(apiquery.OpIsNull)
	case "contains":
		return leaf(apiquery.OpContains, `a%b_c\d`)
	case "starts_with":
		return leaf(apiquery.OpStartsWith, "a")
	case "ends_with":
		return leaf(apiquery.OpEndsWith, "z")
	case "and":
		return group(apiquery.LogicAnd, leaf(apiquery.OpGreater, "a"), leaf(apiquery.OpLess, "z"))
	case "or":
		return group(apiquery.LogicOr, leaf(apiquery.OpEqual, "a"), leaf(apiquery.OpNotEqual, "z"))
	case "not":
		return group(apiquery.LogicNot, leaf(apiquery.OpEqual, "a"))
	default:
		panic("unknown operator " + name)
	}
}

func operatorPlan(ctx context.Context, expression *apiquery.FilterExpr) *apiquery.Plan {
	operators := []apiquery.Operator{apiquery.OpEqual, apiquery.OpNotEqual, apiquery.OpLess, apiquery.OpLessOrEqual, apiquery.OpGreater, apiquery.OpGreaterOrEqual, apiquery.OpIn, apiquery.OpNotIn, apiquery.OpBetween, apiquery.OpContains, apiquery.OpStartsWith, apiquery.OpEndsWith}
	operators = append(operators, apiquery.OpIsNull)
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "records", Revision: "v1",
		Fields:       []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString, Required: true}},
		Filters:      []apiquery.FilterDefinition{{Name: "value", Type: apiquery.TypeString, Nullable: true, AllowEmpty: true, Operators: operators}},
		AllowedLogic: []apiquery.Logic{apiquery.LogicAnd, apiquery.LogicOr, apiquery.LogicNot}})
	must(err)
	plan, err := apiquery.Compile(ctx, schema, apiquery.Request{Filter: expression}, apiquery.CompileOptions{})
	must(err)
	return plan
}

func sortedPlan(ctx context.Context) *apiquery.Plan {
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "records", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString, Required: true}}, Sorts: []apiquery.SortDefinition{{Name: "id", Type: apiquery.TypeString, Nulls: apiquery.NullsLast}}})
	must(err)
	plan, err := apiquery.Compile(ctx, schema, apiquery.Request{Sorts: apiquery.Present([]apiquery.SortTerm{{Name: "id", Direction: apiquery.Descending}})}, apiquery.CompileOptions{})
	must(err)
	return plan
}

func databasePlan(ctx context.Context, status string) *apiquery.Plan {
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "orders", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString, Required: true}, {Name: "status", Type: apiquery.TypeString, Default: true}}, Filters: []apiquery.FilterDefinition{{Name: "status", Type: apiquery.TypeString, Operators: []apiquery.Operator{apiquery.OpEqual}, AllowEmpty: true}}, Sorts: []apiquery.SortDefinition{{Name: "id", Type: apiquery.TypeString, TieBreaker: true}}, DefaultSort: []apiquery.SortTerm{{Name: "id", Direction: apiquery.Ascending}}})
	must(err)
	plan, err := apiquery.Compile(ctx, schema, apiquery.Request{Filter: &apiquery.FilterExpr{Predicate: &apiquery.Predicate{Name: "status", Operator: apiquery.OpEqual, Values: []apiquery.Value{apiquery.StringValue(status)}}}}, apiquery.CompileOptions{MandatoryConstraints: []apiquery.Constraint{{Name: "tenant_id", Value: apiquery.StringValue("tenant-42"), Protected: true}}})
	must(err)
	return plan
}

func valueArguments(values []apiquery.Value) []any {
	result := make([]any, len(values))
	for i := range values {
		result[i] = values[i].String()
	}
	return result
}

func collectIDs(rows pgx.Rows) []string {
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		values, err := rows.Values()
		must(err)
		result = append(result, values[0].(string))
	}
	must(rows.Err())
	return result
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
