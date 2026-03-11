package config

// FullConfig is the top-level configuration mapping org names to their projects.
type FullConfig struct {
	Orgs map[string]OrgProjects `yaml:"orgs"`
}

// OrgProjects holds all project definitions for a single GitHub organization.
type OrgProjects struct {
	Projects []Project `yaml:"projects"`
}

// Project defines the desired state of a single GitHub Project V2.
type Project struct {
	Title       string  `yaml:"title"`
	Description string  `yaml:"description,omitempty"`
	Visibility  string  `yaml:"visibility,omitempty"` // "public" or "private"
	Fields      []Field `yaml:"fields,omitempty"`
	Views       []View  `yaml:"views,omitempty"`
}

// Field defines a custom field on a project.
type Field struct {
	Name    string        `yaml:"name"`
	Type    FieldType     `yaml:"type"`
	Options []FieldOption `yaml:"options,omitempty"` // For single_select and iteration fields
}

// FieldType represents the type of a project field.
type FieldType string

const (
	FieldTypeText         FieldType = "text"
	FieldTypeNumber       FieldType = "number"
	FieldTypeDate         FieldType = "date"
	FieldTypeSingleSelect FieldType = "single_select"
	FieldTypeIteration    FieldType = "iteration"
)

// FieldOption defines an option for a single_select or iteration field.
type FieldOption struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Color       string `yaml:"color,omitempty"`
}

// View defines a project view (board or table layout).
type View struct {
	Name    string     `yaml:"name"`
	Layout  ViewLayout `yaml:"layout"`
	GroupBy string     `yaml:"group_by,omitempty"` // Field name to group by (for board views)
	SortBy  string     `yaml:"sort_by,omitempty"`  // Field name to sort by
	Filter  string     `yaml:"filter,omitempty"`   // Filter expression
	Fields  []string   `yaml:"fields,omitempty"`   // Field names to display (for table views)
}

// ViewLayout represents the layout type of a project view.
type ViewLayout string

const (
	ViewLayoutBoard   ViewLayout = "board"
	ViewLayoutTable   ViewLayout = "table"
	ViewLayoutRoadmap ViewLayout = "roadmap"
)
