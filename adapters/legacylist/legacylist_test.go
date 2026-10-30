package apiquerylegacylist_test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	legacylist "github.com/faustbrian/go-api-query/v5/adapters/legacylist"
)

// locationProfile mirrors Mono Location's app.list_locations declaration, the
// widest of the three legacy consumers (Location, Track and Postal).
func locationProfile() legacylist.Profile {
	return legacylist.Profile{
		Filters: []legacylist.Resource{
			{Name: "self", Attributes: []string{"id", "type", "name", "postal_code"}},
			{Name: "vendor", Attributes: []string{"id", "slug"}},
		},
		Sorts:    []legacylist.Resource{{Name: "self", Attributes: []string{"type", "name", "code"}}},
		MaxBytes: 4096,
	}
}

func listError(t *testing.T, err error) *legacylist.Error {
	t.Helper()
	var target *legacylist.Error
	if !errors.As(err, &target) || !errors.Is(err, legacylist.ErrInvalid) {
		t.Fatalf("error = %v, want *legacylist.Error wrapping ErrInvalid", err)
	}
	return target
}

func TestParseFiltersReadsDeclaredResourcesInProfileOrder(t *testing.T) {
	t.Parallel()

	filters, err := locationProfile().ParseFilters(json.RawMessage(`{
		"vendor": [{"attribute": "slug", "operator": "like", "value": "post%"}],
		"self": [
			{"attribute": "type", "operator": "in", "value": ["pickup", "locker"], "boolean": "and"},
			{"attribute": "name", "operator": "not_like", "value": "x", "boolean": "or"},
			{"attribute": "postal_code", "operator": "is_not_null"},
			{"attribute": "id", "operator": "not_between", "value": [1, 2.50], "boolean": null, "extra": true}
		]
	}`))
	if err != nil {
		t.Fatalf("ParseFilters() error = %v", err)
	}
	want := []legacylist.Filter{
		{Resource: "self", Attribute: "type", Operator: legacylist.In, Value: json.RawMessage(`["pickup", "locker"]`), Boolean: legacylist.And},
		{Resource: "self", Attribute: "name", Operator: legacylist.NotLike, Value: json.RawMessage(`"x"`), Boolean: legacylist.Or},
		{Resource: "self", Attribute: "postal_code", Operator: legacylist.IsNotNull, Boolean: legacylist.And},
		{Resource: "self", Attribute: "id", Operator: legacylist.NotBetween, Value: json.RawMessage(`[1, 2.50]`), Boolean: legacylist.And},
		{Resource: "vendor", Attribute: "slug", Operator: legacylist.Like, Value: json.RawMessage(`"post%"`), Boolean: legacylist.And},
	}
	if !reflect.DeepEqual(filters, want) {
		t.Fatalf("ParseFilters() = %#v\nwant %#v", filters, want)
	}
}

func TestParseFiltersTreatsAbsentAndNullAsNoFilters(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", " \n", "null", `{}`, `{"self": null}`, `{"self": []}`} {
		filters, err := locationProfile().ParseFilters(json.RawMessage(raw))
		if err != nil || len(filters) != 0 {
			t.Fatalf("ParseFilters(%q) = %v, %v; want no filters", raw, filters, err)
		}
	}
}

func TestParseFiltersAcceptsEveryLongOperatorName(t *testing.T) {
	t.Parallel()

	names := []string{
		"equals", "not_equals", "greater_than", "greater_than_or_equal_to",
		"less_than", "less_than_or_equal_to", "like", "not_like", "in", "not_in",
		"between", "not_between", "is_null", "is_not_null",
	}
	operators := legacylist.Operators()
	if len(operators) != len(names) {
		t.Fatalf("Operators() = %v", operators)
	}
	for index, name := range names {
		operator, ok := legacylist.ParseOperator(name)
		if !ok || operator != operators[index] || string(operator) != name {
			t.Fatalf("ParseOperator(%q) = %q, %v", name, operator, ok)
		}
	}
	for _, name := range []string{"", "eq", "=", "EQUALS", "Equals", "neq", "ilike", "not_null", "contains"} {
		if operator, ok := legacylist.ParseOperator(name); ok {
			t.Fatalf("ParseOperator(%q) accepted %q", name, operator)
		}
	}
}

