#!/bin/sh
set -eu

output=$(go tool apidiff -m api/v1.txt github.com/faustbrian/go-api-query)
if [ -n "$output" ]; then
	printf '%s\n' "$output"
	exit 1
fi
