package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Load reads and parses a projects configuration file.
func Load(path string) (*FullConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}
	var cfg FullConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config %s: %w", path, err)
	}
	return &cfg, nil
}

func validate(cfg *FullConfig) error {
	for orgName, org := range cfg.Orgs {
		for i, proj := range org.Projects {
			if proj.Title == "" {
				return fmt.Errorf("org %q project[%d]: title is required", orgName, i)
			}
			if proj.Visibility != "" && proj.Visibility != "public" && proj.Visibility != "private" {
				return fmt.Errorf("org %q project %q: visibility must be \"public\" or \"private\", got %q", orgName, proj.Title, proj.Visibility)
			}
			for j, f := range proj.Fields {
				if f.Name == "" {
					return fmt.Errorf("org %q project %q field[%d]: name is required", orgName, proj.Title, j)
				}
				switch f.Type {
				case FieldTypeText, FieldTypeNumber, FieldTypeDate, FieldTypeSingleSelect, FieldTypeIteration:
				default:
					return fmt.Errorf("org %q project %q field %q: unknown type %q", orgName, proj.Title, f.Name, f.Type)
				}
			}
			for j, v := range proj.Views {
				if v.Name == "" {
					return fmt.Errorf("org %q project %q view[%d]: name is required", orgName, proj.Title, j)
				}
				switch v.Layout {
				case ViewLayoutBoard, ViewLayoutTable, ViewLayoutRoadmap:
				default:
					return fmt.Errorf("org %q project %q view %q: unknown layout %q", orgName, proj.Title, v.Name, v.Layout)
				}
			}
		}
	}
	return nil
}
