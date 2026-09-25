#!/usr/bin/env bash
# Build one eletrocromo.json app and launch it.
# usage: mobile-run.sh ios|android|mac|linux|windows path/to/eletrocromo.json
# linux and windows are cross-compiled with GOOS/GOARCH (CGO_ENABLED=0).
# GOARCH defaults to this machine; override it in the environment.
set -euo pipefail

platform="${1:-}"
config="${2:-}"
case "$platform" in
ios | android | mac | linux | windows) ;;
*)
	echo "usage: $0 ios|android|mac|linux|windows path/to/eletrocromo.json" >&2
	exit 2
	;;
esac
if [[ -z "$config" ]]; then
	echo "usage: $0 ios|android|mac|linux|windows path/to/eletrocromo.json" >&2
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
main = str(cfg.get("go_main") or ".").strip() or "."
print(pkg)
print(name)
print(main)
PY
)"
package_id="${meta%%$'\n'*}"
rest="${meta#*$'\n'}"
app_name="${rest%%$'\n'*}"
go_main="${rest#*$'\n'}"
mod_dir="$(cd "$(dirname "$config")" && cd "$go_main" && pwd)"

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
mac)
	app="$root/dist/${slug}.app"
	go run ./cmd/eletrocromo build macos \
		--config "$config" \
		--out "$app" \
		--workdir "$root/dist/macos-${slug}"
	open "$app"
	echo "launched ${app_name} (${package_id}) from ${app}"
	;;
linux | windows)
	arch="${GOARCH:-$(go env GOARCH)}"
	icons="$root/dist/icons"
	go run ./cmd/eletrocromo build icons --config "$config" --output "$icons"
	mkdir -p "$root/dist"
	bin="$root/dist/${slug}-${platform}-${arch}"
	cleanup=()
	if [[ "$platform" == windows ]]; then
		bin="${bin}.exe"
		syso="$mod_dir/rsrc_windows_${arch}.syso"
		go run ./scripts/winsyso -arch "$arch" -ico "$icons/windows/icon.ico" -o "$syso"
		cleanup+=("$syso")
	else
		cp "$icons/linux/icon-256.png" "$mod_dir/eletrocromo_icon.png"
		pkg="$(awk '/^package /{print $2; exit}' "$mod_dir"/*.go)"
		cat >"$mod_dir/eletrocromo_icon.go" <<EOF
package ${pkg}

import _ "embed"

//go:embed eletrocromo_icon.png
var eletrocromoIconPNG []byte

func init() {
	if len(eletrocromoIconPNG) == 0 {
		panic("eletrocromo icon")
	}
}
EOF
		cleanup+=("$mod_dir/eletrocromo_icon.png" "$mod_dir/eletrocromo_icon.go")
	fi
	trap 'rm -f "${cleanup[@]}"' EXIT
	echo "building ${app_name} GOOS=${platform} GOARCH=${arch}"
	(
		cd "$mod_dir"
		GOOS="$platform" GOARCH="$arch" CGO_ENABLED=0 go build -o "$bin" .
	)
	rm -f "${cleanup[@]}"
	trap - EXIT
	if [[ "$(go env GOHOSTOS)" == "$platform" && "$(go env GOHOSTARCH)" == "$arch" ]]; then
		echo "running ${bin}"
		exec "$bin"
	fi
	echo "built ${bin}"
	;;
esac
