package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shiv-source/TechTracker/internal/config"
	"github.com/shiv-source/TechTracker/internal/fetcher"
	"github.com/shiv-source/TechTracker/internal/history"
	"github.com/shiv-source/TechTracker/internal/models"
	"github.com/shiv-source/TechTracker/internal/renderer"
	"github.com/shiv-source/TechTracker/internal/scorer"
	"github.com/shiv-source/TechTracker/utils"
)

const (
	configFile         = "config.json"
	githubURL          = "https://github.com/"
	githubBaseAPIURL   = "https://api.github.com"
	dataDir            = "data"
	top5HistoryFile    = "data/top5_history.json"
	metadataFile       = "data/metadata.json"
	templateFile       = "template.md"
	outputTemplateFile = "readme.md"
	version            = "2.0.0"
)

func main() {
	groupFilter := flag.String("group", "", "Process only the specified group name")
	dryRun := flag.Bool("dry-run", false, "Print output to stdout without writing files")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	startTime := time.Now()
	accessToken := os.Getenv("GITHUB_TOKEN")
	today := time.Now()
	todayStr := today.Format("2006-01-02")
	todayDir := filepath.Join(dataDir, todayStr)

	var runErrors []string

	// Load and validate configuration.
	appConfig, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	configs := appConfig.Groups

	if err := config.Validate(configs); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
		os.Exit(1)
	}

	// Filter groups if --group is specified.
	if *groupFilter != "" {
		configs = filterGroups(configs, *groupFilter)
		if len(configs) == 0 {
			fmt.Fprintf(os.Stderr, "No group matching %q found\n", *groupFilter)
			os.Exit(1)
		}
		if *verbose {
			fmt.Printf("Filtered to group: %s\n", *groupFilter)
		}
	}

	// Fetch and score all groups concurrently.
	var (
		wg         sync.WaitGroup
		groupsChan = make(chan models.GroupResult, len(configs))
	)

	for _, cfg := range configs {
		wg.Add(1)
		go func(cfg models.Config) {
			defer wg.Done()

			urls, err := utils.LoadUrlsFromTxtFile(cfg.FilePath)
			if err != nil {
				runErrors = append(runErrors, fmt.Sprintf("group %q: %v", cfg.GroupName, err))
				return
			}

			apiURLs := make([]string, len(urls))
			for i, u := range urls {
				repoFullName := strings.TrimPrefix(u, githubURL)
				apiURLs[i] = fmt.Sprintf("%s/repos/%s", githubBaseAPIURL, repoFullName)
			}

			repos, fetchErrs := fetcher.FetchAll(apiURLs, accessToken, 0, *verbose)
			for _, e := range fetchErrs {
				runErrors = append(runErrors, fmt.Sprintf("group %q: %v", cfg.GroupName, e))
			}

			weights := config.EffectiveWeights(cfg)
			repos = scorer.ScoreRepositories(repos, weights)

			baseFileName := strings.TrimSuffix(cfg.FilePath, ".txt")
			baseFileName = strings.TrimPrefix(baseFileName, "projects/")
			groupKey := strings.ReplaceAll(baseFileName, "/", "_")

			groupsChan <- models.GroupResult{
				ID:            cfg.ID,
				GroupName:     cfg.GroupName,
				GroupKey:      groupKey,
				Repositories:  repos,
				InputFilePath: cfg.FilePath,
			}
		}(cfg)
	}

	wg.Wait()
	close(groupsChan)

	var groups []models.GroupResult
	for g := range groupsChan {
		groups = append(groups, g)
	}

	history.SortGroupsByID(groups)

	if *dryRun {
		fmt.Println("=== DRY RUN ===")
	}

	// Compute global ranking.
	allRepos := collectAllRepos(groups)
	defaultWeights := config.DefaultWeights()
	allRepos = scorer.ScoreRepositories(allRepos, defaultWeights)

	// Snapshot + deltas + prune.
	var deltas map[string]models.Delta
	if !*dryRun {
		// Save full snapshot for future trend computation.
		_ = history.SaveSnapshot(dataDir, today, allRepos)

		prevRepos, _, err := history.LatestSnapshot(dataDir)
		if err == nil && len(prevRepos) > 0 {
			deltas = history.ComputeDeltas(allRepos, prevRepos, defaultWeights)
			if *verbose {
				fmt.Printf("Computed deltas against previous snapshot (%d repos)\n", len(prevRepos))
			}
		}

		// Apply deltas to repos for embedding in chunk/group files.
		applyDeltas(allRepos, deltas)
		for i := range groups {
			applyDeltas(groups[i].Repositories, deltas)
		}

		// Prune old date directories.
		_ = history.PruneOldSnapshots(dataDir, appConfig.RetentionDays)
	}

	// Build categories metadata.
	categoryMeta := buildCategoryMeta(groups)
	summary := buildSummary(allRepos, len(groups))

	// Build history info.
	historyInfo := buildHistoryInfo(dataDir)

	// Save all output files.
	if !*dryRun {
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating data dir: %v\n", err)
		} else {
			// Chunked all.json files.
			chunkCount, err := renderer.SaveAllChunks(allRepos, todayDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error saving chunks: %v\n", err)
			} else if *verbose {
				fmt.Printf("Saved %d chunks to %s/all/\n", chunkCount, todayDir)
			}

			// Per-group files.
			for _, g := range groups {
				if err := renderer.SaveGroupFile(g.Repositories, todayDir, g.GroupKey); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving group %s: %v\n", g.GroupKey, err)
				}
			}

			// Root metadata.json (catalog).
			meta := models.RunMetadata{
				Version:    version,
				LatestDate: todayStr,
				StartTime:  startTime.Format(time.RFC3339),
				EndTime:    time.Now().Format(time.RFC3339),
				DurationMs: time.Since(startTime).Milliseconds(),
				Summary:    summary,
				Categories: categoryMeta,
				AllChunks:  chunkCount,
				History:    historyInfo,
				Errors:     runErrors,
				DryRun:     *dryRun,
			}
			if err := renderer.SaveMetadata(meta, metadataFile); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving metadata: %v\n", err)
			} else {
				fmt.Println("Saved data/metadata.json")
			}

			// top5_history.json (append today's top 5).
			top5 := getTopN(allRepos, 5)
			if err := renderer.SaveTop5History(top5HistoryFile, todayStr, top5); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving top5 history: %v\n", err)
			} else if *verbose {
				fmt.Println("Updated data/top5_history.json")
			}
		}
	} else if *verbose {
		fmt.Printf("Would save %d groups, %d repos, %d chunks to %s/\n", len(groups), len(allRepos), (len(allRepos)+49)/50, todayDir)
	}

	// Generate markdown.
	renderer.RenderMarkdown(groups, deltas, templateFile, outputTemplateFile)

	if *verbose || *dryRun {
		fmt.Printf("Total execution time: %.3f seconds\n", time.Since(startTime).Seconds())
	}

	if len(runErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Completed with %d error(s)\n", len(runErrors))
	}
}

