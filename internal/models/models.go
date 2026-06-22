package models

// Repository represents a GitHub repository with its metrics and computed score.
type Repository struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	FullName    string  `json:"full_name"`
	URL         string  `json:"html_url"`
	Description string  `json:"description"`
	Stars       int     `json:"stargazers_count"`
	Forks       int     `json:"forks_count"`
	Watchers    int     `json:"watchers_count"`
	Subscribers int     `json:"subscribers_count"`
	Issues      int     `json:"open_issues_count"`
	Language    string  `json:"language"`
	UpdatedAt   string  `json:"updated_at"`
	Score       float64 `json:"score"`

	// Trend fields (computed from historical comparison)
	PrevScore  float64 `json:"prev_score,omitempty"`
	RankChange int     `json:"rank_change,omitempty"` // positive = moved up
	StarDelta  int     `json:"star_delta,omitempty"`
}

// AppConfig is the top-level configuration structure for config.json.
type AppConfig struct {
	RetentionDays int      `json:"retention_days"`
	Groups        []Config `json:"groups"`
}

// Config represents a single group configuration from config.json.
type Config struct {
	ID        int      `json:"id"`
	GroupName string   `json:"groupName"`
	FilePath  string   `json:"filePath"`
	Weights   *Weights `json:"weights,omitempty"` // per-group override, nil = use defaults
}

// Weights defines the weighted contribution of each metric to the final score.
type Weights struct {
	Stars       float64 `json:"stars"`
	Forks       float64 `json:"forks"`
	Watchers    float64 `json:"watchers"`
	Subscribers float64 `json:"subscribers"`
	Issues      float64 `json:"issues"`
}

// GroupResult holds the processed results for a single group.
type GroupResult struct {
	ID            int          `json:"id"`
	GroupName     string       `json:"groupName"`
	GroupKey      string       `json:"groupKey"`
	InputFilePath string       `json:"inputFilePath"`
	Repositories  []Repository `json:"repositories"`
}

// Delta holds the difference between two snapshots for a single repository.
type Delta struct {
	FullName   string  `json:"full_name"`
	RankChange int     `json:"rank_change"` // positive = moved up in rank
	StarDelta  int     `json:"star_delta"`
	PrevScore  float64 `json:"prev_score"`
}

// Summary holds aggregate statistics for the UI stats bar.
type Summary struct {
	TotalRepos int    `json:"total_repos"`
	Categories int    `json:"categories"`
	TopRepo    string `json:"top_repo"`
	TopStars   int    `json:"top_stars"`
}

// CategoryMeta describes a single category for the UI tab bar.
type CategoryMeta struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// HistoryInfo tracks what historical snapshots are available.
type HistoryInfo struct {
	AvailableDates []string `json:"available_dates"`
	FirstDate      string   `json:"first_date,omitempty"`
	LastDate       string   `json:"last_date,omitempty"`
	Count          int      `json:"count"`
}

// RunMetadata captures information about a single execution run.
// Written to both data/<date>/metadata.json and data/metadata.json (root catalog).
type RunMetadata struct {
	Version    string         `json:"version"`
	LatestDate string         `json:"latest_date"`
	StartTime  string         `json:"start_time"`
	EndTime    string         `json:"end_time"`
	DurationMs int64          `json:"duration_ms"`
	Summary    Summary        `json:"summary"`
	Categories []CategoryMeta `json:"categories"`
	AllChunks  int            `json:"all_chunks"`
	History    HistoryInfo    `json:"history"`
	Errors     []string       `json:"errors,omitempty"`
	DryRun     bool           `json:"dry_run"`
}

// Top5Entry is a single repo's score on a given date (used in top5_history.json).
type Top5Entry struct {
	FullName string  `json:"full_name"`
	Score    float64 `json:"score"`
}

// Top5History maps dates to their top-5 repos (appended to data/top5_history.json each run).
type Top5History map[string][]Top5Entry
