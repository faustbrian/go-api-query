//nolint:staticcheck // This compatibility test intentionally exercises the deprecated facade.
package apiquerypostgres_test

//lint:file-ignore SA1019 This compatibility test intentionally exercises the deprecated facade.

import (
	"context"
	"errors"
	"testing"

	apiquery "github.com/faustbrian/go-api-query"
	apiquerypostgres "github.com/faustbrian/go-api-query/adapters/postgres"
	legacy "github.com/faustbrian/go-api-query/apiquerypgx"
)

func TestCompilerMatchesCompatibilityPath(t *testing.T) {
	t.Parallel()

	mapping := apiquerypostgres.Mapping{Fields: map[string]string{"id": "orders.id"}}
	compiler, err := apiquerypostgres.NewCompiler(mapping)
	if err != nil {
		t.Fatal(err)
	}
	mapping.Fields["id"] = "changed.bad"
	plan := fieldPlan(t)
	parts, err := compiler.Compile(plan)
	if err != nil || parts.Projection != `"orders"."id"` {
		t.Fatalf("Compile() = (%#v, %v)", parts, err)
	}
	parts.Arguments = append(parts.Arguments, apiquery.StringValue("changed"))
	second, err := compiler.Compile(plan)
	if err != nil || len(second.Arguments) != 0 {
		t.Fatalf("second Compile() = (%#v, %v)", second, err)
	}
	if _, err := apiquerypostgres.NewCompiler(apiquerypostgres.Mapping{Fields: map[string]string{"id": "bad-name"}}); !errors.Is(err, apiquerypostgres.ErrInvalid) {
		t.Fatalf("unsafe mapping error = %v", err)
	}
	if _, err := compiler.Compile(nil); !errors.Is(err, apiquerypostgres.ErrInvalid) {
		t.Fatalf("nil plan error = %v", err)
	}
	var zero apiquerypostgres.Compiler
	if _, err := zero.Compile(plan); !errors.Is(err, apiquerypostgres.ErrInvalid) {
		t.Fatalf("zero compiler error = %v", err)
	}
	if _, err := (*apiquerypostgres.Compiler)(nil).Compile(plan); !errors.Is(err, apiquerypostgres.ErrInvalid) {
		t.Fatalf("nil compiler error = %v", err)
	}
	if apiquerypostgres.ErrInvalid != legacy.ErrInvalid { //nolint:errorlint // Exact sentinel identity is the compatibility contract.
		t.Fatal("ErrInvalid does not share compatibility identity")
	}
}

func fieldPlan(t *testing.T) *apiquery.Plan {
	t.Helper()
	schema, err := apiquery.NewSchema(apiquery.SchemaConfig{Resource: "orders", Revision: "v1", Fields: []apiquery.FieldDefinition{{Name: "id", Type: apiquery.TypeString, Required: true}}})
	if err != nil {
		t.Fatal(err)
	}
	plan, err := apiquery.Compile(context.Background(), schema, apiquery.Request{}, apiquery.CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}
