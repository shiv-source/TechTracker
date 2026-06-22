# 🌟 TechTracker

**Live technology rankings at [shiv-source.github.io/TechTracker](https://shiv-source.github.io/TechTracker/)**

TechTracker tracks the popularity of open-source programming languages, frameworks, databases, testing tools, and DevOps tools using real-time GitHub metrics. Rankings are updated automatically every 24 hours and published to the web dashboard.

**Built with:** Go 1.26 · HTML/CSS/JS · Tailwind CSS v3 (CDN)

## 📊 Live Dashboard

👉 **[shiv-source.github.io/TechTracker](https://shiv-source.github.io/TechTracker/)**

The dashboard features:
- **Interactive rankings** — sort, search, and filter across 8 technology categories
- **Trend charts** — score history and movement over time
- **Dark/light themes** — respects your system preference
- **100+ technologies** tracked daily via GitHub Actions

## 📂 Raw Data

Machine-readable JSON is available in the [`data/`](data/) directory:

| File | Description |
|------|-------------|
| [`data/all.json`](data/all.json) | Combined global rankings with scores |
| [`data/frontend_frameworks.json`](data/frontend_frameworks.json) | Frontend frameworks |
| [`data/backend_frameworks.json`](data/backend_frameworks.json) | Backend frameworks |
| [`data/mobile_frameworks.json`](data/mobile_frameworks.json) | Mobile application frameworks |
| [`data/testing_tools.json`](data/testing_tools.json) | Testing tools |
| [`data/devops_tools.json`](data/devops_tools.json) | DevOps tools |
| [`data/databases.json`](data/databases.json) | Databases |
| [`data/programming_languages.json`](data/programming_languages.json) | Programming languages |
| [`data/design_tools.json`](data/design_tools.json) | Design tools |
| [`data/run-metadata.json`](data/run-metadata.json) | Last run timing and status |

## 📊 Score Calculation

The popularity score is a weighted composite of five GitHub metrics, normalized via min-max scaling within each category:

| Metric | Weight |
|--------|--------|
| Stars | 40% |
| Forks | 25% |
| Watchers | 20% |
| Subscribers | 10% |
| Open Issues | 5% |

All metrics are normalized to [0, 1] before weighting. Per-category weights can be customized in [`config.json`](config.json).

## 🤝 Contributing

To add or update tracked technologies:

1. Open the [`projects/`](projects/) directory
2. Edit the relevant `.txt` file (one GitHub URL per line)
3. Submit a pull request

Example:
```
https://github.com/facebook/react
https://github.com/vuejs/vue
```

## ⚙️ Running Locally

**Prerequisites:** Go 1.26

```bash
# Backend (Go)
export GITHUB_TOKEN=your_github_token
make build
make run

# Frontend (UI dashboard)
cd ui
python3 -m http.server 8080    # Serve from project root, open /ui/
```

See [`CLAUDE.md`](CLAUDE.md) for full development documentation.

## 📜 License

MIT — see [LICENSE](LICENSE).
