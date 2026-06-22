package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/shiv-source/TechTracker/internal/models"
	"github.com/shiv-source/TechTracker/internal/scorer"
	"github.com/shiv-source/TechTracker/utils"
)

// SnapshotPath returns the path to the snapshot file for a given date directory.
// Format: data/<YYYY-MM-DD>/snapshot.json
func SnapshotPath(dataDir string, date time.Time) string {
	return filepath.Join(dataDir, date.Format("2006-01-02"), "snapshot.json")
}

// SaveSnapshot writes a full daily snapshot inside a date-named directory.
func SaveSnapshot(dataDir string, date time.Time, repos []models.Repository) error {
	dateDir := filepath.Join(dataDir, date.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return fmt.Errorf("failed to create date dir: %w", err)
	}
	path := SnapshotPath(dataDir, date)
	return utils.SaveToJsonFile(repos, path)
}

// LoadSnapshot reads a historical snapshot for a given date.
func LoadSnapshot(dataDir string, date time.Time) ([]models.Repository, error) {
	path := SnapshotPath(dataDir, date)
	result, err := utils.LoadJSONFromFile[[]models.Repository](path)
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// LatestSnapshot finds the most recent snapshot by scanning date-named directories.
// Returns the repositories, the date, or an error if no snapshots exist.
func LatestSnapshot(dataDir string) ([]models.Repository, time.Time, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to read data dir: %w", err)
	}

	var latest time.Time
	var latestPath string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue
		}
		snapPath := filepath.Join(dataDir, entry.Name(), "snapshot.json")
		if _, err := os.Stat(snapPath); os.IsNotExist(err) {
			continue
		}
		if t.After(latest) {
			latest = t
			latestPath = snapPath
		}
	}

	if latestPath == "" {
		return nil, time.Time{}, fmt.Errorf("no snapshots found in %s", dataDir)
	}

	repos, err := utils.LoadJSONFromFile[[]models.Repository](latestPath)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to load snapshot %s: %w", latestPath, err)
	}

	return *repos, latest, nil
}

// ComputeDeltas compares current rankings against a previous snapshot and returns
// a map of full_name → Delta with rank changes and star deltas.
func ComputeDeltas(current, previous []models.Repository, weights models.Weights) map[string]models.Delta {
	// Score previous repos using the same weights to get fair rank comparison.
	prevScored := scorer.ScoreRepositories(copyRepos(previous), weights)

	// Build maps: full_name → rank (1-based) and star count.
	prevRank := make(map[string]int)
	prevStars := make(map[string]int)
	for i, r := range prevScored {
		prevRank[r.FullName] = i + 1
		prevStars[r.FullName] = r.Stars
	}

	deltas := make(map[string]models.Delta)
	for i, cur := range current {
		curRank := i + 1
		d := models.Delta{
			FullName:  cur.FullName,
			StarDelta: 0,
			PrevScore: 0,
		}
		if prevRank, ok := prevRank[cur.FullName]; ok {
			// Higher rank = lower number, so prevRank - curRank is positive when moving up.
			d.RankChange = prevRank - curRank
			d.StarDelta = cur.Stars - prevStars[cur.FullName]
			if pScore, ok := findPrevScore(prevScored, cur.FullName); ok {
				d.PrevScore = pScore
			}
		}
		deltas[cur.FullName] = d
	}

	return deltas
}

func copyRepos(repos []models.Repository) []models.Repository {
	cp := make([]models.Repository, len(repos))
	copy(cp, repos)
	return cp
}

func findPrevScore(repos []models.Repository, fullName string) (float64, bool) {
	for _, r := range repos {
		if r.FullName == fullName {
			return r.Score, true
		}
	}
	return 0, false
}

// PruneOldSnapshots removes date-named directories older than maxDays.
func PruneOldSnapshots(dataDir string, maxDays int) error {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -maxDays)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := time.Parse("2006-01-02", entry.Name())
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.RemoveAll(filepath.Join(dataDir, entry.Name()))
		}
	}
	return nil
}

// SortGroupsByID sorts groups by their ID field.
func SortGroupsByID(groups []models.GroupResult) {
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ID < groups[j].ID
	})
}
