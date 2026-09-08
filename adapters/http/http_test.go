//nolint:staticcheck // This compatibility test intentionally exercises the deprecated facade.
package apiqueryhttp_test

//lint:file-ignore SA1019 This compatibility test intentionally exercises the deprecated facade.

import (
	"errors"
	"reflect"
	"testing"

	apiqueryhttp "github.com/faustbrian/go-api-query/adapters/http"
	legacy "github.com/faustbrian/go-api-query/apiqueryhttp"
)

func TestParseMatchesCompatibilityPath(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"fields=id,status&sort=-created_at", "fields=id&fields=status", ""} {
		want, wantErr := legacy.Parse(raw, 128)
		got, gotErr := apiqueryhttp.Parse(raw, 128)
		if !reflect.DeepEqual(got, want) || gotErr != wantErr { //nolint:errorlint // Exact sentinel identity is the compatibility contract.
			t.Fatalf("Parse(%q) = (%#v, %v), want (%#v, %v)", raw, got, gotErr, want, wantErr)
		}
	}
	if apiqueryhttp.ErrInvalid != legacy.ErrInvalid || !errors.Is(apiqueryhttp.ErrInvalid, legacy.ErrInvalid) { //nolint:errorlint // Exact sentinel identity is the compatibility contract.
		t.Fatal("ErrInvalid does not share compatibility identity")
	}
}
