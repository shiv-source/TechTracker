# 🌟 TechTracker

Stay at the forefront of the development landscape with **TechTracker**, a meticulously curated ranking of the most popular **programming languages**, **frameworks**, **databases**, **testing tools**, and **DevOps tools**. The popularity score for each technology is determined by evaluating a set of key metrics that reflect its community engagement, adoption, and overall usage.

🔄 **Contribute:** To add or update tracked technologies, simply modify the respective `.txt` file in the [`projects`](projects) directory.

📜 This project is licensed under the [MIT License](LICENSE).

📊 **Live Dashboard:** View the interactive dashboard at [GitHub Pages](https://shiv-source.github.io/TechTracker/)

---

{{.Table}}

---

**⏰ Last Updated:** {{.LastUpdated}}  
_This page is updated automatically every 24 hours to ensure the latest rankings are displayed._

---

## 📊 Trend Indicators

- **▲ ▼** — Rank change since yesterday (▲ = moved up, ▼ = moved down)
- **↑ ↓** — Star count change since yesterday (+N ↑ = gained N stars)
- Trend data appears after the second daily run when historical snapshots are available.

---

## 📂 JSON Data Availability

Looking for programmatic access to this data? The rankings and other details are available in their respective `.json` files inside the [`data`](data) folder.

---

## 📊 Score Calculation

The popularity score of each technology (programming language, framework, database, testing tool, or DevOps tool) is calculated by considering various factors such as Stars, Forks, Watchers, Subscribers, and Issues. Each of these metrics has a weighted contribution to the final score, giving a more comprehensive view of its standing in the developer community.

- **Stars (40%)**: The number of stars a project has on GitHub, indicating its popularity.
- **Forks (25%)**: The number of forks, which represents how often the tool or technology is being used as a base for other projects.
- **Watchers (20%)**: The number of watchers, reflecting interest in the repository for updates.
- **Subscribers (10%)**: The number of users subscribed to the repository for notifications on new changes.
- **Issues (5%)**: The number of open issues, with a higher number of issues affecting the overall score.

The score is calculated using a weighted average of these metrics. Each metric is multiplied by its respective weight, and the total score is normalized to fall between 0 and 1.

Example formula:

```
Score = (Stars * 0.4) + (Forks * 0.25) + (Watchers * 0.2) + (Subscribers * 0.1) + (Issues * 0.05)
```

---

## 🤝 Contribution

We welcome contributions from the community! To contribute:

1. Open the [`projects`](projects) directory.
2. Add a new line with the GitHub repository URL of the technology (programming language, framework, database, testing tool, or DevOps tool) in the respective `.txt` file. Each item should have its own line.  
   Example:

   ```
   https://github.com/facebook/react
   https://github.com/angular/angular
   https://github.com/nodejs/node
   https://github.com/mysql/mysql-server
   ```

3. Save the file, commit your changes, and submit a pull request.

---

## ⚖️ License

This project is open source and available under the [MIT License](LICENSE). Feel free to use, share, and contribute!