func TestParseFiltersClassifiesEachRejection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		kind    legacylist.ErrorKind
		pointer string
		unknown string
		allowed []string
	}{
		{name: "not an object", raw: `[]`, kind: legacylist.KindShape, pointer: "/params/filters"},
		{name: "string", raw: `"self"`, kind: legacylist.KindShape, pointer: "/params/filters"},
		{name: "resource not an array", raw: `{"self": {"attribute": "id"}}`, kind: legacylist.KindShape, pointer: "/params/filters/self"},
		{name: "definition not an object", raw: `{"vendor": ["id"]}`, kind: legacylist.KindShape, pointer: "/params/filters/vendor"},
		{name: "operator not a string", raw: `{"self": [{"attribute": "id", "operator": 1}]}`, kind: legacylist.KindShape, pointer: "/params/filters/self"},
		{name: "undeclared attribute", raw: `{"self": [{"attribute": "slug", "operator": "equals"}]}`, kind: legacylist.KindAttribute, pointer: "/params/filters", unknown: "slug", allowed: []string{"id", "type", "name", "postal_code"}},
		{name: "missing attribute", raw: `{"vendor": [{"operator": "equals"}]}`, kind: legacylist.KindAttribute, pointer: "/params/filters", unknown: "", allowed: []string{"id", "slug"}},
		{name: "short operator", raw: `{"self": [{"attribute": "id", "operator": "eq"}]}`, kind: legacylist.KindOperator, pointer: "/params/filters", unknown: "eq"},
		{name: "missing operator", raw: `{"self": [{"attribute": "id"}]}`, kind: legacylist.KindOperator, pointer: "/params/filters"},
		{name: "unknown boolean", raw: `{"self": [{"attribute": "id", "operator": "equals", "boolean": "xor"}]}`, kind: legacylist.KindBoolean, pointer: "/params/filters", unknown: "xor"},
		{name: "uppercase boolean", raw: `{"self": [{"attribute": "id", "operator": "equals", "boolean": "AND"}]}`, kind: legacylist.KindBoolean, pointer: "/params/filters", unknown: "AND"},
		{name: "undeclared resource", raw: `{"self": [], "zone": [], "account": []}`, kind: legacylist.KindResource, pointer: "/params/filters", unknown: "account", allowed: []string{"self", "vendor"}},
		{name: "oversized", raw: `{"self": [` + strings.Repeat(`{"attribute":"id","operator":"equals"},`, 200) + `{}]}`, kind: legacylist.KindShape, pointer: "/params/filters"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filters, err := locationProfile().ParseFilters(json.RawMessage(tt.raw))
			got := listError(t, err)
			if filters != nil || got.Kind != tt.kind || got.Pointer != tt.pointer || got.Unknown != tt.unknown ||
				!reflect.DeepEqual(got.Allowed, tt.allowed) {
				t.Fatalf("ParseFilters() = %v, %#v", filters, got)
			}
		})
	}
}

// Location checks declared resources before reporting an undeclared one, so an
// attribute error in a declared resource wins for filters.
func TestParseFiltersReportsDeclaredResourceErrorsBeforeUndeclaredResources(t *testing.T) {
	t.Parallel()

	_, err := locationProfile().ParseFilters(json.RawMessage(`{"zone": [], "self": [{"attribute": "slug", "operator": "equals"}]}`))
	if got := listError(t, err); got.Kind != legacylist.KindAttribute {
		t.Fatalf("error kind = %v, want attribute", got.Kind)
	}
}

func TestParseFiltersCanIgnoreUndeclaredResources(t *testing.T) {
	t.Parallel()

	profile := legacylist.Profile{
		Filters:                   []legacylist.Resource{{Name: "self", Attributes: []string{"status"}}},
		IgnoreUndeclaredResources: true,
		MaxBytes:                  1024,
	}
	filters, err := profile.ParseFilters(json.RawMessage(`{"other": "not even an array", "self": [{"attribute": "status", "operator": "equals", "value": "x"}]}`))
	if err != nil || len(filters) != 1 || filters[0].Attribute != "status" {
		t.Fatalf("ParseFilters() = %v, %v", filters, err)
	}
}