func filterGroups(configs []models.Config, name string) []models.Config {
	var filtered []models.Config
	for _, cfg := range configs {
		if strings.EqualFold(cfg.GroupName, name) {
			filtered = append(filtered, cfg)
		}
	}
	return filtered
}

func collectAllRepos(groups []models.GroupResult) []models.Repository {
	var all []models.Repository
	for _, g := range groups {
		all = append(all, g.Repositories...)
	}
	return all
}

func buildCategoryMeta(groups []models.GroupResult) []models.CategoryMeta {
	meta := make([]models.CategoryMeta, len(groups))
	for i, g := range groups {
		meta[i] = models.CategoryMeta{
			Key:   g.GroupKey,
			Label: g.GroupName,
			Count: len(g.Repositories),
		}
	}
	return meta
}

func buildSummary(allRepos []models.Repository, categoryCount int) models.Summary {
	s := models.Summary{
		TotalRepos: len(allRepos),
		Categories: categoryCount,
	}
	if len(allRepos) > 0 {
		// allRepos is already sorted by score descending.
		s.TopRepo = allRepos[0].FullName
		s.TopStars = allRepos[0].Stars
	}
	return s
}

func buildHistoryInfo(dataDir string) models.HistoryInfo {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return models.HistoryInfo{}
	}

	var dates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Check if snapshot.json exists inside.
		snapPath := filepath.Join(dataDir, entry.Name(), "snapshot.json")
		if _, err := os.Stat(snapPath); os.IsNotExist(err) {
			continue
		}
		if _, err := time.Parse("2006-01-02", entry.Name()); err == nil {
			dates = append(dates, entry.Name())
		}
	}

	sort.Strings(dates)

	info := models.HistoryInfo{
		AvailableDates: dates,
		Count:          len(dates),
	}
	if len(dates) > 0 {
		info.FirstDate = dates[0]
		info.LastDate = dates[len(dates)-1]
	}
	return info
}

func applyDeltas(repos []models.Repository, deltas map[string]models.Delta) {
	if deltas == nil {
		return
	}
	for i := range repos {
		if d, ok := deltas[repos[i].FullName]; ok {
			repos[i].RankChange = d.RankChange
			repos[i].StarDelta = d.StarDelta
			repos[i].PrevScore = d.PrevScore
		}
	}
}

func getTopN(repos []models.Repository, n int) []models.Repository {
	if len(repos) <= n {
		return repos
	}
	return repos[:n]
}
