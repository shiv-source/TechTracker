package history

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/shiv-source/TechTracker/internal/models"
	"github.com/shiv-source/TechTracker/internal/scorer"
	"github.com/shiv-source/TechTracker/utils"
)

// SnapshotPath returns the file path for a history snapshot on a given date.
func SnapshotPath(dir string, date time.Time) string {
	return filepath.Join(dir, date.Format("2006-01-02")+".json")
}

// SaveSnapshot writes a full daily snapshot to the history directory.
func SaveSnapshot(dir string, date time.Time, repos []models.Repository) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history dir: %w", err)
	}
	path := SnapshotPath(dir, date)
	return utils.SaveToJsonFile(repos, path)
}

// LoadSnapshot reads a historical snapshot for a given date.
func LoadSnapshot(dir string, date time.Time) ([]models.Repository, error) {
	path := SnapshotPath(dir, date)
	result, err := utils.LoadJSONFromFile[[]models.Repository](path)
	if err != nil {
		return nil, err
	}
	return *result, nil
}

// LatestSnapshot finds the most recent snapshot in the history directory.
// Returns the repositories, the date, or an error if no snapshots exist.
func LatestSnapshot(dir string) ([]models.Repository, time.Time, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to read history dir: %w", err)
	}

	var latest time.Time
	var latestFile string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		t, err := time.Parse("2006-01-02", name)
		if err != nil {
			continue
		}
		if t.After(latest) {
			latest = t
			latestFile = entry.Name()
		}
	}

	if latestFile == "" {
		return nil, time.Time{}, fmt.Errorf("no snapshots found in %s", dir)
	}

	path := filepath.Join(dir, latestFile)
	repos, err := utils.LoadJSONFromFile[[]models.Repository](path)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("failed to load snapshot %s: %w", latestFile, err)
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

// PruneOldSnapshots removes snapshots older than maxDays from the history directory.
func PruneOldSnapshots(dir string, maxDays int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -maxDays)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		t, err := time.Parse("2006-01-02", name)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.Remove(filepath.Join(dir, entry.Name()))
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