func TestParseFiltersDoesNotRetainInput(t *testing.T) {
	t.Parallel()

	raw := json.RawMessage(`{"self": [{"attribute": "name", "operator": "equals", "value": "abc"}]}`)
	filters, err := locationProfile().ParseFilters(raw)
	if err != nil {
		t.Fatal(err)
	}
	for index := range raw {
		raw[index] = ' '
	}
	if string(filters[0].Value) != `"abc"` {
		t.Fatalf("filter value aliases caller input: %s", filters[0].Value)
	}
}

func TestFilterValidateValueEnforcesOperatorArity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		operator legacylist.Operator
		value    string
		valid    bool
	}{
		{legacylist.In, `["a"]`, true},
		{legacylist.In, `[]`, true},
		{legacylist.NotIn, `"a"`, false},
		{legacylist.In, `null`, false},
		{legacylist.In, ``, false},
		{legacylist.Between, `[1, 2]`, true},
		{legacylist.NotBetween, `[1]`, false},
		{legacylist.Between, `[1, 2, 3]`, false},
		{legacylist.Between, `{"from": 1}`, false},
		{legacylist.Equals, ``, true},
		{legacylist.Equals, `{"any": "json"}`, true},
		{legacylist.Like, `[1`, false},
		{legacylist.IsNull, ``, true},
	}
	for _, tt := range tests {
		filter := legacylist.Filter{Resource: "self", Attribute: "id", Operator: tt.operator, Value: json.RawMessage(tt.value), Boolean: legacylist.And}
		err := filter.ValidateValue()
		if (err == nil) != tt.valid {
			t.Fatalf("%s %q ValidateValue() = %v, want valid %v", tt.operator, tt.value, err, tt.valid)
		}
		if err != nil {
			if got := listError(t, err); got.Kind != legacylist.KindValue || got.Pointer != "/params/filters" {
				t.Fatalf("ValidateValue() = %#v", got)
			}
		}
	}
}

func TestParseSortsReadsDirectionsAndRejectsEachViolation(t *testing.T) {
	t.Parallel()

	sorts, err := locationProfile().ParseSorts(json.RawMessage(`{"self": [{"attribute": "name", "direction": "desc"}, {"attribute": "type", "direction": "asc"}]}`))
	want := []legacylist.Sort{
		{Resource: "self", Attribute: "name", Direction: legacylist.Descending},
		{Resource: "self", Attribute: "type", Direction: legacylist.Ascending},
	}
	if err != nil || !reflect.DeepEqual(sorts, want) {
		t.Fatalf("ParseSorts() = %#v, %v", sorts, err)
	}
	for _, raw := range []string{"", "null", `{}`, `{"self": null}`} {
		if sorts, err := locationProfile().ParseSorts(json.RawMessage(raw)); err != nil || len(sorts) != 0 {
			t.Fatalf("ParseSorts(%q) = %v, %v", raw, sorts, err)
		}
	}

	tests := []struct {
		name    string
		raw     string
		kind    legacylist.ErrorKind
		pointer string
		unknown string
	}{
		{name: "not an object", raw: `["name"]`, kind: legacylist.KindShape, pointer: "/params/sorts"},
		{name: "resource not an array", raw: `{"self": "name"}`, kind: legacylist.KindShape, pointer: "/params/sorts/self"},
		{name: "undeclared attribute", raw: `{"self": [{"attribute": "id", "direction": "asc"}]}`, kind: legacylist.KindAttribute, pointer: "/params/sorts", unknown: "id"},
		{name: "missing direction", raw: `{"self": [{"attribute": "name"}]}`, kind: legacylist.KindDirection, pointer: "/params/sorts", unknown: ""},
		{name: "uppercase direction", raw: `{"self": [{"attribute": "name", "direction": "DESC"}]}`, kind: legacylist.KindDirection, pointer: "/params/sorts", unknown: "DESC"},
		{name: "undeclared resource first", raw: `{"vendor": [], "self": [{"attribute": "id", "direction": "up"}]}`, kind: legacylist.KindResource, pointer: "/params/sorts", unknown: "vendor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sorts, err := locationProfile().ParseSorts(json.RawMessage(tt.raw))
			got := listError(t, err)
			if sorts != nil || got.Kind != tt.kind || got.Pointer != tt.pointer || got.Unknown != tt.unknown {
				t.Fatalf("ParseSorts() = %v, %#v", sorts, got)
			}
		})
	}
}

