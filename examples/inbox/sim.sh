#!/usr/bin/env bash
# Inbox-only Simulator helpers. Launch itself is `mise run ios:run`.
# ping opens the custom scheme and drops a markdown file into Cache/open.jsonl.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

BUNDLE="br.tec.lew.eletrocromo.inbox"
SCHEME="eletrocromo-inbox"

cmd="${1:-sim}"
case "$cmd" in
build)
	GOOS=ios go run "$ROOT/cmd/eletrocromo" run examples/inbox/eletrocromo.json
	;;
ping)
	xcrun simctl openurl booted "${SCHEME}://from-simctl"
	data="$(xcrun simctl get_app_container booted "$BUNDLE" data)"
	cache="$data/Library/Caches"
	mkdir -p "$cache/inbox"
	note="$cache/inbox/from-sim.md"
	printf '# from simctl\nhello inbox\n' >"$note"
	printf '%s\n' "{\"kind\":\"files\",\"paths\":[\"$note\"]}" >>"$cache/open.jsonl"
	echo "opened ${SCHEME}://from-simctl and dropped $note"
	;;
sim)
	"$0" build
	sleep 4
	"$0" ping
	echo "Inbox is on the simulator. Page reloads every 3s."
	;;
*)
	echo "usage: $0 [sim|build|ping]" >&2
	exit 2
	;;
esac
