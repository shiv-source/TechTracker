package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/shiv-source/TechTracker/internal/models"
	"github.com/shiv-source/TechTracker/utils"
)

// ChunkSize is the number of repos per chunk file for the "All" view.
const ChunkSize = 50

// SaveAllChunks splits repos into chunk files under dir/all/.
// Returns the number of chunks written.
func SaveAllChunks(repos []models.Repository, dir string) (int, error) {
	chunkDir := dir + "/all"
	chunks := (len(repos) + ChunkSize - 1) / ChunkSize
	for i := 0; i < chunks; i++ {
		start := i * ChunkSize
		end := start + ChunkSize
		if end > len(repos) {
			end = len(repos)
		}
		path := fmt.Sprintf("%s/chunk_%d.json", chunkDir, i+1)
		if err := utils.SaveToJsonFile(repos[start:end], path); err != nil {
			return 0, fmt.Errorf("save chunk %d: %w", i+1, err)
		}
	}
	return chunks, nil
}

// SaveGroupFile writes a single group's repos to a group file.
func SaveGroupFile(repos []models.Repository, dir, key string) error {
	groupDir := dir + "/groups"
	path := fmt.Sprintf("%s/%s.json", groupDir, key)
	return utils.SaveToJsonFile(repos, path)
}

// SaveTop5History loads the existing top5_history.json, appends today's top 5,
// and writes it back. The dateKey is the YYYY-MM-DD string.
func SaveTop5History(path, dateKey string, top5 []models.Repository) error {
	// Load existing history (ok if missing — start fresh).
	history := make(models.Top5History)
	if existing, err := utils.LoadJSONFromFile[models.Top5History](path); err == nil && existing != nil {
		history = *existing
	}

	entries := make([]models.Top5Entry, len(top5))
	for i, r := range top5 {
		entries[i] = models.Top5Entry{FullName: r.FullName, Score: r.Score}
	}
	history[dateKey] = entries

	return utils.SaveToJsonFile(history, path)
}

// SaveMetadata writes a metadata.json file.
func SaveMetadata(meta models.RunMetadata, path string) error {
	return utils.SaveToJsonFile(meta, path)
}

// RenderMarkdown generates the readme.md from the template with group tables and trend data.
func RenderMarkdown(groups []models.GroupResult, deltas map[string]models.Delta, templatePath, outputPath string) error {
	tableContent := buildAllTables(groups, deltas)

	data := struct {
		Table       string
		LastUpdated string
	}{
		Table:       tableContent,
		LastUpdated: time.Now().Format("January 02, 2006"),
	}

	return utils.SaveToMarkdown(templatePath, data, outputPath)
}

func buildAllTables(groups []models.GroupResult, deltas map[string]models.Delta) string {
	header := []string{"SL", "Name", "Stars", "Forks", "Issues", "Language", "Trend", "Description", "UpdatedAt"}
	var result strings.Builder

	for _, group := range groups {
		result.WriteString(fmt.Sprintf("## 📋 %s \n\n", group.GroupName))
		result.WriteString(buildTable(header, group.Repositories, deltas))
		result.WriteString("\n\n")
	}

	return result.String()
}

func buildTable(header []string, repos []models.Repository, deltas map[string]models.Delta) string {
	var sb strings.Builder

	// Header row
	sb.WriteString("| " + strings.Join(header, " | ") + " |\n")

	// Separator row
	sep := make([]string, len(header))
	for i := range sep {
		sep[i] = "---"
	}
	sb.WriteString("| " + strings.Join(sep, " | ") + " |\n")

	// Data rows
	for i, repo := range repos {
		row := []string{
			fmt.Sprintf("%d", i+1),
			fmt.Sprintf("[%s](%s)", repo.Name, repo.URL),
			fmt.Sprintf("%d", repo.Stars),
			fmt.Sprintf("%d", repo.Forks),
			fmt.Sprintf("%d", repo.Issues),
			repo.Language,
			formatTrend(repo.FullName, deltas),
			escapeMarkdown(strings.TrimSpace(repo.Description)),
			formatDate(repo.UpdatedAt),
		}
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}

	return sb.String()
}

func formatTrend(fullName string, deltas map[string]models.Delta) string {
	if deltas == nil {
		return ""
	}
	d, ok := deltas[fullName]
	if !ok {
		return ""
	}

	var parts []string

	if d.StarDelta != 0 {
		arrow := "↑"
		if d.StarDelta < 0 {
			arrow = "↓"
		}
		parts = append(parts, fmt.Sprintf("%+d %s", d.StarDelta, arrow))
	}

	if d.RankChange != 0 {
		arrow := "▲"
		if d.RankChange < 0 {
			arrow = "▼"
		}
		parts = append(parts, fmt.Sprintf("%+d %s", d.RankChange, arrow))
	}

	return strings.Join(parts, " ")
}

func formatDate(dateTime string) string {
	if dateTime == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, dateTime)
	if err != nil {
		return dateTime // Return as-is instead of panicking
	}
	return parsed.Format("2006-01-02")
}

func escapeMarkdown(s string) string {
	// Escape pipe characters in description to avoid breaking table layout.
	return strings.ReplaceAll(s, "|", "\\|")
}
