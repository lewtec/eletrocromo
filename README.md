# eletrocromo

A simpler approach to desktop apps without Electron or Wails: pure Go HTTP
handler + local loopback server + Chromium-style **app window** (Helium first).

See [SPEC.md](SPEC.md) for the product contract.

## Architecture

On start, eletrocromo wraps your `http.Handler` with always-on token auth, binds
loopback, and opens the UI with `--app`:

1. Prefer **Helium** on `PATH`
2. Else other already-installed Chromium-likes
3. Else ensure Helium via **workspaced** (`tool which helium-browser helium`),
   bootstrapping a pinned workspaced binary into the user cache if needed
4. If still missing → **error** (never the system default browser)

```go
app := eletrocromo.App{
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
