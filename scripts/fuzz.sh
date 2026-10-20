#!/bin/sh
# Runs every Fuzz* target in the module for FUZZTIME (default 30s each).
set -eu
fuzztime="${FUZZTIME:-30s}"
for pkg in $(go list ./...); do
	dir=$(go list -f '{{.Dir}}' "$pkg")
	targets=$(grep -h -o '^func Fuzz[A-Za-z0-9_]*' "$dir"/*_test.go 2>/dev/null | sed 's/^func //' || true)
	for target in $targets; do
		echo "fuzz $pkg $target ($fuzztime)"
		go test "$pkg" -run '^$' -fuzz "^${target}\$" -fuzztime "$fuzztime"
	done
done
