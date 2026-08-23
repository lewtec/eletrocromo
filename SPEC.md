# SPEC: eletrocromo

CGo-less Go library: run a local web UI on loopback (auth always on) and surface it in a dumb window. Not Electron. Not Wails. Not a native widget toolkit.

Status: approved (grill sessions 2026-07-20 desktop shell; 2026-07-23 app icons / packaging CLI; 2026-08-22 macOS `.app` packaging). This document is the expectation contract. Implementation may lag checklists.

## One-liner

A pure-Go process owns the HTTP app; eletrocromo binds loopback, gates access, opens a **Helium** (Chromium-based) `--app` window, and on Linux can keep the process alive and reopen the UI from the system tray. A separate **packaging CLI** (`cmd/eletrocromo`) generates multi-platform **app icons**, scaffolds Android hosts JIT into an APK, and scaffolds macOS hosts JIT into an unsigned Debug `.app`.

## Goals

- Ship desktop apps as **CGo-less Go binaries** whose UI is a normal webapp talking to a **server on the same device**.
- Let the app focus on an `http.Handler` or `*http.Server`; the library handles bind, auth handshake, window launch, and process lifetime modes.
- On **Linux (v1 bar)**: window-owned lifetime by default; optional background mode with **tray Open/Quit** so the user never retypes a token URL.
- Stay thin: on desktop `Run()`, the window is **Helium only**, not a native toolkit and not “whatever Chromium is on PATH.” Packaged Android/macOS hosts use the OS WebView.
- Prefer **[Helium](https://helium.computer/)** as the desktop shell: privacy-oriented Chromium fork that still supports `--app` app windows.
- **Ensure** Helium when missing via **[workspaced](https://github.com/lucasew/workspaced)** (registry tool `helium-browser`), including bootstrapping the workspaced binary if needed — without vendoring browser blobs in this module.
- **Packaging CLI** (not the runtime library): generate a full **icon matrix** from one PNG/SVG (or a shipped default mark) and build **Android APKs** and **macOS `.app` bundles** via JIT host scaffold + attached Go binary. Leave DMG, notarization, PE embedding, and GoReleaser installer wiring to later tracks or the app author.

## Non-goals

Not this library’s job (now or as “quiet scope creep”):

| Out | Why |
|-----|-----|
| Native menus, custom window chrome beyond `--app` | Dumb browser surface |
| File/folder dialogs as library APIs | App/HTTP/browser concerns |
| JS ↔ Go IPC bridge beyond ordinary HTTP/WebSocket | Would become Wails |
| Vendoring Helium/Chromium **inside the eletrocromo module** | Ensure uses workspaced’s tool store + registry, not a copy of the browser in-tree |
| Importing workspaced into the **runtime library** used at `App.Run()` | Helium ensure stays **subprocess** to the workspaced binary (stable boundary). Packaging CLI may import workspaced packages (e.g. `taskgroup`) |
| Auto-updater, full installer product, mandatory PE embedding | Distribution is separate. JIT `build macos` is the Mac packaging product; DMG/notarization are not v1 |
| Frontend framework or SPA opinions | App serves whatever it wants |
| Multi-window platform APIs | Out of scope |
| LAN / non-loopback bind as a happy path | Same-device only |
| Optional “no auth” mode | Auth always on |
| CGo in core, tray, or required **runtime** deps | Non-negotiable for library/apps at run time |
| Win/mac tray/lifecycle parity in v1 | Linux-first |
| Forcing Android SDK / Xcode / CGo into the **importable desktop library** | APK and `.app` tooling live in `cmd/eletrocromo` + `internal/…` (and icon code), not in `Run()` |
| Vendoring a prebuilt Mach-O host stub | Ephemeral Xcode compile on a Mac; no stub releases in this module |
| Firefox (or other non-Chromium) as app window | No `--app`-style borderless/PWA window mode |
| Emulating app windows via Firefox extensions / profiles | Out of scope |
| System default browser as the **desktop `Run()` window** | Full browser chrome; may be Firefox; not an app window — **removed**. Packaged Mac WKWebView may open **off-loopback** http(s) in the default browser; that is not the app window |
| Depending on home `lazy_tools` / personal cue aliases | Registry name only so any workspaced install works |
| Runtime icon conversion inside `App.Run()` | Icons are a packaging concern |
| GoReleaser plugin ABI | Hooks + documented YAML only |

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

### Packaging CLI (sibling surface, same module)

Runtime library and packaging CLI share a repo but **different dependency rules**:

```
┌─────────────────────────────────────────────────────────────┐
│  cmd/eletrocromo (packaging)                                │
│                                                             │
│  build icons  ──► dist/icons/** + manifest.json             │
│       ▲                                                     │
│       │ missing / --refresh-icons                           │
│  build android ──► JIT Gradle host + jniLibs + APK          │
│  build macos   ──► JIT XcodeGen host + darwin Go + .app     │
│  build ios     ──► JIT XcodeGen host + ios c-archive + .app │
│       │            (icns / mipmaps from icon tree)          │
│       └── taskgroup orchestration (workspaced import OK)    │
│                                                             │
│  Config: eletrocromo.json (package_id, icon, …) + flags     │
└─────────────────────────────────────────────────────────────┘
```

### APK (wrapper architecture)

APK is a **wrapper**, not a second app architecture:

- WebView for UI
- A service that **runs the same Go app** (local server on device loopback)
- Same mental model: webapp ↔ on-device server

Scaffold is **just-in-time** during `build android` (no happy-path “create host project and commit it”). Must not force CGo or Android SDK into the **importable desktop library**.

### macOS `.app` (wrapper architecture)

`.app` is a **wrapper**, not a second app architecture and not a Helium bundle:

- WKWebView for UI
- The host **runs the same Go app** (`ELETROCROMO_NO_UI=1`) on loopback
- Same handshake as Android: `ELETROCROMO_READY` on stdout and/or `ELETROCROMO_READY_FILE`
- Same mental model: webapp ↔ on-device server

Desktop `App.Run()` from a terminal still uses **Helium only**. The packaged `.app` does not launch Helium.

Scaffold is **just-in-time** during `build macos` (ephemeral XcodeGen + Swift host). No happy-path “create host project and commit it”. Must not force Xcode, CGo, or a vendored Mach-O into the **importable desktop library**.

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
| UI open | **App window** via Helium only + `--app` + per-app `--user-data-dir`. **No** Chrome/Edge/system browser. |
| Host resolve | Local Helium → ensure Helium via workspaced → hard error if still missing. |
| App identity | Required reverse-domain `App.ID` (e.g. `br.tec.lew.myapp`) for profile isolation and future APK package name. |
| Launch failure | If Helium exits during startup grace, `Run` returns an error (not ignored). Later Helium exit cancels the app. |
| Scheme | Only `http` / `https` for launch URLs. |
| Lifecycle | See modes below. |
| Background work | Existing task/`WaitGroup` style coordination remains valid for app-scheduled work. |

### Desktop window surface: Helium only

On desktop, the UI is an app-mode window launched with `--app=<url>` on **[Helium](https://helium.computer/)** only (Chromium-based engine; we do not discover other browsers).

| Path | Priority | Behavior |
|------|----------|----------|
| **Helium** (local) | 1 | `helium` on `PATH` |
| **Helium via workspaced** | 2 | Ensure registry tool `helium-browser`, binary `helium` |
| Chrome / Chromium / Edge / Brave / … | **Forbidden** | Not on the discovery list |
| System default browser | **Forbidden** | Never `xdg-open` / OS URL opener |
| Firefox / Gecko | **Forbidden** | Never |

**Normative constraints:**

- **Helium-only.** No secondary Chromium-like discover path.
- **No system-browser fallback.** After the resolve/ensure pipeline fails, `Run` / launch returns a clear error.
- Browser bits live in **workspaced’s tool store** (or a pre-existing Helium install), not inside the eletrocromo module tree.

### Host resolve / ensure pipeline

```text
1. Local Helium
   LookPath("helium"). If found → use it.

2. Ensure Helium via workspaced
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

3. Fail closed
   Return an error that explains: need Helium, or network/workspaced ensure failed.
   Never fall back to Chrome or the system default browser.
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
| Prefer local | Step 1 never hits the network |
| Workspaced integration (runtime library) | **Subprocess CLI only** for Helium ensure — do not import `github.com/lucasew/workspaced` into the **importable library** path used by apps at `Run()` |
| Workspaced integration (packaging CLI) | **`cmd/eletrocromo` may import** workspaced packages (e.g. `taskgroup`) and still **subprocess** `workspaced tool which` for icon/raster tools |
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

| Platform | Desktop runtime (v1 bar) | Packaging / icons |
|----------|--------------------------|-------------------|
| **Linux** | Full vision: bind, auth, launch, both lifetime modes, tray Open/Quit, docs, dogfood example | Icon matrix; desktop package recipes documented (nFPM/Snap/etc. are user-wired) |
| **Windows** | Best-effort later; launching UI + server may work; no tray/lifecycle parity promise | Icon matrix includes `.ico`; PE embedding left to user/GR |
| **macOS** | Best-effort Helium `Run()`; no tray/lifecycle parity promise | Icon matrix includes `.icns`; **`build macos`** JIT unsigned Debug `.app` (WKWebView host) |
| **Android** | N/A (not Helium desktop) | APK via packaging CLI; launcher mipmaps from icon pipeline |
| **iOS / iPadOS** | N/A (not Helium desktop) | Scaffold: `build ios` JIT Debug `.app` (WKWebView + in-process c-archive). Not grilled. |

## v1 done checklist (Linux desktop runtime)

v1 **desktop library** is **complete** when all of the following hold:

1. **Entry:** constructor/API for `http.Handler` and for `*http.Server`; library owns loopback bind.
2. **Auth:** always on; documented handshake; fail-closed.
3. **Launch:** Helium only (PATH or workspaced ensure of `helium-browser`); `--app` only; **hard error** if still no host (no other browsers, no system fallback).
4. **Default lifetime:** window close ⇒ process exit.
5. **Background mode:** explicit flag; process outlives window.
6. **Tray:** Open and Quit work without address-bar token ritual.
7. **Single-instance / lockfile:** second start or tray path can resume UI against the running process as designed (no second competing server as the happy path).
8. **Docs:** modes, auth, loopback-only, non-goals, how to run the example.
9. **Dogfood example:** counter UI with Go `html/template` (see below).
10. **CGo-less:** `go build` with CGO disabled succeeds for the library and example on Linux.

**Not required for desktop v1:** Win/mac tray, native dialogs, bundled browser, icon/APK/`.app` packaging (those are the packaging track below).

## Packaging track checklist (icons + Android + macOS)

Separate from desktop tray/lifetime. Complete when:

1. **Default assets:** vendored square **mark** + **lockup** in-repo; generator never depends on ephemeral paths.
2. **`eletrocromo build`:** bare invocation errors and lists targets.
3. **`build icons`:** one master (config/`--icon`/default) → full `dist/icons` tree + `manifest.json`; pad+center; `--output`; `--refresh-icons`.
4. **`build android`:** JIT scaffold + multiarch Go + APK; runs icons when outputs missing (or `--refresh-icons`); mipmaps applied in workdir.
5. **`build macos`:** JIT XcodeGen + Swift host + host-arch darwin Go + unsigned Debug `.app`; runs icons when `macos/icon.icns` is missing (or `--refresh-icons`); `--go-only` skips `xcodebuild`.
6. **Config:** `icon` field on `eletrocromo.json`; flags override; **no Mac-only json keys** in v1.
7. **Conversion:** Go libs preferred; workspaced ensure for missing tools; fail closed.
8. **Orchestration:** workspaced `taskgroup` (or equivalent imported API) for named deps.
9. **Docs:** GoReleaser OSS hooks + Pro `app_bundles.icon` / nFPM/Snap recipes; users wire embedding. `build macos` is the happy-path `.app`; DMG is not.
10. **Deprecate or demote** happy-path `android create` (JIT build is the product).

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

## App icons and packaging CLI (normative; grill 2026-07-23)

Product decision for **apps built with eletrocromo**, not primarily for branding the `eletrocromo` CLI **release** binary. **Generator + docs**; users wire GoReleaser / installers themselves.

### Product goals

- Ship a **default logo** (eletrocromo brand): square **mark** for the icon pipeline; full **lockup** (mark + wordmark) for marketing/docs only.
- App authors supply **one** master **PNG or SVG** (`icon` in `eletrocromo.json` or `--icon`); tooling **rasterizes and generates** all platform artifacts.
- Cover **Windows, macOS, Linux, Android, and web** favicon surfaces in the output tree.
- **Pad + center** non-square masters (letterbox; transparency when the format allows).

### Feature non-goals

- Embedding icons at **`App.Run()`** / runtime conversion.
- Auto-wiring PE resources or nFPM as a mandatory pipeline (document recipes only). JIT `build macos` **is** the `.app` product; GoReleaser Pro `app_bundles` remains optional docs.
- Content-hash auto-invalidation of icons (v1 of this track).
- Neutral non-brand default (default **is** the eletrocromo mark).
- True GoReleaser “plugin” ABI — use **hooks + documented YAML**.

### Source and defaults

| Input | Rule |
|-------|------|
| Config | `icon` in `eletrocromo.json` (path relative to config dir) |
| Flag | `--icon` overrides config (full word; no short `-i`) |
| Missing both | Use **embedded default mark** (vendored asset; never a live `/tmp/…` path) |
| Lockup | Separate vendored asset for README/site; **not** used for 16×16 / mipmaps |

### Output tree

Default root: **`dist/icons`** (override with **`--output`**).

```text
dist/icons/
  source/           # normalized master (always master.png; SVG copy if applicable)
  windows/          # e.g. icon.ico
  macos/            # e.g. icon.icns
  linux/            # multi-size PNGs (+ SVG if master was SVG)
  android/          # mipmap-* launcher trees
  web/              # favicon.ico, apple-touch-icon, etc.
  manifest.json     # index of paths/sizes for docs and tooling
```

### Conversion stack

- Prefer **in-process Go libraries** when adequate.
- Otherwise **ensure tools via workspaced** (same mental model as Helium ensure).
- Fail closed with a clear “install/ensure X or pass a PNG” style error when a required converter is missing.

### CLI shape (`cmd/eletrocromo`)

```text
eletrocromo build              → error; list targets (icons, android, macos, …)
eletrocromo build icons        → write the icon tree
eletrocromo build android      → JIT scaffold + cross-compile Go + APK;
                                 runs icons first if outputs missing
eletrocromo build macos        → JIT XcodeGen + Swift host + darwin Go + .app;
                                 runs icons first if macos/icon.icns is missing
eletrocromo build ios          → JIT XcodeGen + UIKit host + GOOS=ios c-archive + .app;
                                 scaffold (not grilled); READY file only; Mac + Xcode
```

| Flag / behavior | Rule |
|-----------------|------|
| `--config` | Existing JSON load pattern (`eletrocromo.json`) |
| `--icon` | Master image path |
| `--output` | Icon tree root (default `dist/icons`) |
| `--refresh-icons` | Force full icon regen; without it, generate only when **expected outputs are missing** |
| Scaffold | **Just-in-time** for android, macos, and ios (no happy-path `create` / commit host project) |
| Orchestration | **workspaced `taskgroup`** (named tasks, deps: icons → android/macos/ios steps) |

Migrate existing `eletrocromo android build` / `android create` to the `build …` surface; `create` is not the product path and may be removed after migration.

### GoReleaser integration

- **Not a plugin.** App authors run `eletrocromo build icons` from **`before.hooks`** (or equivalent).
- Document **OSS** (hooks, nFPM/Snap file drops, Windows `.syso` recipes if desired) and **Pro** (`app_bundles.icon` → generated `.icns`, etc.).
- eletrocromo **emits** assets under `dist/icons`; **users** point packaging configs at those paths.

### Android packaging

- JIT host already used by android build; icon step must feed **mipmaps** (and related) into that workdir when building APKs.
- Reuse the same generator as `build icons` (shared code under `internal/…`, not the public runtime API).

### macOS packaging (normative; grill 2026-08-22)

Same wrapper idea as the APK. Same `eletrocromo.json`. Packaging tricks from **rterm** (XcodeGen, ad-hoc unsigned, no sandbox) — not rterm’s SwiftTerm UI or fill-screen chrome.

| Rule | Decision |
|------|----------|
| Command | `eletrocromo build macos` |
| Config | Same keys as android. **No new fields** in v1. |
| `package_id` | `CFBundleIdentifier` |
| `app_name` | `CFBundleName` / `CFBundleDisplayName` |
| `version_name` | `CFBundleShortVersionString` |
| `version_code` | `CFBundleVersion` |
| `icon` / `--icon` | Existing icon pipeline; copy `macos/icon.icns` into the bundle |
| `go_main` | `CGO_ENABLED=0` darwin binary for the **host arch** |
| Default `--out` | `dist/<app_name>.app` |
| Flags | Same family as android: `--config`, `--out`, `--workdir`, `--go-only`, `--icon`, `--refresh-icons`, identity/version overrides |
| Full `.app` | Mac with **Xcode** (`xcodegen` + `xcodebuild`) |
| `--go-only` | Write the host tree + darwin Go binary on any OS; skip `xcodebuild` |
| Artifact | One **unsigned Debug** `.app`. Ad-hoc sign (`CODE_SIGN_IDENTITY="-"`). |
| Arch | **Host arch only** |
| Sandbox | **Off** |
| Min OS | **14.0** |
| Chrome | Splash + status until READY; Retry on failure; normal title bar; `⌘R` reloads |
| Lifetime | **Not a Mac special case.** Same two desktop modes (window-owned vs tray). Those modes are still pending on Linux. |
| Off-loopback http(s) | Open in the **default browser**. Loopback stays in WKWebView. `target=_blank` / `window.open` use the same rule. |
| ATS | Cleartext only for `127.0.0.1` / `localhost` (same policy as Android `network_security_config`) |
| Inspector | On (`isInspectable`) |
| Icons | Run `build icons` when `macos/icon.icns` is missing (or `--refresh-icons`) |
| Handshake | `ELETROCROMO_NO_UI=1`, `ELETROCROMO_NO_ENSURE=1`, `ELETROCROMO_READY` / `ELETROCROMO_READY_FILE` (same as Android) |

v1 macos non-goals:

- DMG, notarization, Developer ID, universal `lipo`
- App Sandbox, App Store, prebuilt Mach-O stub, new json keys
- Camera/mic usage strings, custom URL schemes
- Native menus beyond system defaults
- rterm notch / fill-screen
- Darwin Helium discovery (desktop `Run()` track)

Success test:

1. On a Mac with Xcode: `eletrocromo build macos --config examples/counter/eletrocromo.json`
2. Open `dist/Counter.app` (Gatekeeper: right-click → Open).
3. Splash until READY; counter `+` / `−` work.
4. A non-loopback `https://` link opens the default browser.
5. `--go-only` on any OS writes a host tree + darwin Go binary and does not need Xcode.

### `eletrocromo.json` (packaging config)

Existing fields remain (`schema_version`, `package_id`, `app_name`, `go_main`, `abis`, version fields as implemented). Add:

| Field | Rule |
|-------|------|
| `icon` | Optional path to master PNG/SVG; relative to the config file’s directory. Empty/absent → default mark |

`package_id` is the Android applicationId **and** the Mac `CFBundleIdentifier`. `abis` applies to android only. Flags always override config for a single invocation.

## Implementation notes (today → target)

| Area | Today (approx.) | Target |
|------|-----------------|--------|
| Entry | `App{Handler, Context, …}.Run()` | Handler **or** `*http.Server` constructors; bind always library-owned |
| Server | `httptest` | Keep ephemeral loopback; do not hand bind to the app |
| Auth | Token + cookie; fail-closed | Keep always-on; no opt-out in v1 |
| Launch | Helium-only resolve + workspaced ensure (in progress / landed per code) | Helium only → `workspaced tool which helium-browser helium` (bootstrap workspaced) → `--app` |
| Lifetime | Context cancel; browser lifecycle partial | Default window-owned; flag background + tray |
| Tray / lockfile | Absent or partial | Linux v1 requirement |
| Example | counter / ticker / basic / astro | Template counter + mode flag as dogfood bar |
| Packaging CLI | `build icons` / `build android` / `build macos`; default mark; full icon matrix; taskgroup | `build ios` scaffold: JIT Debug `.app` (WKWebView + c-archive). Grill before treating as product |
| workspaced dep | Subprocess only (library + CLI) | Library: subprocess only. CLI: may `require` workspaced modules + tool ensure |
| README | Architecture + CLI + APK blurb | Align with this SPEC (desktop + packaging) |

## Success criteria

**Desktop runtime**

- A developer writes only HTTP/template logic and gets a usable Linux “desktop” window **under Helium**.
- Clean machine with network: first `Run` can bootstrap workspaced + ensure `helium-browser`, then open `--app` (no system browser).
- Offline with no local host: fails loudly; never opens the default browser.
- CGO=0 builds and runs the dogfood counter on Linux.
- Background mode is usable daily without ever typing the loopback token URL.
- The project description never requires “we’ll add native menus next” to feel complete.

**Packaging**

- `eletrocromo build icons` with no config produces a complete `dist/icons` tree from the default mark.
- With `icon` / `--icon`, the same tree is derived from the user master (pad+center).
- `eletrocromo build android` produces an APK whose launcher icon is not the Android system placeholder when icons were generated.
- `eletrocromo build macos` on a Mac with Xcode produces an unsigned Debug `.app` whose icon is the generated `macos/icon.icns` and whose window is WKWebView (not Helium).
- `--go-only` for macos writes the host tree + darwin Go binary without Xcode.
- Documented GoReleaser hook can call `build icons` without eletrocromo owning the release.

## Implementation order (toward full SPEC)

**Desktop**

1. **Launch contract:** Helium only; remove other Chromium-likes and system-browser fallback; hard error.
2. **Ensure via workspaced on PATH:** `tool which helium-browser helium` → `--app`.
3. **Bootstrap workspaced binary** (pinned + verified cache) when missing.
4. Window-owned lifetime; background + tray + lockfile.
5. Handler / `*http.Server` constructors; template counter dogfood; README ↔ SPEC.

**Packaging (can proceed in parallel with desktop tray work)**

1. Vend default **mark** + **lockup**; `internal` icon generator skeleton + `manifest.json`.
2. `eletrocromo build` parent (bare → error) + `build icons` (libs + workspaced tools).
3. Wire `icon` / `--icon` / `--output` / `--refresh-icons` + missing-only policy.
4. Fold android into `build android`; icons dep; mipmaps into JIT workdir; demote `create`.
5. Docs: GR OSS + Pro recipes; README packaging section.
6. `build macos`: ephemeral XcodeGen + Swift WKWebView host; host-arch Go; unsigned Debug `.app`; icons → icns; `--go-only`.
7. `build ios` (scaffold): ephemeral XcodeGen + UIKit WKWebView; `c-archive` (`EletrocromoStart`); READY file; simulator Debug; `--go-only`. Grill before App Store / device product claims.

## Open implementation choices (not product questions)

Resolved by engineering when building, not by re-litigating product meaning:

- Exact flag names (`--tray`, `--background`, `ELETROCROMO_NO_ENSURE`, …)
- Lockfile path and resume protocol (signal, local socket, etc.)
- How window-death is detected under CGo-less constraints (browser process wait, WM heuristics, …)
- Whether tray is a build-tagged Linux file set vs always compiled stubs
- Precise constructor names and option functional options vs struct fields
- Workspaced release pin value, checksum source, and cache layout under XDG
- Whether Helium catalog version is left floating to workspaced or pinned later
- Exact PNG/ICO/ICNS size lists and Android density set
- Which workspaced catalog tool names back SVG/ICO/ICNS conversion
- Precise “outputs missing” checklist for skip-vs-generate
- Whether `cmd/eletrocromo` stays `CGO_ENABLED=0` while shelling to external converters
- Module path/version pin for importing workspaced (`taskgroup`, …)
- AppKit vs SwiftUI for the WKWebView shell (dumb splash + WebView, not rterm chrome)
- Path of the Go child inside the `.app` (`Contents/MacOS` vs `Helpers`)
- Ephemeral project shape (XcodeGen `project.yml` vs checked-in template xcodeproj)
- How host arch is detected (`runtime.GOARCH` vs `uname`)
- Exact Info.plist ATS keys and splash asset (app icns vs default mark)

---

*Aligned in grill sessions. Do not expand scope into non-goals without a new explicit decision.*

- 2026-07-20: Helium-only desktop shell. workspaced `helium-browser`. No other Chromium-likes. No system-browser fallback for desktop `Run()`.
- 2026-07-23: Icon matrix. `build icons` / `build android`. Default mark/lockup. GoReleaser hooks, not a plugin. taskgroup import for packaging only.
- 2026-08-22: JIT unsigned Debug `.app`. WKWebView host. Same json as android. rterm packaging tricks, not rterm UI. Off-loopback links open in the default browser.
