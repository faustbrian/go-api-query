package apiquery_test

import (
	"context"
	"net/url"
	"testing"

	apiquery "github.com/faustbrian/go-api-query/v5"
	apiqueryjsonapi "github.com/faustbrian/go-api-query/v5/adapters/jsonapi"
	jsonapi "github.com/faustbrian/go-jsonapi/v2"
)

func TestJSONAPIV2AdapterComposesFiniteCursorQuery(t *testing.T) {
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
		page, parseErr := pagination.Parse(family)
		return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: page.Size}, parseErr
	}
	request, err := apiqueryjsonapi.FromQuery(query, apiqueryjsonapi.Config{Resource: "orders", DecodeFilter: decodeFilter, DecodePage: decodePage})
	if err != nil {
		t.Fatal(err)
	}
	if request.Page.Mode != apiquery.PageCursor || request.Page.Size != 20 {
		t.Fatal("finite JSONAPI page request was not preserved")
	}
	if _, err := apiquery.Compile(context.Background(), transportSchema(t), request, apiquery.CompileOptions{}); err != nil {
		t.Fatal(err)
	}
}
