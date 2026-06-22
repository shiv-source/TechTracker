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
	ID             int          `json:"id"`
	GroupName      string       `json:"groupName"`
	InputFilePath  string       `json:"InputFilePath"`
	OutputFilePath string       `json:"OutputFilePath"`
	Repositories   []Repository `json:"repositories"`
}

// Delta holds the difference between two snapshots for a single repository.
type Delta struct {
	FullName   string `json:"full_name"`
	RankChange int    `json:"rank_change"` // positive = moved up in rank
	StarDelta  int    `json:"star_delta"`
	PrevScore  float64 `json:"prev_score"`
}

// RunMetadata captures information about a single execution run.
type RunMetadata struct {
	Version    string         `json:"version"`
	StartTime  string         `json:"start_time"`
	EndTime    string         `json:"end_time"`
	DurationMs int64          `json:"duration_ms"`
	Groups     map[string]int `json:"groups"` // group name → repo count
	Errors     []string       `json:"errors,omitempty"`
	DryRun     bool           `json:"dry_run"`
}