func TestProfileRejectsInvalidDeclarations(t *testing.T) {
	t.Parallel()

	for name, profile := range map[string]legacylist.Profile{
		"zero bound":         {Filters: []legacylist.Resource{{Name: "self", Attributes: []string{"id"}}}},
		"empty resource":     {Filters: []legacylist.Resource{{Name: "", Attributes: []string{"id"}}}, MaxBytes: 1},
		"duplicate resource": {Sorts: []legacylist.Resource{{Name: "self"}, {Name: "self"}}, MaxBytes: 1},
		"empty attribute":    {Filters: []legacylist.Resource{{Name: "self", Attributes: []string{""}}}, MaxBytes: 1},
		"duplicate attr":     {Sorts: []legacylist.Resource{{Name: "self", Attributes: []string{"a", "a"}}}, MaxBytes: 1},
	} {
		if _, err := profile.ParseFilters(nil); !errors.Is(err, legacylist.ErrInvalidProfile) {
			t.Fatalf("%s: ParseFilters() error = %v", name, err)
		}
		if _, err := profile.ParseSorts(nil); !errors.Is(err, legacylist.ErrInvalidProfile) {
			t.Fatalf("%s: ParseSorts() error = %v", name, err)
		}
	}
}

func TestErrorTextDoesNotEchoClientInput(t *testing.T) {
	t.Parallel()

	_, err := locationProfile().ParseFilters(json.RawMessage(`{"self": [{"attribute": "private-name", "operator": "equals"}]}`))
	if err == nil || strings.Contains(err.Error(), "private-name") {
		t.Fatalf("error = %v", err)
	}
}

func encodedDocument(t *testing.T, document string) string {
	t.Helper()
	return base64.RawURLEncoding.EncodeToString([]byte(document))
}

func TestEncodeCursorWritesTheLaravelCursorDocument(t *testing.T) {
	t.Parallel()

	encoded, err := legacylist.EncodeCursor(legacylist.Cursor{
		Parameters: []legacylist.CursorParameter{
			{Name: "name", Value: json.RawMessage(`"Helsinki <1>"`)},
			{Name: "locations.id", Value: json.RawMessage(` "01HX" `)},
		},
		PointsToNextItems: true,
	})
	want := "{\"name\":\"Helsinki \\u003c1\\u003e\",\"locations.id\":\"01HX\",\"_pointsToNextItems\":true}"
	if err != nil || encoded != base64.RawURLEncoding.EncodeToString([]byte(want)) {
		decoded, _ := base64.RawURLEncoding.DecodeString(encoded)
		t.Fatalf("EncodeCursor() = %s, %v", decoded, err)
	}
	backward, err := legacylist.EncodeCursor(legacylist.Cursor{Parameters: []legacylist.CursorParameter{{Name: "id", Value: json.RawMessage(`7`)}}})
	if err != nil || backward != encodedDocument(t, `{"id":7,"_pointsToNextItems":false}`) {
		t.Fatalf("EncodeCursor(backward) = %q, %v", backward, err)
	}
}

func TestEncodeCursorRejectsUnrepresentableCursors(t *testing.T) {
	t.Parallel()

	for name, cursor := range map[string]legacylist.Cursor{
		"no parameters":  {},
		"empty name":     {Parameters: []legacylist.CursorParameter{{Name: "", Value: json.RawMessage(`1`)}}},
		"reserved name":  {Parameters: []legacylist.CursorParameter{{Name: "_pointsToNextItems", Value: json.RawMessage(`true`)}}},
		"duplicate name": {Parameters: []legacylist.CursorParameter{{Name: "id", Value: json.RawMessage(`1`)}, {Name: "id", Value: json.RawMessage(`2`)}}},
		"missing value":  {Parameters: []legacylist.CursorParameter{{Name: "id"}}},
		"invalid value":  {Parameters: []legacylist.CursorParameter{{Name: "id", Value: json.RawMessage(`{`)}}},
	} {
		if encoded, err := legacylist.EncodeCursor(cursor); !errors.Is(err, legacylist.ErrInvalidCursor) || encoded != "" {
			t.Fatalf("%s: EncodeCursor() = %q, %v", name, encoded, err)
		}
	}
}

