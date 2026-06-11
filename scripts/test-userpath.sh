#!/usr/bin/env bash
# User-path smoke test for keysync CLI (real binary, isolated HOME).
#
# Run from repo root:
#   ./scripts/test-userpath.sh
#
# Or after install:
#   KEYSYNC_BIN=/opt/homebrew/bin/keysync ./scripts/test-userpath.sh
#
# Uses --store fallback so no OS keychain is required.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${KEYSYNC_BIN:-$ROOT/bin/keysync}"
HOME_DIR="$(mktemp -d)"
WORK_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$HOME_DIR" "$WORK_DIR"
}
trap cleanup EXIT

export HOME="$HOME_DIR"

if [[ ! -x "$BIN" ]]; then
  echo "Building keysync..."
  make -C "$ROOT" build
  BIN="$ROOT/bin/keysync"
fi

cat >"$WORK_DIR/.keysync.json" <<'EOF'
{
  "repos": {
    "test-org/test-repo": {
      "project": "test-app"
    }
  }
}
EOF

run() {
  echo "+ keysync $*" >&2
  (cd "$WORK_DIR" && "$BIN" --store fallback "$@")
}

echo "== set / get with -p geo =="
run set LANGCHAIN_API_KEY=userpath_test -p geo
out="$(run get LANGCHAIN_API_KEY -p geo -u)"
[[ "$out" == "LANGCHAIN_API_KEY=userpath_test" ]] || {
  echo "unexpected get output: $out"
  exit 1
}

echo "== list -p (project names) =="
run set A_KEY=1 -p alpha
run set Z_KEY=1 -p zebra
out="$(run list -p)"
echo "$out" | grep -q alpha
echo "$out" | grep -q zebra
! echo "$out" | grep -q 'A_KEY' || {
  echo "list -p should not show secret keys"
  exit 1
}

echo "== list --project hyperdx =="
run set H_KEY=1 -p hyperdx
run set O_KEY=1 -p other
out="$(run list --project hyperdx)"
echo "$out" | grep -q H_KEY
! echo "$out" | grep -q O_KEY || {
  echo "list --project hyperdx should not show other project keys"
  exit 1
}

echo "== export -p geo =="
run set EXPORT_KEY=exported -p geo
out="$(run export EXPORT_KEY -p geo)"
echo "$out" | grep -q 'export EXPORT_KEY='

echo "== rm -p geo =="
run set DELETE_ME=1 -p geo
run rm DELETE_ME -p geo
if run get DELETE_ME -p geo -u 2>/dev/null; then
  echo "expected get after rm to fail"
  exit 1
fi

echo "All user-path checks passed."
