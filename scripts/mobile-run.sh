#!/usr/bin/env bash
# Build one eletrocromo.json app and launch it.
# usage: mobile-run.sh ios|android path/to/eletrocromo.json
set -euo pipefail

platform="${1:-}"
config="${2:-}"
if [[ "$platform" != ios && "$platform" != android ]] || [[ -z "$config" ]]; then
	echo "usage: $0 ios|android path/to/eletrocromo.json" >&2
	exit 2
fi

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
if [[ ! -f "$config" ]]; then
	echo "config not found: $config" >&2
	exit 1
fi
config="$(cd "$(dirname "$config")" && pwd)/$(basename "$config")"
slug="$(basename "$(dirname "$config")")"

meta="$(python3 - "$config" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    cfg = json.load(f)
pkg = str(cfg.get("package_id", "")).strip()
if not pkg:
    sys.exit("package_id is required")
name = str(cfg.get("app_name") or pkg.rsplit(".", 1)[-1]).strip()
print(pkg)
print(name)
PY
)"
package_id="${meta%%$'\n'*}"
app_name="${meta#*$'\n'}"

case "$platform" in
ios)
	app="$root/dist/${slug}.app"
	go run ./cmd/eletrocromo build ios \
		--config "$config" \
		--out "$app" \
		--workdir "$root/dist/ios-${slug}"
	if ! xcrun simctl list devices booted | grep -q iPhone; then
		udid="$(xcrun simctl list devices available | awk -F '[()]' '/iPhone/{print $2; exit}')"
		if [[ -z "$udid" ]]; then
			echo "no available iPhone simulator" >&2
			exit 1
		fi
		xcrun simctl boot "$udid"
		open -a Simulator
		xcrun simctl bootstatus "$udid" -b
	fi
	xcrun simctl install booted "$app"
	xcrun simctl launch booted "$package_id"
	echo "launched ${app_name} (${package_id}) from ${app}"
	;;
android)
	apk="$root/dist/${slug}-debug.apk"
	yes | sdkmanager --licenses >/dev/null || true
	sdkmanager --install "platforms;android-35" "build-tools;35.0.0"
	go run ./cmd/eletrocromo build android \
		--config "$config" \
		--out "$apk" \
		--workdir "$root/dist/android-${slug}"
	adb install -r "$apk"
	adb shell am start -n "${package_id}/.MainActivity"
	echo "launched ${app_name} (${package_id}) from ${apk}"
	;;
esac
