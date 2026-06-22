package scorer

import (
	"math"
	"sort"

	"github.com/shiv-source/TechTracker/internal/models"
)

// ScoreRepositories normalizes metrics using min-max scaling within the group,
// applies weighted scoring, sorts descending by score, and returns the result.
func ScoreRepositories(repositories []models.Repository, weights models.Weights) []models.Repository {
	if len(repositories) == 0 {
		return repositories
	}

	// Find min/max for each metric.
	ranges := computeRanges(repositories)

	// Score each repository.
	for i := range repositories {
		normalizedStars := normalize(repositories[i].Stars, ranges.starsMin, ranges.starsMax)
		normalizedForks := normalize(repositories[i].Forks, ranges.forksMin, ranges.forksMax)
		normalizedWatchers := normalize(repositories[i].Watchers, ranges.watchersMin, ranges.watchersMax)
		normalizedSubscribers := normalize(repositories[i].Subscribers, ranges.subsMin, ranges.subsMax)
		normalizedIssues := normalize(repositories[i].Issues, ranges.issuesMin, ranges.issuesMax)

		repositories[i].Score = math.Round(
			(normalizedStars*weights.Stars+
				normalizedForks*weights.Forks+
				normalizedWatchers*weights.Watchers+
				normalizedSubscribers*weights.Subscribers+
				normalizedIssues*weights.Issues)*100000,
		) / 100000.0
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].Score > repositories[j].Score
	})

	return repositories
}

type metricRanges struct {
	starsMin, starsMax     int
	forksMin, forksMax     int
	watchersMin, watchersMax int
	subsMin, subsMax       int
	issuesMin, issuesMax   int
}

func computeRanges(repos []models.Repository) metricRanges {
	r := metricRanges{
		starsMin:    math.MaxInt,
		forksMin:    math.MaxInt,
		watchersMin: math.MaxInt,
		subsMin:     math.MaxInt,
		issuesMin:   math.MaxInt,
	}
	for _, repo := range repos {
		r.starsMin = min(r.starsMin, repo.Stars)
		r.starsMax = max(r.starsMax, repo.Stars)
		r.forksMin = min(r.forksMin, repo.Forks)
		r.forksMax = max(r.forksMax, repo.Forks)
		r.watchersMin = min(r.watchersMin, repo.Watchers)
		r.watchersMax = max(r.watchersMax, repo.Watchers)
		r.subsMin = min(r.subsMin, repo.Subscribers)
		r.subsMax = max(r.subsMax, repo.Subscribers)
		r.issuesMin = min(r.issuesMin, repo.Issues)
		r.issuesMax = max(r.issuesMax, repo.Issues)
	}
	return r
}

// normalize scales value to [0,1]. When all repos have the same value (max==min),
// returns 0.5 (neutral midpoint) to avoid division by zero.
func normalize(value, minVal, maxVal int) float64 {
	if maxVal == minVal {
		return 0.5
	}
	return float64(value-minVal) / float64(maxVal-minVal)
}