func TestDecodeCursorRoundTripsAndKeepsParameterOrder(t *testing.T) {
	t.Parallel()

	original := legacylist.Cursor{
		Parameters: []legacylist.CursorParameter{
			{Name: "type", Value: json.RawMessage(`"pickup"`)},
			{Name: "name", Value: json.RawMessage(`null`)},
			{Name: "distance", Value: json.RawMessage(`12.50`)},
		},
	}
	encoded, err := legacylist.EncodeCursor(original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := legacylist.DecodeCursor(encoded, 1024)
	if err != nil || !reflect.DeepEqual(decoded, original) {
		t.Fatalf("DecodeCursor() = %#v, %v", decoded, err)
	}

	foreign := encodedDocument(t, " {\"_pointsToNextItems\": true, \"locations.id\": \"01HX\"} \n")
	decoded, err = legacylist.DecodeCursor(foreign, 1024)
	if err != nil || !decoded.PointsToNextItems || len(decoded.Parameters) != 1 ||
		decoded.Parameters[0].Name != "locations.id" || string(decoded.Parameters[0].Value) != `"01HX"` {
		t.Fatalf("DecodeCursor(foreign order) = %#v, %v", decoded, err)
	}
}

func TestDecodeCursorRejectsMalformedDocuments(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"empty":               "",
		"padded":              base64.URLEncoding.EncodeToString([]byte(`{"id":1,"_pointsToNextItems":true}`)),
		"standard alphabet":   encodedDocument(t, `{"id":1,"_pointsToNextItems":true}`)[:8] + "+/" + encodedDocument(t, `{"id":1,"_pointsToNextItems":true}`)[10:],
		"not base64":          "%%%",
		"array":               encodedDocument(t, `[1]`),
		"missing direction":   encodedDocument(t, `{"id":1}`),
		"string direction":    encodedDocument(t, `{"id":1,"_pointsToNextItems":"true"}`),
		"duplicate direction": encodedDocument(t, `{"id":1,"_pointsToNextItems":true,"_pointsToNextItems":false}`),
		"no parameters":       encodedDocument(t, `{"_pointsToNextItems":true}`),
		"duplicate parameter": encodedDocument(t, `{"id":1,"id":2,"_pointsToNextItems":true}`),
		"empty name":          encodedDocument(t, `{"":1,"_pointsToNextItems":true}`),
		"trailing data":       encodedDocument(t, `{"id":1,"_pointsToNextItems":true}{}`),
		"truncated":           encodedDocument(t, `{"id":1,"_pointsToNextItems":true`),
		"invalid value":       encodedDocument(t, `{"id":tru,"_pointsToNextItems":true}`),
		"oversized":           encodedDocument(t, `{"id":"`+strings.Repeat("x", 80)+`","_pointsToNextItems":true}`),
	}
	for name, encoded := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cursor, err := legacylist.DecodeCursor(encoded, 96)
			if !errors.Is(err, legacylist.ErrInvalidCursor) || !reflect.DeepEqual(cursor, legacylist.Cursor{}) {
				t.Fatalf("DecodeCursor() = %#v, %v", cursor, err)
			}
		})
	}
	if _, err := legacylist.DecodeCursor(encodedDocument(t, `{"id":1,"_pointsToNextItems":true}`), 0); !errors.Is(err, legacylist.ErrInvalidCursor) {
		t.Fatalf("DecodeCursor(zero bound) error = %v", err)
	}
}

func TestCursorValuesRequiresTheExactParameterSet(t *testing.T) {
	t.Parallel()

	cursor := legacylist.Cursor{Parameters: []legacylist.CursorParameter{
		{Name: "name", Value: json.RawMessage(`"b"`)},
		{Name: "type", Value: json.RawMessage(`"a"`)},
	}}
	values, ok := cursor.Values("type", "name")
	if !ok || len(values) != 2 || string(values[0]) != `"a"` || string(values[1]) != `"b"` {
		t.Fatalf("Values() = %s, %v", values, ok)
	}
	for _, names := range [][]string{{"type"}, {"type", "name", "code"}, {"type", "code"}, {}} {
		if values, ok := cursor.Values(names...); ok || values != nil {
			t.Fatalf("Values(%v) = %s, %v", names, values, ok)
		}
	}
}

