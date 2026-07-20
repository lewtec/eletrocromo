# Astro + orvalho workers + eletrocromo

**Astro SSR** (Cloudflare adapter): cat fact in **frontmatter** with `fetch` on
each request. No client JS, no offline fallback.

The orvalho guest script and client assets are **`//go:embed`’d** into the Go
binary after a pre-bundle step (no runtime esbuild).

## Flow

1. `mise run astro` — `bun run build` (Astro) → `dist/{server,client}`
2. `mise run assemble` — CF server → `worker/`
3. `mise run embed` — orvalho `BundleEntry` → `embed/guest.js` + `embed/assets/`
4. `go run .` — `//go:embed` guest + assets → eletrocromo window

`mise run build` runs astro → embed (includes assemble).

## Prerequisites

- [mise](https://mise.jdx.dev/) — [`mise.toml`](./mise.toml) (`bun`; go from parent mise)
- Local [orvalho](https://github.com/lucasew/orvalho) checkout (see `go.mod` `replace`)

## Run

```bash
mise install
mise run build   # produces embed/ for //go:embed
mise run run
```
