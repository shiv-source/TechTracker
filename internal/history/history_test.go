package history

import (
	"os"
	"testing"
	"time"

	"github.com/shiv-source/TechTracker/internal/models"
)

func defaultWeights() models.Weights {
	return models.Weights{
		Stars: 0.4, Forks: 0.25, Watchers: 0.2, Subscribers: 0.1, Issues: 0.05,
	}
}

func TestSaveAndLoadSnapshot(t *testing.T) {
	dir := t.TempDir()
	date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	repos := []models.Repository{
		{FullName: "a/b", Stars: 100, Score: 0.9},
		{FullName: "c/d", Stars: 50, Score: 0.5},
	}

	err := SaveSnapshot(dir, date, repos)
	if err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	// Verify file exists.
	path := SnapshotPath(dir, date)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("snapshot file not created")
	}

	// Load back.
	loaded, err := LoadSnapshot(dir, date)
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 repos, got %d", len(loaded))
	}
	if loaded[0].FullName != "a/b" {
		t.Errorf("expected a/b, got %s", loaded[0].FullName)
	}
}

func TestSnapshotPath_Format(t *testing.T) {
	dir := "/tmp/data"
	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	path := SnapshotPath(dir, date)
	expected := "/tmp/data/2024-06-15/snapshot.json"
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestLatestSnapshot_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, _, err := LatestSnapshot(dir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestLatestSnapshot_FindsMostRecent(t *testing.T) {
	dir := t.TempDir()

	// Save two snapshots.
	date1 := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	SaveSnapshot(dir, date1, []models.Repository{{FullName: "old", Stars: 1}})
	SaveSnapshot(dir, date2, []models.Repository{{FullName: "new", Stars: 2}})

	repos, latest, err := LatestSnapshot(dir)
	if err != nil {
		t.Fatalf("LatestSnapshot failed: %v", err)
	}
	if latest.YearDay() != 15 { // Jan 15
		t.Errorf("expected latest Jan 15, got %v", latest)
	}
	if len(repos) == 0 || repos[0].FullName != "new" {
		t.Errorf("expected 'new' repo, got %v", repos)
	}
}

func TestComputeDeltas_RankChange(t *testing.T) {
	prev := []models.Repository{
		{FullName: "a/first", Stars: 100},
		{FullName: "b/second", Stars: 80},
		{FullName: "c/third", Stars: 60},
	}

	// b/second overtakes a/first in stars.
	curr := []models.Repository{
		{FullName: "b/second", Stars: 200},
		{FullName: "a/first", Stars: 150},
		{FullName: "c/third", Stars: 60},
	}

	deltas := ComputeDeltas(curr, prev, defaultWeights())

	// b/second was rank 2, now rank 1 → rankChange = 2-1 = +1.
	if d, ok := deltas["b/second"]; ok {
		if d.RankChange != 1 {
			t.Errorf("expected b/second rankChange +1, got %d", d.RankChange)
		}
		if d.StarDelta != 120 {
			t.Errorf("expected b/second starDelta 120, got %d", d.StarDelta)
		}
	} else {
		t.Error("missing delta for b/second")
	}

	// a/first was rank 1, now rank 2 → rankChange = 1-2 = -1.
	if d, ok := deltas["a/first"]; ok {
		if d.RankChange != -1 {
			t.Errorf("expected a/first rankChange -1, got %d", d.RankChange)
		}
	}

	// c/third unchanged.
	if d, ok := deltas["c/third"]; ok {
		if d.RankChange != 0 {
			t.Errorf("expected c/third rankChange 0, got %d", d.RankChange)
		}
	}
}

func TestComputeDeltas_NewRepo(t *testing.T) {
	prev := []models.Repository{
		{FullName: "a/old", Stars: 100},
	}
	curr := []models.Repository{
		{FullName: "a/old", Stars: 110},
		{FullName: "b/new", Stars: 200},
	}

	deltas := ComputeDeltas(curr, prev, defaultWeights())
	if d, ok := deltas["b/new"]; ok {
		// New repo not in previous snapshot → no rank change or star delta.
		if d.RankChange != 0 || d.StarDelta != 0 {
			t.Errorf("new repo should have zero deltas")
		}
	}
}

func TestPruneOldSnapshots(t *testing.T) {
	dir := t.TempDir()

	oldDate := time.Now().AddDate(0, 0, -100)
	recentDate := time.Now().AddDate(0, 0, -1)

	SaveSnapshot(dir, oldDate, []models.Repository{{FullName: "old"}})
	SaveSnapshot(dir, recentDate, []models.Repository{{FullName: "recent"}})

	err := PruneOldSnapshots(dir, 30)
	if err != nil {
		t.Fatalf("PruneOldSnapshots failed: %v", err)
	}

	// Old snapshot should be gone.
	if _, err := LoadSnapshot(dir, oldDate); err == nil {
		t.Error("old snapshot should have been pruned")
	}
	// Recent snapshot should still exist.
	if _, err := LoadSnapshot(dir, recentDate); err != nil {
		t.Error("recent snapshot should still exist")
	}
}
