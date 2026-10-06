//nolint:staticcheck // This coexistence test intentionally exercises deprecated compatibility facades.
package apiquery_test

//lint:file-ignore SA1019 This coexistence test intentionally exercises deprecated compatibility facades.

import (
	"testing"

	newhttp "github.com/faustbrian/go-api-query/v4/adapters/http"
	newjsonapi "github.com/faustbrian/go-api-query/v4/adapters/jsonapi"
	newjsonrpc "github.com/faustbrian/go-api-query/v4/adapters/jsonrpc"
	newpostgres "github.com/faustbrian/go-api-query/v4/adapters/postgres"
	newvalidation "github.com/faustbrian/go-api-query/v4/adapters/validation"
	legacyhttp "github.com/faustbrian/go-api-query/v4/apiqueryhttp"
	legacyjsonapi "github.com/faustbrian/go-api-query/v4/apiqueryjsonapi"
	legacypostgres "github.com/faustbrian/go-api-query/v4/apiquerypgx"
	legacyjsonrpc "github.com/faustbrian/go-api-query/v4/apiqueryrpc"
	legacyvalidation "github.com/faustbrian/go-api-query/v4/apiqueryvalidation"
)

func TestLegacyAndSuccessorAdaptersCompileTogether(t *testing.T) {
	t.Parallel()

	_ = legacyhttp.Parse
	_ = legacyjsonapi.FromQuery
	_ = legacypostgres.NewCompiler
	_ = legacyjsonrpc.Parse
	_ = legacyvalidation.Report

	_ = newhttp.Parse
	_ = newjsonapi.FromQuery
	_ = newpostgres.NewCompiler
	_ = newjsonrpc.Parse
	_ = newvalidation.Report
}
