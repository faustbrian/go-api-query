//nolint:staticcheck // This compatibility test intentionally exercises the deprecated facade.
package apiqueryvalidation_test

//lint:file-ignore SA1019 This compatibility test intentionally exercises the deprecated facade.

import (
	"context"
	"errors"
	"fmt"
	"testing"

	apiqueryvalidation "github.com/faustbrian/go-api-query/adapters/validation"
	legacy "github.com/faustbrian/go-api-query/apiqueryvalidation"
	validation "github.com/faustbrian/go-validation"
)

func TestReportMatchesCompatibilityPath(t *testing.T) {
	t.Parallel()

	err := errors.New("unsafe details")
	want := legacy.Report(err, validation.DefaultLimits())
	got := apiqueryvalidation.Report(err, validation.DefaultLimits())
	if fmt.Sprint(got.Violations()) != fmt.Sprint(want.Violations()) {
		t.Fatalf("Report() = %#v, want %#v", got.Violations(), want.Violations())
	}
}

func ExampleReport_contextTerminalOrdering() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	queryErr := errors.New("query failed")
	if err := ctx.Err(); err != nil {
		fmt.Println("canceled before validation projection")
		return
	}
	_ = apiqueryvalidation.Report(queryErr, validation.DefaultLimits())
	// Output: canceled before validation projection
}
