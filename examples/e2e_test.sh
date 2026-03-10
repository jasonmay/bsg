#!/bin/bash
set -euo pipefail

# E2E test for BSG — exercises specs, edges, links, check-file, custom hooks, and prime.
# Requires: bsg binary on PATH (go install ./... first)

PASS=0
FAIL=0

assert_contains() {
	local label="$1" output="$2" expected="$3"
	if echo "$output" | grep -qF "$expected"; then
		PASS=$((PASS + 1))
	else
		FAIL=$((FAIL + 1))
		echo "FAIL: $label"
		echo "  expected to contain: $expected"
		echo "  got: $output"
	fi
}

assert_not_contains() {
	local label="$1" output="$2" unexpected="$3"
	if echo "$output" | grep -qF "$unexpected"; then
		FAIL=$((FAIL + 1))
		echo "FAIL: $label"
		echo "  expected NOT to contain: $unexpected"
		echo "  got: $output"
	else
		PASS=$((PASS + 1))
	fi
}

assert_exit_code() {
	local label="$1" expected="$2" actual="$3"
	if [ "$actual" -eq "$expected" ]; then
		PASS=$((PASS + 1))
	else
		FAIL=$((FAIL + 1))
		echo "FAIL: $label — expected exit $expected, got $actual"
	fi
}

# Setup temp project
ROOT=$(mktemp -d)
trap "rm -rf $ROOT" EXIT
cd "$ROOT"

mkdir -p src/api src/web tests
echo 'package api' > src/api/handler.go
echo 'func ServeAPI() {}' >> src/api/handler.go
echo 'package web' > src/web/render.go
echo 'func Render() {}' >> src/web/render.go
echo 'package tests' > tests/api_test.go
echo 'func TestAPI() {}' >> tests/api_test.go

# Init BSG
bsg init

echo "=== Test: spec creation ==="
ID1=$(bsg add "API must validate input" --type constraint --tag api,validation --body "Validate all API inputs before processing")
assert_contains "add returns ID" "$ID1" "bsg-"

ID2=$(bsg add "Web rendering" --type behavior --tag web --body "Render HTML pages")
assert_contains "add behavior" "$ID2" "bsg-"

ID3=$(bsg add "All features must have tests" --type invariant --body "Every feature spec must have at least one test link")
assert_contains "add invariant" "$ID3" "bsg-"

echo "=== Test: spec show ==="
OUT=$(bsg show "$ID1")
assert_contains "show has name" "$OUT" "API must validate input"
assert_contains "show has type" "$OUT" "constraint"
assert_contains "show has status" "$OUT" "draft"

echo "=== Test: status transitions ==="
bsg status "$ID1" accepted
bsg status "$ID2" accepted

OUT=$(bsg show "$ID1")
assert_contains "status accepted" "$OUT" "accepted"

echo "=== Test: trace with auto-transition ==="
bsg trace "$ID1" --file src/api/handler.go:ServeAPI
OUT=$(bsg show "$ID1")
assert_contains "auto-transition to implemented" "$OUT" "implemented"

bsg trace "$ID2" --file src/web/render.go
bsg trace "$ID1" --file tests/api_test.go --as tests

echo "=== Test: edges ==="
bsg link "$ID1" --refines "$ID3"
bsg link "$ID2" --refines "$ID3"

OUT=$(bsg show "$ID3")
assert_contains "edge visible on parent" "$OUT" "$ID1"

echo "=== Test: check-file shows direct specs ==="
OUT=$(bsg check-file src/api/handler.go)
assert_contains "check-file direct spec" "$OUT" "$ID1"
assert_contains "check-file shows name" "$OUT" "API must validate input"

echo "=== Test: check-file edge traversal ==="
OUT=$(bsg check-file src/api/handler.go)
assert_contains "check-file upstream section" "$OUT" "Upstream specs"
assert_contains "check-file shows parent via edges" "$OUT" "$ID3"
assert_contains "check-file shows parent name" "$OUT" "All features must have tests"

echo "=== Test: check-file on unlinked file ==="
echo 'package main' > unlinked.go
OUT=$(bsg check-file unlinked.go)
assert_contains "unlinked file returns empty" "$OUT" ""

echo "=== Test: inspect ==="
OUT=$(bsg inspect src/api/handler.go)
assert_contains "inspect shows spec" "$OUT" "$ID1"
assert_contains "inspect shows symbol" "$OUT" "ServeAPI"