func position(value string) []legacylist.CursorParameter {
	return []legacylist.CursorParameter{{Name: "id", Value: json.RawMessage(value)}}
}

func mustEncode(t *testing.T, parameters []legacylist.CursorParameter, next bool) string {
	t.Helper()
	encoded, err := legacylist.EncodeCursor(legacylist.Cursor{Parameters: parameters, PointsToNextItems: next})
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

// The rule table is Laravel's CursorPaginator previousCursor/nextCursor, which
// Location, Track and Postal each reimplemented identically.
func TestPageLinksFollowCursorPaginatorRules(t *testing.T) {
	t.Parallel()

	forward := &legacylist.Cursor{Parameters: position(`3`), PointsToNextItems: true}
	backward := &legacylist.Cursor{Parameters: position(`9`), PointsToNextItems: false}
	self := func(cursor *legacylist.Cursor) string {
		return mustEncode(t, cursor.Parameters, cursor.PointsToNextItems)
	}
	prev := mustEncode(t, position(`4`), false)
	next := mustEncode(t, position(`8`), true)
	tests := []struct {
		name    string
		current *legacylist.Cursor
		items   bool
		hasMore bool
		want    legacylist.Links
	}{
		{name: "single first page", want: legacylist.Links{}, items: true},
		{name: "first page with more", items: true, hasMore: true, want: legacylist.Links{Next: next}},
		{name: "forward middle", current: forward, items: true, hasMore: true, want: legacylist.Links{Self: self(forward), Prev: prev, Next: next}},
		{name: "forward last", current: forward, items: true, want: legacylist.Links{Self: self(forward), Prev: prev}},
		{name: "backward middle", current: backward, items: true, hasMore: true, want: legacylist.Links{Self: self(backward), Prev: prev, Next: next}},
		{name: "backward first", current: backward, items: true, want: legacylist.Links{Self: self(backward), Next: next}},
		{name: "empty first page", want: legacylist.Links{}},
		{name: "empty cursor page", current: forward, want: legacylist.Links{Self: self(forward)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			page := legacylist.Page{Current: tt.current, HasMore: tt.hasMore}
			if tt.items {
				page.First, page.Last = position(`4`), position(`8`)
			}
			links, err := page.Links()
			if err != nil || links != tt.want {
				t.Fatalf("Links() = %#v, %v; want %#v", links, err, tt.want)
			}
			if links.Empty() != (tt.want == legacylist.Links{}) {
				t.Fatalf("Empty() = %v", links.Empty())
			}
		})
	}
}

func TestPageLinksRejectIncompletePositions(t *testing.T) {
	t.Parallel()

	for name, page := range map[string]legacylist.Page{
		"first without last": {First: position(`1`), HasMore: true},
		"invalid last":       {First: position(`1`), Last: []legacylist.CursorParameter{{Name: "id"}}, HasMore: true},
		"more without items": {HasMore: true},
		"invalid current":    {Current: &legacylist.Cursor{}},
	} {
		if links, err := page.Links(); !errors.Is(err, legacylist.ErrInvalidCursor) || links != (legacylist.Links{}) {
			t.Fatalf("%s: Links() = %#v, %v", name, links, err)
		}
	}
}

func TestLinksMarshalAsCursorMetadata(t *testing.T) {
	t.Parallel()

	encoded, err := json.Marshal(legacylist.Links{Self: "s", Next: "n"})
	if err != nil || string(encoded) != `{"self":"s","next":"n"}` {
		t.Fatalf("Marshal() = %s, %v", encoded, err)
	}
	encoded, err = json.Marshal(legacylist.Links{Self: "s", Prev: "p", Next: "n"})
	if err != nil || string(encoded) != `{"self":"s","prev":"p","next":"n"}` {
		t.Fatalf("Marshal() = %s, %v", encoded, err)
	}
}
