.PHONY: docs final-replay postgres

docs:
	go list -f '{{if .GoFiles}}{{.ImportPath}}{{end}}' ./... | xargs -n 1 go doc >/dev/null

final-replay:
	go run ./.verification/cohesion/api-query-v1-final-replay/verify.go

postgres:
	APIQUERY_TEST_DATABASE_URL="$${APIQUERY_TEST_DATABASE_URL:-$${TEST_DATABASE_URL:?}}" \
		go test -count=1 -run '^TestPostgresInjectionResistanceAndStableCursorOrder$$' ./apiquerypgx
