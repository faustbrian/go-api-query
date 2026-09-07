//nolint:staticcheck // This compatibility test intentionally exercises the deprecated facade.
package apiqueryjsonrpc_test

//lint:file-ignore SA1019 This compatibility test intentionally exercises the deprecated facade.

import (
	"errors"
	"reflect"
	"testing"

	apiquery "github.com/faustbrian/go-api-query"
	apiqueryjsonrpc "github.com/faustbrian/go-api-query/adapters/jsonrpc"
	legacy "github.com/faustbrian/go-api-query/apiqueryrpc"
)

func TestParamsAndDescriptorMatchCompatibilityPath(t *testing.T) {
	t.Parallel()

	data := []byte(`{"schema_revision":"v1","fields":["id"],"includes":[],"sorts":[{"name":"id","direction":"asc"}],"page":{"mode":"cursor","size":10}}`)
	want, wantErr := legacy.Parse(data, 512)
	got, gotErr := apiqueryjsonrpc.Parse(data, 512)
	if gotErr != wantErr || !reflect.DeepEqual(got.Request(), want.Request()) { //nolint:errorlint // Exact sentinel identity is the compatibility contract.
		t.Fatalf("Parse() = (%#v, %v), want (%#v, %v)", got, gotErr, want, wantErr)
	}
	fields, fieldsPresent := got.Request().Fields.Value()
	if !fieldsPresent || len(fields) != 1 {
		t.Fatalf("Request fields = %#v", fields)
	}
	fields[0] = "changed"
	fieldsAgain, fieldsAgainPresent := got.Request().Fields.Value()
	if !fieldsAgainPresent || len(fieldsAgain) != 1 || fieldsAgain[0] != "id" {
		t.Fatal("Request retained returned field mutation")
	}
	if _, err := apiqueryjsonrpc.Parse([]byte(`{"unknown":true}`), 128); !errors.Is(err, apiqueryjsonrpc.ErrInvalid) || apiqueryjsonrpc.ErrInvalid != legacy.ErrInvalid { //nolint:errorlint // Exact sentinel identity is the compatibility contract.
		t.Fatalf("invalid parse = %v", err)
	}
	descriptor := apiqueryjsonrpc.OpenRPCContentDescriptor()
	descriptor.Schema["type"] = "changed"
	if next := apiqueryjsonrpc.OpenRPCContentDescriptor(); next.Name != "query" || next.Required || next.Schema["type"] != "object" {
		t.Fatalf("descriptor = %#v", next)
	}
	_ = apiqueryjsonrpc.Params{Filter: &apiquery.FilterExpr{}}.Request()
}
