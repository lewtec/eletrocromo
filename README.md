# eletrocromo

A simpler approach to desktop apps without Electron or Wails: pure Go HTTP
handler + local loopback server + **Helium** `--app` window.

See [SPEC.md](SPEC.md) for the product contract.

## Architecture

On start, eletrocromo wraps your `http.Handler` with always-on token auth, binds
loopback, and opens the UI with Helium `--app` and a **per-app profile**:

1. Require reverse-domain `App.ID` (e.g. `br.tec.lew.myapp`) → isolated
   `--user-data-dir` under the OS data dir (`…/eletrocromo/profiles/<id>`)
2. **Helium** on `PATH`, else ensure via **workspaced**
   (`tool which helium-browser helium`), bootstrapping workspaced if needed
3. Start server only after Helium resolves; fail if Helium exits on startup
4. Never Chrome/Edge/system browser

```go
app := eletrocromo.App{
    ID:      "br.tec.lew.myapp", // reverse-domain; also future APK package name
    Handler: myHandler,
    Context: ctx, // cancel to shut down
}
log.Fatal(app.Run())
```

Set `ELETROCROMO_NO_ENSURE=1` to disable network ensure (tests/CI).
Set `ELETROCROMO_WORKSPACED=/path/to/workspaced` to pin the ensure helper binary.

## Try it

Each example is its own Go module under `examples/*` (`go -C examples/<name> run .`).

Template counter dogfood (Helium-first launch):

```bash
mise run example:counter
```

Ctrl+C in the terminal stops the process. `+` / `−` / reset hit the local server via form POST.

### Astro + orvalho workers

Astro **SSR** (Cloudflare adapter; cat fact in frontmatter per request) hosted by [orvalho `pkg/workers`](https://github.com/lucasew/orvalho) and opened via eletrocromo.
Guest JS + assets are **`//go:embed`’d** after `mise run build` (no runtime esbuild).
Tools live in [`examples/astro/mise.toml`](examples/astro/mise.toml).

```bash
cd examples/astro
mise install
mise run build   # astro + orvalho pre-bundle → embed/
mise run run     # go run with embedded guest
# from repo root: mise run example:astro:build && mise run example:astro
```

Needs a local orvalho checkout (see `examples/astro/go.mod` `replace`). Details: [examples/astro/README.md](examples/astro/README.md).

## CLI (`cmd/eletrocromo`)

Cobra tooling binary (separate from the importable library):

```bash
go run ./cmd/eletrocromo --help
go run ./cmd/eletrocromo android create --help
```

### Android APK (straight build)

Standard app config is `eletrocromo.json` next to your Go main (see
`examples/counter/eletrocromo.json`). One command scaffolds the WebView host,
cross-compiles multiarch Go (`GOOS=android`), and runs Gradle:

```bash
# from the app module:
cd examples/counter
go run ../../cmd/eletrocromo android build
# → dist/counter-debug.apk (package id from eletrocromo.json)

# from repo root:
go run ./cmd/eletrocromo android build \
  --config examples/counter/eletrocromo.json \
  --out dist/counter-debug.apk

mise run apk:counter
```

Default ABI is **arm64-v8a** only (pure Go / `CGO_ENABLED=0`; other ABIs need
an NDK). Full APK also needs **JDK 17+**, **Android SDK** (`ANDROID_HOME`), and
**Gradle 8.9+** on `PATH`. Without the SDK:

```bash
go run ./cmd/eletrocromo android build --config examples/counter/eletrocromo.json --go-only --workdir dist/android-counter
```

`android create` still exists if you only want the Gradle tree. Runtime: the
service sets `ELETROCROMO_NO_UI=1` and loads the `ELETROCROMO_READY` URL in
WebView. Packaging lives in `internal/apkgen/` + `cmd/eletrocromo` (not in the
core library import path for apps).
