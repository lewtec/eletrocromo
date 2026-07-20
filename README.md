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

Template counter dogfood (Helium-first launch):

```bash
mise run example:counter
# or: go run ./examples/counter
```

Ctrl+C in the terminal stops the process. `+` / `−` / reset hit the local server via form POST.
