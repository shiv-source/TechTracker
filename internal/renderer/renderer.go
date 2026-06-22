package renderer

import (
	"fmt"
	"strings"
	"time"

	"github.com/shiv-source/TechTracker/internal/models"
	"github.com/shiv-source/TechTracker/utils"
)

// SaveGroupJSON writes a group's repository data to a JSON file.
func SaveGroupJSON(repos []models.Repository, path string) error {
	return utils.SaveToJsonFile(repos, path)
}

// SaveAllJSON writes all repositories to a combined JSON file.
func SaveAllJSON(repos []models.Repository, path string) error {
	return utils.SaveToJsonFile(repos, path)
}

// SaveRunMetadata writes the run metadata JSON file.
func SaveRunMetadata(meta models.RunMetadata, path string) error {
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
