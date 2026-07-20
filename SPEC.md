# SPEC: eletrocromo

CGo-less Go library: run a local web UI on loopback (auth always on) and surface it in a dumb window. Not Electron. Not Wails. Not a native widget toolkit.

Status: approved (grill session, 2026-07-20). This document is the expectation contract. Implementation may lag the v1 checklist.

## One-liner

A pure-Go process owns the HTTP app; eletrocromo binds loopback, gates access, opens a **Helium** (Chromium-based) `--app` window, and on Linux can keep the process alive and reopen the UI from the system tray.

## Goals

- Ship desktop apps as **CGo-less Go binaries** whose UI is a normal webapp talking to a **server on the same device**.
- Let the app focus on an `http.Handler` or `*http.Server`; the library handles bind, auth handshake, window launch, and process lifetime modes.
- On **Linux (v1 bar)**: window-owned lifetime by default; optional background mode with **tray Open/Quit** so the user never retypes a token URL.
- Stay thin: the window is **Helium** (or another already-installed Chromium-like as a secondary discover path), not a native toolkit.
- Prefer **[Helium](https://helium.computer/)** as the desktop shell: privacy-oriented Chromium fork that still supports `--app` app windows.
- **Ensure** Helium when missing via **[workspaced](https://github.com/lucasew/workspaced)** (registry tool `helium-browser`), including bootstrapping the workspaced binary if needed — without vendoring browser blobs in this module.

## Non-goals

Not this library’s job (now or as “quiet scope creep”):

| Out | Why |
|-----|-----|
| Native menus, custom window chrome beyond `--app` | Dumb browser surface |
| File/folder dialogs as library APIs | App/HTTP/browser concerns |
| JS ↔ Go IPC bridge beyond ordinary HTTP/WebSocket | Would become Wails |
| Vendoring Helium/Chromium **inside the eletrocromo module** | Ensure uses workspaced’s tool store + registry, not a copy of the browser in-tree |
| Importing workspaced as a fat Go library | Ensure is **subprocess** to the workspaced binary (stable boundary; `internal/tool` is not public) |
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
| Depending on home `lazy_tools` / personal cue aliases | Registry name only so any workspaced install works |

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│  Go process (the app)                                        │
│                                                              │
│  App HTTP logic ──► eletrocromo auth wrapper                 │
│                     (token query → cookie, fail-closed)      │
│                            │                                 │
│                            ▼                                 │
│                   loopback listener                          │
│                   (library owns bind)                        │
│                            │                                 │
│                            ▼                                 │
│                   Resolve browser host                       │
│                   (see Ensure pipeline)                      │
│                            │                                 │
│              ┌─────────────┴─────────────┐                   │
│              ▼                           ▼                   │
│     helium --app <url>            tray (Linux)               │
│     (primary shell)               Open / Quit                │
│              │                           │                   │
│              └──────── reopen / focus ───┘                   │
│                                                              │
│  No host after ensure? → hard error (never system browser)   │
│  Lifetime: window-owned (default) | background (--flag)      │
│  Single-instance: lockfile / PID + resume as needed          │
└──────────────────────────────────────────────────────────────┘
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
| UI open | **App window** via Helium (or secondary already-installed Chromium-like) + `--app`. **No** system default browser. |
| Host resolve | Discover local → ensure Helium via workspaced → hard error if still missing. |
| Scheme | Only `http` / `https` for launch URLs. |
| Lifecycle | See modes below. |
| Background work | Existing task/`WaitGroup` style coordination remains valid for app-scheduled work. |

### Desktop window surface: Helium-first, Chromium `--app` only

On desktop, the UI is an app-mode window launched with `--app=<url>`. The **product focus** is **[Helium](https://helium.computer/)** (Chromium-based; supports Chromium app-window flags).

| Path | Priority | Behavior |
|------|----------|----------|
| **Helium** (local) | 1 | `helium` on `PATH` / known install paths |
| Other Chromium-likes | 2 | Already-installed Chrome/Chromium/Edge/Brave/… only; **no** ensure for these |
| **Helium via workspaced** | 3 | Ensure registry tool `helium-browser`, binary `helium` (see pipeline) |
| System default browser | **Forbidden** | Never `xdg-open` / OS URL opener as substitute |
| Firefox / Gecko | **Forbidden** | Never on discovery list |

**Normative constraints:**

- **No system-browser fallback.** After the resolve/ensure pipeline fails, `Run` / launch returns a clear error. Never open a full tabbed browser with the token URL.
- **Helium is dogfood and quality bar** for Linux v1.
- **No Firefox** app-window path or extension/profile hacks.
- Browser bits live in **workspaced’s tool store** (or a pre-existing install), not inside the eletrocromo module tree.

### Host resolve / ensure pipeline

Ordered steps for obtaining a Chromium-like binary that will run with `--app`:

```text
1. Local Helium
   LookPath("helium") and known Helium install paths.
   If found → use it.

2. Local secondary Chromium-likes (optional convenience)
   Existing discovery list (chrome, chromium, edge, brave, …).
   If found → use it.
   Do not download these.

3. Ensure Helium via workspaced
   a. Locate workspaced binary:
      - LookPath("workspaced"), else
      - cached bootstrap under XDG cache (e.g. ~/.cache/eletrocromo/workspaced/<pinned-version>/workspaced), else
      - download pinned workspaced release asset for GOOS/GOARCH into that cache
        (GitHub Releases for lucasew/workspaced; same idea as workspaced’s setup script).
      Verify before exec (at least checksum / release digest policy — no curl|bash).
   b. Resolve Helium path (installs if missing):
        workspaced tool which helium-browser helium
      Registry tool name: **helium-browser** (not a home lazy alias, not raw github: OS fork).
      Binary name: **helium**.
   c. Use the printed absolute path.

4. Fail closed
   Return an error that explains: need Helium, or network/workspaced ensure failed.
   Never fall back to the system default browser.
```

**Launch** (always, once a binary path is chosen):

```text
<bin> --app <url>
```

Use **`tool which`** (not only `tool with`) so eletrocromo **owns** the browser process (`Start`/`Wait`) for window-owned lifetime and relaunch. `tool with helium-browser -- helium --app …` is acceptable only as an equivalent if process ownership requirements are met; prefer which → exec.

**Defaults:**

| Policy | Decision |
|--------|----------|
| Ensure when Helium/secondary missing | **On** for normal desktop `Run` (product magic on first launch) |
| Offline / ensure failure | Hard error; no degraded browser |
| Prefer local | Steps 1–2 never hit the network |
| Workspaced integration | **Subprocess CLI only** — do not import `github.com/lucasew/workspaced` as a library dependency of eletrocromo |
| Registry vs GitHub ref | Always **`helium-browser`** (catalog/registry); multi-OS artifacts are workspaced’s job |
| Home `lazy_tools.helium_browser` | Out of library path; users may still have personal shims, but ensure must not require them |
| Version pins | **Pin workspaced** release used for bootstrap in code/config. Helium version follows workspaced catalog resolution unless a pin is added later |
| Cache | Bootstrap binary under eletrocromo XDG cache; Helium install under workspaced’s normal tool store |
| Tests / CI | May disable ensure (option/env) so unit tests never download browsers; discovery-only tests remain pure |

**Security / trust:**

- Bootstrapping workspaced is a **trust decision**: pin version + verify artifact; document the pin.
- First run may download **workspaced** and **Helium** (large); log progress; reuse cache afterward.
- Token URL must not be passed to untrusted openers; only the resolved `--app` host.

### Auth details (normative intent)

- Per-process token (e.g. UUID); not a stable long-lived secret across restarts unless the app sets `AuthToken` deliberately.
- Constant-time compare for token checks.
- Empty `AuthToken` must not accept unauthenticated traffic (fail closed).
- Reopen/tray/resume must **not** depend on the user pasting the token URL. Resume is in-process or via single-instance protocol to the existing PID.

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
3. **Launch:** Helium-first discovery; optional secondary Chromium discover; workspaced ensure of registry `helium-browser` (bootstrap workspaced if needed); `--app` only; **hard error** if still no host (no system-browser fallback).
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
- Disable ensure (for tests/CI): discovery-only, no network
- Optional override path to workspaced binary / pin version

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
| Launch | `GetChromium` + `--app`; system open fallback | Helium → secondary discover → `workspaced tool which helium-browser helium` (bootstrap workspaced) → `--app`; **no** system browser |
| Lifetime | Context cancel only; browser `Start` fire-and-forget | Default window-owned; flag background + tray |
| Tray / lockfile | Absent | Linux v1 requirement |
| Example | `examples/basic` | Add/replace with template counter + mode flag |
| README | Short architecture blurb | Align with this SPEC |

## Success criteria

- A developer writes only HTTP/template logic and gets a usable Linux “desktop” window **under Helium**.
- Clean machine with network: first `Run` can bootstrap workspaced + ensure `helium-browser`, then open `--app` (no system browser).
- Offline with no local host: fails loudly; never opens the default browser.
- CGO=0 builds and runs the dogfood counter on Linux.
- Background mode is usable daily without ever typing the loopback token URL.
- The project description never requires “we’ll add native menus next” to feel complete.

## Implementation order (toward full SPEC)

1. **Launch contract:** Helium-first + secondary discover; remove system-browser fallback; hard error.
2. **Ensure via workspaced on PATH:** `tool which helium-browser helium` → `--app`.
3. **Bootstrap workspaced binary** (pinned + verified cache) when missing.
4. Window-owned lifetime; background + tray + lockfile.
5. Handler / `*http.Server` constructors; template counter dogfood; README ↔ SPEC.

## Open implementation choices (not product questions)

Resolved by engineering when building, not by re-litigating product meaning:

- Exact flag names (`--tray`, `--background`, `ELETROCROMO_NO_ENSURE`, …)
- Lockfile path and resume protocol (signal, local socket, etc.)
- How window-death is detected under CGo-less constraints (browser process wait, WM heuristics, …)
- Whether tray is a build-tagged Linux file set vs always compiled stubs
- Precise constructor names and option functional options vs struct fields
- Exact Helium path list per distro; which secondary Chromium names stay in the list
- Workspaced release pin value, checksum source, and cache layout under XDG
- Whether Helium catalog version is left floating to workspaced or pinned later

---

*Aligned in grill session; amended for Helium-first shell, no system-browser fallback, and workspaced registry ensure (`helium-browser`). Do not expand scope into non-goals without a new explicit decision.*
