# CLAUDE.md

TechTracker is a Go CLI tool that tracks GitHub repository metrics (stars, forks, watchers, subscribers, open issues) for curated lists of open-source technologies, computes weighted popularity scores via min-max normalization, and outputs ranked JSON + Markdown leaderboards. A GitHub Actions bot runs it daily and publishes results to GitHub Pages.

**Requirements:** Go 1.26 · Node.js 24 · pnpm v11

## Project Structure

```
cmd/techtracker/main.go       — Entry point, CLI flags, orchestration
internal/
  models/models.go            — Shared types (Repository, Config, Weights, GroupResult, Delta, RunMetadata)
  config/config.go            — Load/validate config.json, weight resolution
  fetcher/fetcher.go          — GitHub API client, rate limiter, retry logic
  scorer/scorer.go            — Min-max normalization, weighted scoring, sorting
  renderer/renderer.go        — JSON file output, Markdown table generation
  history/history.go          — Daily snapshots, delta computation, pruning
utils/utils.go                — File I/O helpers (JSON, Markdown, URL loading)
config.json                   — Category definitions with optional per-group weights
template.md                   — Go html/template for readme.md output
projects/*.txt                — Input: one GitHub URL per line per category
data/                         — Generated JSON output (per category + all.json + history/)
ui/                           — GitHub Pages dashboard (pure static: HTML + CSS + JS + Tailwind CDN)
```

## Build & Run

```bash
make build          # go build → bin/main
make run            # build + run (generates data/*.json + readme.md)
make test           # go test -v -race ./... (24 tests across 4 packages)
make vet            # go vet ./...
make lint           # golangci-lint run ./...
make clean-stack    # rm -rf data readme.md
make docker         # docker build -t techtracker .
```

## CLI Flags

```
--group <name>      Process only one category (e.g. "Frontend Frameworks")
--dry-run           Print output to stdout, don't write files
--verbose           Enable detailed logging
```

## UI Development

Zero build step — pure static HTML + CSS + JS. Tailwind CSS v3 via CDN.

```bash
cd ui
python3 -m http.server 8080  # Serve from project root for data/ access
# Or: npx serve . -p 3000
```

**Structure:** `index.html` · `css/style.css` (Nebula design system) · `js/app.js` (vanilla JS)

The design system uses CSS custom properties for light/dark themes, glassmorphism panels, animated dot-grid background, floating orbs, neon accents, and HUD-style stat cards.

## Key Design Decisions

- **Zero external Go dependencies** beyond stdlib — the old `markdownTable` dep was inlined into `renderer.go`.
- **Rate limiting**: Token-bucket at 30 req/min with exponential backoff on 429/403, max 5 concurrent requests.
- **Scoring**: Min-max normalization per metric within a category group. When max==min, normalized value is 0.5 to avoid NaN.
- **Historical tracking**: Full daily snapshots in `data/history/YYYY-MM-DD.json`, pruned after 90 days. Deltas computed by comparing scored rankings against the most recent previous snapshot.
- **Backward compatibility**: `make clean-stack && make run` still works exactly as before. Output paths unchanged.

## CI/CD

- **bot.yaml** — Daily at 2AM UTC: `make clean-stack && make build && make run`, commits results
- **ci.yaml** — On push to master: `go vet`, `go test`, `go build`
- **pages.yaml** — On push to master: deploys `ui/` directly to GitHub Pages (zero build step)

## Config

`config.json` is an array of group objects. Each group optionally specifies a `weights` object; if omitted, defaults are used (Stars:0.4, Forks:0.25, Watchers:0.2, Subscribers:0.1, Issues:0.05).
