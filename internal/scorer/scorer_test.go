package scorer

import (
	"math"
	"testing"

	"github.com/shiv-source/TechTracker/internal/models"
)

func defaultWeights() models.Weights {
	return models.Weights{
		Stars: 0.4, Forks: 0.25, Watchers: 0.2, Subscribers: 0.1, Issues: 0.05,
	}
}

func TestScoreRepositories_Normal(t *testing.T) {
	repos := []models.Repository{
		{FullName: "a/b", Stars: 100, Forks: 50, Watchers: 30, Subscribers: 10, Issues: 5},
		{FullName: "c/d", Stars: 10, Forks: 5, Watchers: 3, Subscribers: 1, Issues: 0},
	}
	scored := ScoreRepositories(repos, defaultWeights())
	if len(scored) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(scored))
	}
	// First repo should have higher score.
	if scored[0].FullName != "a/b" {
		t.Errorf("expected a/b first, got %s", scored[0].FullName)
	}
	// Score should be between 0 and 1.
	for _, r := range scored {
		if r.Score < 0 || r.Score > 1 {
			t.Errorf("score %f for %s out of [0,1]", r.Score, r.FullName)
		}
	}
}

func TestScoreRepositories_SingleRepo(t *testing.T) {
	repos := []models.Repository{
		{FullName: "a/b", Stars: 50, Forks: 10, Watchers: 5, Subscribers: 3, Issues: 2},
	}
	scored := ScoreRepositories(repos, defaultWeights())
	if len(scored) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(scored))
	}
	// With a single repo, all normalized values are 0.5 (max==min).
	// Expected: 0.5*(0.4+0.25+0.2+0.1+0.05) = 0.5
	expected := math.Round(0.5*100000) / 100000.0
	if scored[0].Score != expected {
		t.Errorf("expected score %f, got %f", expected, scored[0].Score)
	}
}

func TestScoreRepositories_AllSameValues(t *testing.T) {
	repos := []models.Repository{
		{FullName: "a/b", Stars: 42, Forks: 10, Watchers: 5, Subscribers: 3, Issues: 2},
		{FullName: "c/d", Stars: 42, Forks: 10, Watchers: 5, Subscribers: 3, Issues: 2},
		{FullName: "e/f", Stars: 42, Forks: 10, Watchers: 5, Subscribers: 3, Issues: 2},
	}
	scored := ScoreRepositories(repos, defaultWeights())
	if len(scored) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(scored))
	}
	// All repos identical → all should have same score (0.5 after normalization).
	for _, r := range scored {
		expected := math.Round(0.5*100000) / 100000.0
		if r.Score != expected {
			t.Errorf("expected score %f, got %f for %s", expected, r.Score, r.FullName)
		}
	}
}

func TestScoreRepositories_AllZeros(t *testing.T) {
	repos := []models.Repository{
		{FullName: "a/b", Stars: 0, Forks: 0, Watchers: 0, Subscribers: 0, Issues: 0},
		{FullName: "c/d", Stars: 0, Forks: 0, Watchers: 0, Subscribers: 0, Issues: 0},
	}
	scored := ScoreRepositories(repos, defaultWeights())
	if len(scored) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(scored))
	}
	// All zeros, max==min → normalized = 0.5 for each metric.
	expected := math.Round(0.5*100000) / 100000.0
	for _, r := range scored {
		if r.Score != expected {
			t.Errorf("expected %f, got %f", expected, r.Score)
		}
	}
	// Scores should not be NaN.
	for _, r := range scored {
		if math.IsNaN(r.Score) {
			t.Error("score is NaN (divide by zero not guarded)")
		}
	}
}

func TestScoreRepositories_EmptySlice(t *testing.T) {
	scored := ScoreRepositories([]models.Repository{}, defaultWeights())
	if len(scored) != 0 {
		t.Errorf("expected empty, got %d", len(scored))
	}
}

func TestNormalize_EdgeCases(t *testing.T) {
	// max == min → 0.5.
	if v := normalize(10, 10, 10); v != 0.5 {
		t.Errorf("expected 0.5 when max==min, got %f", v)
	}
	// min value → 0.
	if v := normalize(0, 0, 100); v != 0.0 {
		t.Errorf("expected 0.0 for min, got %f", v)
	}
	// max value → 1.
	if v := normalize(100, 0, 100); v != 1.0 {
		t.Errorf("expected 1.0 for max, got %f", v)
	}
	// midpoint.
	if v := normalize(50, 0, 100); v != 0.5 {
		t.Errorf("expected 0.5 for midpoint, got %f", v)
	}
}

func TestScoreRepositories_WeightsRespected(t *testing.T) {
	// Custom weights: only stars matter.
	weights := models.Weights{Stars: 1.0, Forks: 0, Watchers: 0, Subscribers: 0, Issues: 0}
	repos := []models.Repository{
		{FullName: "high/stars", Stars: 100, Forks: 0, Watchers: 0, Subscribers: 0, Issues: 1000},
		{FullName: "low/stars", Stars: 10, Forks: 1000, Watchers: 1000, Subscribers: 1000, Issues: 0},
	}
	scored := ScoreRepositories(repos, weights)
	if scored[0].FullName != "high/stars" {
		t.Errorf("with stars-only weight, expected high/stars first, got %s", scored[0].FullName)
	}
}
