package apiquerylegacylist_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	legacylist "github.com/faustbrian/go-api-query/v5/adapters/legacylist"
)

func FuzzDecodeCursor(f *testing.F) {
	for _, seed := range []string{
		"", "e30", "eyJpZCI6MSwiX3BvaW50c1RvTmV4dEl0ZW1zIjp0cnVlfQ",
		"eyJpZCI6MSwiaWQiOjIsIl9wb2ludHNUb05leHRJdGVtcyI6dHJ1ZX0", "%%%",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, encoded string) {
		cursor, err := legacylist.DecodeCursor(encoded, 512)
		if err != nil {
			if !errors.Is(err, legacylist.ErrInvalidCursor) || !reflect.DeepEqual(cursor, legacylist.Cursor{}) {
				t.Fatalf("DecodeCursor() = %#v, %v", cursor, err)
			}
			return
		}
		reencoded, err := legacylist.EncodeCursor(cursor)
		if err != nil {
			t.Fatalf("decoded cursor cannot be encoded: %v", err)
		}
		again, err := legacylist.DecodeCursor(reencoded, 4096)
		if err != nil || again.PointsToNextItems != cursor.PointsToNextItems || len(again.Parameters) != len(cursor.Parameters) {
			t.Fatalf("re-encoded cursor changed: %#v, %v", again, err)
		}
		for index, parameter := range again.Parameters {
			if parameter.Name != cursor.Parameters[index].Name {
				t.Fatalf("parameter order changed at %d", index)
			}
		}
	})
}

func FuzzParseFilters(f *testing.F) {
	for _, seed := range []string{
		``, `null`, `{"self":[{"attribute":"id","operator":"in","value":[1],"boolean":"or"}]}`,
		`{"self":[{"attribute":"id","operator":"eq"}]}`, `{"zone":[]}`, `[`,
	} {
		f.Add(seed)
	}
	profile := legacylist.Profile{
		Filters:  []legacylist.Resource{{Name: "self", Attributes: []string{"id", "name"}}},
		MaxBytes: 1024,
	}
	f.Fuzz(func(t *testing.T, raw string) {
		filters, err := profile.ParseFilters(json.RawMessage(raw))
		if err != nil {
			var listErr *legacylist.Error
			if !errors.As(err, &listErr) || listErr.Kind < legacylist.KindShape || listErr.Kind > legacylist.KindValue || filters != nil {
				t.Fatalf("ParseFilters() = %v, %v", filters, err)
			}
			return
		}
		for _, filter := range filters {
			if filter.Resource != "self" || (filter.Attribute != "id" && filter.Attribute != "name") ||
				(filter.Boolean != legacylist.And && filter.Boolean != legacylist.Or) {
				t.Fatalf("undeclared filter accepted: %#v", filter)
			}
			if _, ok := legacylist.ParseOperator(string(filter.Operator)); !ok {
				t.Fatalf("unknown operator accepted: %q", filter.Operator)
			}
		}
	})
}
