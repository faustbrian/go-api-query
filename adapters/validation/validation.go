// Package apiqueryvalidation projects API Query failures into immutable
// go-validation reports. Report is stateless, retains no caller values, and is
// safe for concurrent independent calls. Applications must route cancellation
// and deadline outcomes before projection; this package does not own context,
// transport envelopes, localization, or validation execution.
//
//nolint:staticcheck // This successor facade intentionally delegates to the deprecated compatibility implementation.
package apiqueryvalidation

//lint:file-ignore SA1019 This successor facade intentionally delegates to the supported compatibility implementation.

import (
	legacy "github.com/faustbrian/go-api-query/apiqueryvalidation"
	validation "github.com/faustbrian/go-validation"
)

// Report converts query failures into a validation report.
func Report(err error, limits validation.Limits) validation.Report {
	return legacy.Report(err, limits)
}
