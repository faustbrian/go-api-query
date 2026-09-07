//nolint:staticcheck // This compatibility test intentionally exercises the deprecated facade.
package apiqueryjsonapi_test

//lint:file-ignore SA1019 This compatibility test intentionally exercises the deprecated facade.

import (
	"errors"
	"reflect"
	"testing"

	apiquery "github.com/faustbrian/go-api-query"
	apiqueryjsonapi "github.com/faustbrian/go-api-query/adapters/jsonapi"
	legacy "github.com/faustbrian/go-api-query/apiqueryjsonapi"
	jsonapi "github.com/faustbrian/go-jsonapi"
)

func TestFromQueryMatchesCompatibilityPath(t *testing.T) {
	t.Parallel()

	query := jsonapi.Query{Fields: map[string][]string{"orders": {"id"}}, Filter: jsonapi.ParameterFamily{"filter[status]": {"paid"}}, Page: jsonapi.ParameterFamily{"page[size]": {"10"}}}
	decodeFilter := func(jsonapi.ParameterFamily) (*apiquery.FilterExpr, error) {
		return &apiquery.FilterExpr{Predicate: &apiquery.Predicate{Name: "status", Operator: apiquery.OpEqual, Values: []apiquery.Value{apiquery.StringValue("paid")}}}, nil
	}
	decodePage := func(jsonapi.ParameterFamily) (apiquery.PageRequest, error) {
		return apiquery.PageRequest{Mode: apiquery.PageCursor, Size: 10}, nil
	}
	want, wantErr := legacy.FromQuery(query, legacy.Config{Resource: "orders", DecodeFilter: decodeFilter, DecodePage: decodePage})
	got, gotErr := apiqueryjsonapi.FromQuery(query, apiqueryjsonapi.Config{Resource: "orders", DecodeFilter: decodeFilter, DecodePage: decodePage})
	if !reflect.DeepEqual(got, want) || gotErr != wantErr { //nolint:errorlint // Exact sentinel identity is the compatibility contract.
		t.Fatalf("FromQuery() = (%#v, %v), want (%#v, %v)", got, gotErr, want, wantErr)
	}
	_, gotErr = apiqueryjsonapi.FromQuery(jsonapi.Query{Filter: jsonapi.ParameterFamily{"filter[x]": {"1"}}}, apiqueryjsonapi.Config{Resource: "orders"})
	if !errors.Is(gotErr, apiqueryjsonapi.ErrUnsupported) || apiqueryjsonapi.ErrInvalid != legacy.ErrInvalid || apiqueryjsonapi.ErrUnsupported != legacy.ErrUnsupported { //nolint:errorlint // Exact sentinel identity is the compatibility contract.
		t.Fatalf("sentinel behavior = (%v, %v, %v)", gotErr, apiqueryjsonapi.ErrInvalid, apiqueryjsonapi.ErrUnsupported)
	}
}
