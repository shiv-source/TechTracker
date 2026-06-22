package renderer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shiv-source/TechTracker/internal/models"
)

func TestSaveAllChunks(t *testing.T) {
	dir := t.TempDir()

	repos := make([]models.Repository, 120)
	for i := range repos {
		repos[i] = models.Repository{FullName: fmt.Sprintf("repo/%d", i), Stars: i * 10, Score: float64(i) / 120.0}
	}

	count, err := SaveAllChunks(repos, dir)
	if err != nil {
		t.Fatalf("SaveAllChunks failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 chunks for 120 repos, got %d", count)
	}

	// Verify chunk files exist and have correct sizes.
	for i := 1; i <= 3; i++ {
		path := filepath.Join(dir, "all", fmt.Sprintf("chunk_%d.json", i))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("chunk_%d.json not found: %v", i, err)
		}
		var loaded []models.Repository
		if err := json.Unmarshal(data, &loaded); err != nil {
			t.Fatal(err)
		}
		expected := 50
		if i == 3 {
			expected = 20
		}
		if len(loaded) != expected {
			t.Errorf("chunk %d: expected %d repos, got %d", i, expected, len(loaded))
		}
	}
}

func TestSaveGroupFile(t *testing.T) {
	dir := t.TempDir()

	repos := []models.Repository{
		{FullName: "a/b", Stars: 100, Score: 0.95},
		{FullName: "c/d", Stars: 50, Score: 0.5},
	}

	err := SaveGroupFile(repos, dir, "test_group")
	if err != nil {
		t.Fatalf("SaveGroupFile failed: %v", err)
	}

	path := filepath.Join(dir, "groups", "test_group.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded []models.Repository
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Errorf("expected 2 repos, got %d", len(loaded))
	}
}

func TestSaveMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.json")

	meta := models.RunMetadata{
		Version: "2.0.0",
		History: models.HistoryInfo{
			AvailableDates: []string{"2026-06-20", "2026-06-21", "2026-06-22"},
			FirstDate:      "2026-06-20",
			LastDate:       "2026-06-22",
			Count:          3,
		},
	}

	err := SaveMetadata(meta, path)
	if err != nil {
		t.Fatalf("SaveMetadata failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded models.RunMetadata
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %s", loaded.Version)
	}
	if loaded.History.Count != 3 {
		t.Errorf("expected history count 3, got %d", loaded.History.Count)
	}
}

func TestFormatDate_Valid(t *testing.T) {
	result := formatDate("2024-01-15T10:30:00Z")
	if result != "2024-01-15" {
		t.Errorf("expected 2024-01-15, got %s", result)
	}
}

func TestFormatDate_Empty(t *testing.T) {
	if result := formatDate(""); result != "" {
		t.Errorf("expected empty string, got %s", result)
	}
}

func TestFormatDate_Invalid(t *testing.T) {
	// Should not panic on invalid date; returns the input as-is.
	result := formatDate("not-a-date")
	if result != "not-a-date" {
		t.Errorf("expected 'not-a-date' fallback, got %s", result)
	}
}

func TestFormatTrend(t *testing.T) {
	deltas := map[string]models.Delta{
		"a/b": {RankChange: 2, StarDelta: 150},
		"c/d": {RankChange: -1, StarDelta: -10},
		"e/f": {RankChange: 0, StarDelta: 0},
	}

	trend := formatTrend("a/b", deltas)
	if !strings.Contains(trend, "+150") || !strings.Contains(trend, "↑") {
		t.Errorf("unexpected trend for a/b: %s", trend)
	}
	if !strings.Contains(trend, "+2") || !strings.Contains(trend, "▲") {
		t.Errorf("expected rank +2 ▲ in: %s", trend)
	}

	trend = formatTrend("c/d", deltas)
	if !strings.Contains(trend, "▼") || !strings.Contains(trend, "↓") {
		t.Errorf("expected ▼ and ↓ in: %s", trend)
	}

	// No deltas for e/f means all zeros → empty trend.
	trend = formatTrend("e/f", deltas)
	if trend != "" {
		t.Errorf("expected empty trend for no changes, got: %s", trend)
	}
}

func TestFormatTrend_NilMap(t *testing.T) {
	if result := formatTrend("a/b", nil); result != "" {
		t.Errorf("expected empty for nil deltas, got %s", result)
	}
}

func TestEscapeMarkdown(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"a|b", "a\\|b"},
		{"a|b|c", "a\\|b\\|c"},
		{"", ""},
	}
	for _, tc := range tests {
		result := escapeMarkdown(tc.input)
		if result != tc.expected {
			t.Errorf("escapeMarkdown(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestBuildTable_RendersCorrectColumns(t *testing.T) {
	repos := []models.Repository{
		{Name: "testrepo", URL: "https://github.com/test/repo", Stars: 100, Forks: 10, Issues: 5, Language: "Go", Description: "A test repo", UpdatedAt: "2024-01-15T10:30:00Z"},
	}
	header := []string{"SL", "Name", "Stars", "Forks", "Issues", "Language", "Trend", "Description", "UpdatedAt"}
	table := buildTable(header, repos, nil)

	if !strings.Contains(table, "testrepo") {
		t.Error("table should contain repo name")
	}
	if !strings.Contains(table, "2024-01-15") {
		t.Error("table should contain formatted date")
	}
	if !strings.Contains(table, "---") {
		t.Error("table should contain separator row")
	}
}