echo "=== Test: inspect --recursive ==="
OUT=$(bsg inspect src/api/handler.go --recursive)
assert_contains "inspect recursive shows upstream" "$OUT" "Upstream"
assert_contains "inspect recursive shows parent" "$OUT" "$ID3"

echo "=== Test: inspect directory ==="
OUT=$(bsg inspect src/)
assert_contains "inspect dir has handler" "$OUT" "handler.go"
assert_contains "inspect dir has render" "$OUT" "render.go"

echo "=== Test: summarize ==="
OUT=$(bsg summarize)
assert_contains "summarize has spec" "$OUT" "$ID1"
assert_contains "summarize has files" "$OUT" "handler.go"

echo "=== Test: tags ==="
OUT=$(bsg tags)
assert_contains "tags has api" "$OUT" "api"
assert_contains "tags has web" "$OUT" "web"
assert_contains "tags has validation" "$OUT" "validation"

echo "=== Test: tree ==="
OUT=$(bsg tree)
assert_contains "tree has src" "$OUT" "src"
assert_contains "tree has handler" "$OUT" "handler.go"

echo "=== Test: prime ==="
OUT=$(bsg prime)
assert_contains "prime has coverage" "$OUT" "Coverage"
assert_contains "prime has constraint" "$OUT" "API must validate input"
assert_contains "prime has invariant" "$OUT" "All features must have tests"

echo "=== Test: prime --compact ==="
OUT=$(bsg prime --compact)
assert_contains "compact has BSG" "$OUT" "BSG:"

echo "=== Test: prime --json ==="
OUT=$(bsg prime --json)
assert_contains "json has total" "$OUT" '"total"'

echo "=== Test: custom check-file hooks ==="
mkdir -p .bsg/hooks/check-file.d
cat > .bsg/hooks/check-file.d/warn-api.sh <<'HOOK'
#!/bin/bash
FILE="$1"
if echo "$FILE" | grep -q "^src/api/"; then
    echo "# [custom] API file edited — ensure tests are updated"
fi
HOOK
chmod +x .bsg/hooks/check-file.d/warn-api.sh

OUT=$(bsg check-file src/api/handler.go)
assert_contains "custom hook fires for api" "$OUT" "[custom] API file edited"

OUT=$(bsg check-file src/web/render.go)
assert_not_contains "custom hook silent for non-api" "$OUT" "[custom] API file"

echo "=== Test: custom hook in prime output ==="
OUT=$(bsg prime)
assert_contains "prime shows custom hooks" "$OUT" "Custom check-file hooks"
assert_contains "prime lists hook name" "$OUT" "warn-api.sh"

echo "=== Test: multiple custom hooks ==="
cat > .bsg/hooks/check-file.d/always-warn.sh <<'HOOK'
#!/bin/bash
echo "# [always] remember to update docs"
HOOK
chmod +x .bsg/hooks/check-file.d/always-warn.sh

OUT=$(bsg check-file src/api/handler.go)
assert_contains "both hooks fire" "$OUT" "[custom] API file edited"
assert_contains "second hook fires" "$OUT" "[always] remember to update docs"

echo "=== Test: non-executable hooks are skipped ==="
echo '#!/bin/bash' > .bsg/hooks/check-file.d/disabled.sh
echo 'echo "should not appear"' >> .bsg/hooks/check-file.d/disabled.sh
# deliberately not chmod +x

OUT=$(bsg check-file src/api/handler.go)
assert_not_contains "non-executable hook skipped" "$OUT" "should not appear"

echo "=== Test: link health — missing symbol ==="
ID4=$(bsg add "Extra feature" --type behavior --body "test link health" --tag test)
bsg status "$ID4" accepted
bsg trace "$ID4" --file src/web/render.go:MissingFunc
OUT=$(bsg check-file src/web/render.go)
assert_contains "missing symbol warning" "$OUT" "MissingFunc not found"

echo "=== Test: untrace ==="
bsg untrace "$ID4" src/web/render.go
OUT=$(bsg check-file src/web/render.go)
assert_not_contains "untrace removes spec" "$OUT" "$ID4"

echo "=== Test: delete ==="
bsg delete "$ID4"
OUT=$(bsg summarize)
assert_not_contains "delete removes spec" "$OUT" "Extra feature"

echo "=== Test: sync rebuilds from files ==="
bsg sync
OUT=$(bsg summarize)
assert_contains "sync preserves specs" "$OUT" "$ID1"

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
if [ "$FAIL" -gt 0 ]; then
	exit 1
fi
