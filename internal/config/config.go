package config

import (
	"fmt"
	"os"

	"github.com/shiv-source/TechTracker/internal/models"
	"github.com/shiv-source/TechTracker/utils"
)

// DefaultWeights returns the standard scoring weights.
func DefaultWeights() models.Weights {
	return models.Weights{
		Stars:       0.4,
		Forks:       0.25,
		Watchers:    0.2,
		Subscribers: 0.1,
		Issues:      0.05,
	}
}

// EffectiveWeights returns per-group weights if set, otherwise defaults.
func EffectiveWeights(cfg models.Config) models.Weights {
	if cfg.Weights != nil {
		return *cfg.Weights
	}
	return DefaultWeights()
}

// Load reads and parses the config.json file.
func Load(path string) ([]models.Config, error) {
	configs, err := utils.LoadJSONFromFile[[]models.Config](path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if configs == nil || len(*configs) == 0 {
		return nil, fmt.Errorf("no configurations found in %s", path)
	}
	return *configs, nil
}

// Validate checks that all configured file paths exist and IDs are unique.
func Validate(configs []models.Config) error {
	seenIDs := make(map[int]bool)
	for _, cfg := range configs {
		if cfg.ID == 0 {
			return fmt.Errorf("group %q has invalid id 0", cfg.GroupName)
		}
		if seenIDs[cfg.ID] {
			return fmt.Errorf("duplicate id %d for group %q", cfg.ID, cfg.GroupName)
		}
		seenIDs[cfg.ID] = true

		if cfg.GroupName == "" {
			return fmt.Errorf("group with id %d has empty name", cfg.ID)
		}
		if cfg.FilePath == "" {
			return fmt.Errorf("group %q has empty filePath", cfg.GroupName)
		}
		if _, err := os.Stat(cfg.FilePath); os.IsNotExist(err) {
			return fmt.Errorf("group %q: input file not found: %s", cfg.GroupName, cfg.FilePath)
		}
	}
	return nil
}
