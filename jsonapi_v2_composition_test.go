//nolint:staticcheck // The supported compatibility adapter is part of this contract.
package apiquery_test

//lint:file-ignore SA1019 The supported compatibility adapter is part of this contract.

import (
	"context"
	"net/url"
	"testing"

	apiquery "github.com/faustbrian/go-api-query/v4"
	preferred "github.com/faustbrian/go-api-query/v4/adapters/jsonapi"
	legacy "github.com/faustbrian/go-api-query/v4/apiqueryjsonapi"
	jsonapi "github.com/faustbrian/go-jsonapi/v2"
)

func TestJSONAPIV2BothAdaptersComposeFiniteCursorQuery(t *testing.T) {
	query, err := jsonapi.ParseQuery(url.Values{
		"fields[orders]": {"status"}, "filter[status]": {"paid"},
		"sort": {"-created_at"}, "page[size]": {"20"},
	})
	if err != nil {
		t.Fatal(err)
	}
	pagination, err := jsonapi.NewCursorPagination(jsonapi.CursorPaginationConfig{DefaultSize: 20, MaxSize: 100})
	if err != nil {
		t.Fatal(err)
	}
	decodeFilter := func(family jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
		return &apiquery.FilterExpr{Predicate: &apiquery.Predicate{
			Name: "status", Operator: apiquery.OpEqual,
			Values: []apiquery.Value{apiquery.StringValue(family["filter[status]"][0])},
		}}, nil
	}
	decodePage := func(family jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
		page, err := pagination.Parse(family)
		return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: page.Size}, err
	}
	first, err := legacy.FromQuery(query, legacy.Config{Resource: "orders", DecodeFilter: decodeFilter, DecodePage: decodePage})
	if err != nil {
		t.Fatal(err)
	}
	second, err := preferred.FromQuery(query, preferred.Config{Resource: "orders", DecodeFilter: decodeFilter, DecodePage: decodePage})
	if err != nil {
		t.Fatal(err)
	}
	schema := transportSchema(t)
	var canonical []byte
	for index, request := range []apiquery.Request{first, second} {
		if request.Page.Mode != apiquery.PageCursor || request.Page.Size != 20 {
			t.Fatal("finite JSONAPI page request was not preserved")
		}
		plan, err := apiquery.Compile(context.Background(), schema, request, apiquery.CompileOptions{})
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := plan.Canonical()
		if err != nil {
			t.Fatal(err)
		}
		if index == 0 {
			canonical = encoded
		} else if string(encoded) != string(canonical) {
			t.Fatal("JSONAPI adapter variants compiled different canonical plans")
		}
	}
}
