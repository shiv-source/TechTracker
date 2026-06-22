package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shiv-source/TechTracker/internal/models"
)

func TestDefaultWeights(t *testing.T) {
	w := DefaultWeights()
	if w.Stars != 0.4 {
		t.Errorf("expected Stars=0.4, got %f", w.Stars)
	}
	if w.Forks != 0.25 {
		t.Errorf("expected Forks=0.25, got %f", w.Forks)
	}
}

func TestEffectiveWeights_UsesOverride(t *testing.T) {
	custom := models.Weights{Stars: 0.5, Forks: 0.5, Watchers: 0, Subscribers: 0, Issues: 0}
	cfg := models.Config{ID: 1, GroupName: "test", FilePath: "x.txt", Weights: &custom}
	result := EffectiveWeights(cfg)
	if result.Stars != 0.5 {
		t.Errorf("expected Stars=0.5 from override, got %f", result.Stars)
	}
}

func TestEffectiveWeights_UsesDefault(t *testing.T) {
	cfg := models.Config{ID: 1, GroupName: "test", FilePath: "x.txt", Weights: nil}
	result := EffectiveWeights(cfg)
	if result.Stars != 0.4 {
		t.Errorf("expected default Stars=0.4, got %f", result.Stars)
	}
}

func TestLoad_Valid(t *testing.T) {
	// Create a temporary config file.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	content := `{"retention_days": 14, "groups": [{"id": 1, "groupName": "Test", "filePath": "test.txt"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// Need a dummy test.txt file for Validate.
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), nil, 0644); err != nil {
		t.Fatal(err)
	}

	appConfig, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if appConfig.RetentionDays != 14 {
		t.Errorf("expected RetentionDays=14, got %d", appConfig.RetentionDays)
	}
	if len(appConfig.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(appConfig.Groups))
	}
	if appConfig.Groups[0].GroupName != "Test" {
		t.Errorf("expected 'Test', got %q", appConfig.Groups[0].GroupName)
	}
}

func TestValidate_EmptyName(t *testing.T) {
	configs := []models.Config{{ID: 1, GroupName: "", FilePath: "x.txt"}}
	err := Validate(configs)
	if err == nil {
		t.Error("expected error for empty group name")
	}
}

func TestValidate_DuplicateID(t *testing.T) {
	configs := []models.Config{
		{ID: 1, GroupName: "A", FilePath: "a.txt"},
		{ID: 1, GroupName: "B", FilePath: "b.txt"},
	}
	err := Validate(configs)
	if err == nil {
		t.Error("expected error for duplicate ID")
	}
}
