package apiqueryvalidation_test

import (
	"context"
	"errors"
	"fmt"

	apiqueryvalidation "github.com/faustbrian/go-api-query/v5/adapters/validation"
	validation "github.com/faustbrian/go-validation/v3"
)

// Applications route cancellation and deadline outcomes before projecting a
// query failure into a validation report.
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
