package apiquerylegacylist_test

import (
	"encoding/json"
	"errors"
	"fmt"

	legacylist "github.com/faustbrian/go-api-query/v5/adapters/legacylist"
)

func ExampleProfile_ParseFilters() {
	profile := legacylist.Profile{
		Filters:  []legacylist.Resource{{Name: "self", Attributes: []string{"status", "created_at"}}},
		Sorts:    []legacylist.Resource{{Name: "self", Attributes: []string{"created_at"}}},
		MaxBytes: 64 << 10,
	}
	filters, err := profile.ParseFilters(json.RawMessage(`{"self": [
		{"attribute": "status", "operator": "in", "value": ["delivered", "returned"]},
		{"attribute": "created_at", "operator": "is_not_null", "boolean": "or"}
	]}`))
	for _, filter := range filters {
		fmt.Println(filter.Attribute, filter.Operator, filter.Boolean, filter.ValidateValue())
	}
	fmt.Println(err)

	_, err = profile.ParseSorts(json.RawMessage(`{"self": [{"attribute": "status", "direction": "asc"}]}`))
	var listErr *legacylist.Error
	if errors.As(err, &listErr) {
		fmt.Println(listErr.Kind, listErr.Unknown, listErr.Allowed)
	}
	// Output:
	// status in and <nil>
	// created_at is_not_null or <nil>
	// <nil>
	// undeclared attribute status [created_at]
}

func ExamplePage_Links() {
	current, _ := legacylist.DecodeCursor("eyJpZCI6MTAsIl9wb2ludHNUb05leHRJdGVtcyI6dHJ1ZX0", 2048)
	values, ok := current.Values("id")
	fmt.Println(string(values[0]), ok, current.PointsToNextItems)

	links, _ := legacylist.Page{
		Current: &current,
		First:   []legacylist.CursorParameter{{Name: "id", Value: json.RawMessage(`11`)}},
		Last:    []legacylist.CursorParameter{{Name: "id", Value: json.RawMessage(`20`)}},
		HasMore: true,
	}.Links()
	document, _ := json.Marshal(map[string]any{"page": map[string]any{"cursor": links}})
	fmt.Println(string(document))
	// Output:
	// 10 true true
	// {"page":{"cursor":{"self":"eyJpZCI6MTAsIl9wb2ludHNUb05leHRJdGVtcyI6dHJ1ZX0","prev":"eyJpZCI6MTEsIl9wb2ludHNUb05leHRJdGVtcyI6ZmFsc2V9","next":"eyJpZCI6MjAsIl9wb2ludHNUb05leHRJdGVtcyI6dHJ1ZX0"}}}
}
