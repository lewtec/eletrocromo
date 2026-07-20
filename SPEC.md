# SPEC: eletrocromo

CGo-less Go library: run a local web UI on loopback (auth always on) and surface it in a dumb window. Not Electron. Not Wails. Not a native widget toolkit.

Status: approved (grill session, 2026-07-20). This document is the expectation contract. Implementation may lag the v1 checklist.

## One-liner

A pure-Go process owns the HTTP app; eletrocromo binds loopback, gates access, opens a **Helium** (Chromium-based) `--app` window, and on Linux can keep the process alive and reopen the UI from the system tray.

## Goals

- Ship desktop apps as **CGo-less Go binaries** whose UI is a normal webapp talking to a **server on the same device**.
- Let the app focus on an `http.Handler` or `*http.Server`; the library handles bind, auth handshake, window launch, and process lifetime modes.
- On **Linux (v1 bar)**: window-owned lifetime by default; optional background mode with **tray Open/Quit** so the user never retypes a token URL.
- Stay thin: the window is **Helium** (or another Chromium-like if configured/discovered secondarily), not a bundled runtime and not a native toolkit. Deep native integration is out of scope by design.
- Prefer **[Helium](https://helium.computer/)** as the desktop shell: privacy-oriented Chromium fork that still supports `--app` app windows.

## Non-goals

Not this library’s job (now or as “quiet scope creep”):

| Out | Why |
|-----|-----|
| Native menus, custom window chrome beyond `--app` | Dumb browser surface |
| File/folder dialogs as library APIs | App/HTTP/browser concerns |
| JS ↔ Go IPC bridge beyond ordinary HTTP/WebSocket | Would become Wails |
| Bundled / downloaded browser runtime inside eletrocromo | Library discovers an install; packaging is separate |
| Auto-updater, installers, full packaging product | Distribution is separate |
| Frontend framework or SPA opinions | App serves whatever it wants |
| Multi-window platform APIs | Out of scope |
| LAN / non-loopback bind as a happy path | Same-device only |
| Optional “no auth” mode | Auth always on |
| CGo in core, tray, or required deps | Non-negotiable |
| Win/mac tray/lifecycle parity in v1 | Linux-first |
| APK generation inside the core module (v1) | Later sibling / packaging story |
| Firefox (or other non-Chromium) as app window | No `--app`-style borderless/PWA window mode |
| Emulating app windows via Firefox extensions / profiles | Out of scope |
| System default browser fallback | Full browser chrome; may be Firefox; not an app window — **removed** |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  Go process (the app)                                       │
│                                                             │
│  App HTTP logic  ──►  eletrocromo auth wrapper              │
│                       (token query → cookie, fail-closed)   │
│                              │                              │
│                              ▼                              │
│                     loopback listener                       │
│                     (library owns bind)                     │
│                              │                              │
│              ┌───────────────┴───────────────┐              │
│              ▼                               ▼              │
│     Helium --app                      tray (Linux)          │
│     (primary shell)                   Open / Quit           │
│     [optional: other Chromium-like]          │              │
│              │                               │              │
│              └──────── reopen / focus ───────┘              │
│                                                             │
│  No host? → hard error (no system-browser fallback)         │
│  Lifetime: window-owned (default) | background (--flag)     │
│  Single-instance: lockfile / PID + resume as needed         │
└─────────────────────────────────────────────────────────────┘
```

**Invariant:** Go owns server and business logic. The shell only presents the UI and manages process lifetime. The webapp communicates with the server on the **same device** only.

### Later: APK (north star, not v1)

APK is a **wrapper**, not a second architecture:

- WebView for UI
- A service that **runs the same Go binary** (local server on device loopback)
- Same mental model: webapp ↔ on-device server

Implementation may live as a sibling tool/module. It must not force CGo or Android SDK into the core desktop library.

## Public contract

### What the app supplies

Two entry shapes (same ownership model):

1. **Handler** — convenience constructor around `http.Handler`.
2. **Server** — pass `*http.Server` for handler, timeouts, and related config the app cares about.

The app does **not** own bind as the source of truth. If `Server.Addr` is set, the library either ignores it or rejects it; the library assigns the loopback address/port.

### What the library owns

| Concern | Rule |
|---------|------|
| Bind | Loopback only (`127.0.0.1` / `localhost`, and `::1` if used). Never `0.0.0.0` / LAN as default or silent behavior. |
| Auth | Always on. Mint token if unset; fail-closed when missing/invalid. |
| Auth UX | Initial URL may carry `?token=…`; set HttpOnly cookie; subsequent requests use cookie. |
| UI open | **App window** via Helium (primary) or secondary Chromium-like + `--app`. **No** system default browser. Missing host ⇒ error. |
| Scheme | Only `http` / `https` for launch URLs. |
| Lifecycle | See modes below. |
| Background work | Existing task/`WaitGroup` style coordination remains valid for app-scheduled work. |

### Desktop window surface: Helium-first, Chromium `--app` only

On desktop, the UI is an app-mode window launched with `--app=<url>`. The **product focus** is **[Helium](https://helium.computer/)** (Chromium-based, ungoogled-oriented; supports the same app-window flags as Chromium).

| Path | Priority | Behavior |
|------|----------|----------|
| **Helium** | **Primary** | Discover and prefer Helium binaries/paths first; launch with `--app` |
| Other Chromium-likes | Secondary (optional) | Chrome, Chromium, Edge, Brave, … only if Helium is absent; same `--app` |
| System default browser (`xdg-open`, etc.) | **Forbidden** | Do not open the token URL in a random full browser |
| Firefox / Gecko | **Forbidden** as shell | No `--app` equivalent; never on the discovery list |

**Normative constraints:**

- **No system-browser fallback.** If no suitable host is found, `Run` / launch fails with a clear error (e.g. “install Helium” / `ErrNoChromium` evolved to Helium-oriented messaging). Do not silently degrade to Firefox or a full tabbed browser.
- **Helium is the dogfood and recommended install** for Linux v1. Docs and example assume Helium is available.
- Secondary Chromium-likes may remain in the discovery list for convenience, but quality and docs are judged on Helium.
- **No Firefox app-window support** and no extension/profile hacks.
- The library does **not** download or vendor Helium; the environment provides it (distro package, [imputnet/helium](https://github.com/imputnet/helium), workspaced shim, etc.).

This is a product boundary: one app-window stack (Chromium `--app`), centered on Helium, fail closed when missing.

### Auth details (normative intent)

- Per-process token (e.g. UUID); not a stable long-lived secret across restarts unless the app sets `AuthToken` deliberately.
- Constant-time compare for token checks.
- Empty `AuthToken` must not accept unauthenticated traffic (fail closed).
- Reopen/tray/resume must **not** depend on the user pasting the token URL. Resume is in-process or via single-instance protocol to the existing PID.

### Browser discovery (target)

1. **Helium first** — known binary names and install paths (e.g. `helium` on `PATH`, common Linux install locations; extend as installs are observed).
2. **Then** other Chromium-likes (existing list: Edge, Chrome, Chromium, Brave, Vivaldi, Opera, …), first hit wins among those.
3. **Never** `firefox` / Gecko, and **never** system URL openers as a substitute for `--app`.
4. No download, no pin of a browser binary into this module.

Launch always uses the Chromium app-window flag shape: ` <browser> --app=<url> `.

## Lifetime modes (Linux v1)

| Mode | How selected | Process exits when |
|------|----------------|--------------------|
| **Window-owned** | **Default** | UI window is gone (window dies ⇒ process dies) |
| **Background** | Explicit **flag** (CLI and/or API) | Context cancel / signal / tray **Quit** (or equivalent) |

### Background + tray

- Closing the window does **not** kill the process.
- System tray provides at least:
  - **Open** — show/relaunch the UI without typing address or token
  - **Quit** — cancel context and exit cleanly
- Tray is the primary human reopen affordance. Lockfile + PID (and a resume signal/protocol) support single-instance and “second launch resumes” if needed; tray remains the product-facing story.

### CGo-less tray

Tray and lifecycle must work **without CGo**. If a approach requires CGo, it is rejected; use pure Go, subprocess helpers, D-Bus, lockfiles, or browser-process handles as available on Linux. Weaker tray on non-Linux is acceptable until those platforms are in scope.

## Platform support

| Platform | v1 |
|----------|----|
| **Linux** | Full vision: bind, auth, launch, both lifetime modes, tray Open/Quit, docs, dogfood example |
| **Windows / macOS** | Best-effort later; launching UI + server may work; no tray/lifecycle parity promise |
| **Android APK** | After desktop Linux v1; packaging wrapper only |

## v1 done checklist (Linux)

v1 is **complete** when all of the following hold:

1. **Entry:** constructor/API for `http.Handler` and for `*http.Server`; library owns loopback bind.
2. **Auth:** always on; documented handshake; fail-closed.
3. **Launch:** Helium-first discovery; `--app` only; **hard error** if no Chromium-like host (no system-browser fallback).
4. **Default lifetime:** window close ⇒ process exit.
5. **Background mode:** explicit flag; process outlives window.
6. **Tray:** Open and Quit work without address-bar token ritual.
7. **Single-instance / lockfile:** second start or tray path can resume UI against the running process as designed (no second competing server as the happy path).
8. **Docs:** modes, auth, loopback-only, non-goals, how to run the example.
9. **Dogfood example:** counter UI with Go `html/template` (see below).
10. **CGo-less:** `go build` with CGO disabled succeeds for the library and example on Linux.

**Not required for v1:** APK tool, Win/mac tray, native dialogs, bundled browser.

## Dogfood example

**Counter** app using **Go `html/template`** (server-rendered; no SPA framework requirement).

| Behavior | Requirement |
|----------|-------------|
| Default | Dies when window closes |
| Flag | Background + tray (Open / Quit) |
| Role | Acceptance binary for Linux v1 |

Success test:

1. Run counter → increment works → close window → process exits.
2. Run with background/tray flag → close window → process still up → tray **Open** restores UI → tray **Quit** exits cleanly.
3. Never required: manually open a browser and paste `http://127.0.0.1:…/?token=…`.

## API shape (intent; names may evolve)

Illustrative, not frozen identifiers:

```text
// Handler path
app := eletrocromo.New(handler, opts...)

// Server path
app := eletrocromo.NewServer(server, opts...)

app.Run(ctx)  // blocks until shutdown
```

Options / flags (conceptual):

- Background / tray mode (off by default)
- Pre-set auth token (optional; otherwise mint)
- Context for cancellation (signals wired by the app or helpers)

Exact API is an implementation detail as long as the contract above holds.

## Security summary

| Control | Policy |
|---------|--------|
| Network exposure | Loopback only |
| Auth | Always on; token gate |
| Token in URL | Bootstrap only; prefer cookie afterward; do not treat URL as long-term bookmark |
| Local attackers | Other local processes may still be a threat; token raises the bar vs open loopback; not a multi-user OS security boundary |
| URL schemes for launch | `http` / `https` only |

## Implementation notes (today → target)

| Area | Today (approx.) | Target |
|------|-----------------|--------|
| Entry | `App{Handler, Context, …}.Run()` | Handler **or** `*http.Server` constructors; bind always library-owned |
| Server | `httptest` | Keep ephemeral loopback; do not hand bind to the app |
| Auth | Token + cookie; fail-closed | Keep always-on; no opt-out in v1 |
| Launch | `GetChromium` + `--app`; system open fallback | **Helium first**; secondary Chromium-likes OK; **remove** system-browser fallback; fail if none |
| Lifetime | Context cancel only; browser `Start` fire-and-forget | Default window-owned; flag background + tray |
| Tray / lockfile | Absent | Linux v1 requirement |
| Example | `examples/basic` | Add/replace with template counter + mode flag |
| README | Short architecture blurb | Align with this SPEC |

## Success criteria

- A developer writes only HTTP/template logic and gets a usable Linux “desktop” window **under Helium**.
- Missing Helium/Chromium-like host fails loudly; never opens the default browser.
- CGO=0 builds and runs the dogfood counter on Linux.
- Background mode is usable daily without ever typing the loopback token URL.
- The project description never requires “we’ll add native menus next” to feel complete.

## Open implementation choices (not product questions)

Resolved by engineering when building, not by re-litigating product meaning:

- Exact flag names (`--tray`, `--background`, …)
- Lockfile path and resume protocol (signal, local socket, etc.)
- How window-death is detected under CGo-less constraints (browser process wait, WM heuristics, …)
- Whether tray is a build-tagged Linux file set vs always compiled stubs
- Precise constructor names and option functional options vs struct fields
- Exact Helium path list per distro / workspaced layout; whether secondary Chromium-likes stay enabled by default or opt-in

---

*Aligned in grill session; amended for Helium-first shell and no system-browser fallback. Do not expand scope into non-goals without a new explicit decision.*
