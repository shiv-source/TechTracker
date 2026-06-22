package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
	outputDir          = "data"
	historyDir         = "data/history"
	templateFile       = "template.md"
	outputTemplateFile = "readme.md"
	runMetadataFile    = "data/run-metadata.json"
	version            = "2.0.0"
)

func main() {
	// CLI flags
	groupFilter := flag.String("group", "", "Process only the specified group name")
	dryRun := flag.Bool("dry-run", false, "Print output to stdout without writing files")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	flag.Parse()

	startTime := time.Now()
	accessToken := os.Getenv("GITHUB_TOKEN")

	var runErrors []string

	// Load and validate configuration.
	configs, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

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

			// Load URLs from input file.
			urls, err := utils.LoadUrlsFromTxtFile(cfg.FilePath)
			if err != nil {
				runErrors = append(runErrors, fmt.Sprintf("group %q: %v", cfg.GroupName, err))
				return
			}

			// Build API URLs.
			apiURLs := make([]string, len(urls))
			for i, u := range urls {
				repoFullName := strings.TrimPrefix(u, githubURL)
				apiURLs[i] = fmt.Sprintf("%s/repos/%s", githubBaseAPIURL, repoFullName)
			}

			// Fetch repositories with rate limiting.
			repos, fetchErrs := fetcher.FetchAll(apiURLs, accessToken, 0, *verbose)
			for _, e := range fetchErrs {
				runErrors = append(runErrors, fmt.Sprintf("group %q: %v", cfg.GroupName, e))
			}

			// Score repositories.
			weights := config.EffectiveWeights(cfg)
			repos = scorer.ScoreRepositories(repos, weights)

			// Determine output file path.
			baseFileName := filepath.Base(cfg.FilePath)
			ext := filepath.Ext(baseFileName)
			outputFileName := strings.TrimSuffix(baseFileName, ext)
			outputFilePath := filepath.Join(outputDir, outputFileName+".json")

			groupsChan <- models.GroupResult{
				ID:             cfg.ID,
				GroupName:      cfg.GroupName,
				Repositories:   repos,
				InputFilePath:  cfg.FilePath,
				OutputFilePath: outputFilePath,
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

	// Save per-group JSON files.
	if !*dryRun {
		saveGroupsJSON(groups)
	} else if *verbose {
		for _, g := range groups {
			fmt.Printf("Would save %d repos to %s\n", len(g.Repositories), g.OutputFilePath)
		}
	}

	// Compute global ranking and save all.json.
	allRepos := collectAllRepos(groups)
	defaultWeights := config.DefaultWeights()
	allRepos = scorer.ScoreRepositories(allRepos, defaultWeights)

	if !*dryRun {
		renderer.SaveAllJSON(allRepos, filepath.Join(outputDir, "all.json"))
	} else if *verbose {
		fmt.Printf("Would save %d total repos to data/all.json\n", len(allRepos))
	}

	// Historical snapshots and deltas.
	var deltas map[string]models.Delta
	if !*dryRun {
		today := time.Now()
		_ = history.SaveSnapshot(historyDir, today, allRepos)

		prevRepos, _, err := history.LatestSnapshot(historyDir)
		if err == nil && len(prevRepos) > 0 {
			deltas = history.ComputeDeltas(allRepos, prevRepos, defaultWeights)
			if *verbose {
				fmt.Printf("Computed deltas against previous snapshot (%d repos)\n", len(prevRepos))
			}
		}

		// Prune snapshots older than 90 days.
		_ = history.PruneOldSnapshots(historyDir, 90)
	}

	// Generate markdown.
	if !*dryRun {
		renderer.RenderMarkdown(groups, deltas, templateFile, outputTemplateFile)
	} else {
		renderer.RenderMarkdown(groups, deltas, templateFile, outputTemplateFile)
		if *verbose {
			fmt.Printf("Would write markdown to %s\n", outputTemplateFile)
		}
	}

	// Save run metadata.
	meta := models.RunMetadata{
		Version:    version,
		StartTime:  startTime.Format(time.RFC3339),
		EndTime:    time.Now().Format(time.RFC3339),
		DurationMs: time.Since(startTime).Milliseconds(),
		Groups:     make(map[string]int),
		Errors:     runErrors,
		DryRun:     *dryRun,
	}
	for _, g := range groups {
		meta.Groups[g.GroupName] = len(g.Repositories)
	}

	if !*dryRun {
		renderer.SaveRunMetadata(meta, runMetadataFile)
	} else if *verbose {
		fmt.Printf("Run metadata: %+v\n", meta)
	}

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

func saveGroupsJSON(groups []models.GroupResult) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		return
	}
	var wg sync.WaitGroup
	for _, group := range groups {
		wg.Add(1)
		go func(g models.GroupResult) {
			defer wg.Done()
			if err := renderer.SaveGroupJSON(g.Repositories, g.OutputFilePath); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving %s: %v\n", g.OutputFilePath, err)
			}
		}(group)
	}
	wg.Wait()
	fmt.Println("All files saved.")
}

func collectAllRepos(groups []models.GroupResult) []models.Repository {
	var all []models.Repository
	for _, g := range groups {
		all = append(all, g.Repositories...)
	}
	return all
}
