package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	content := `
orgs:
  kubevirt:
    projects:
      - title: "CI Board"
        description: "Tracking CI improvements"
        visibility: public
        fields:
          - name: Status
            type: single_select
            options:
              - name: Todo
                color: GRAY
              - name: In Progress
                color: YELLOW
              - name: Done
                color: GREEN
          - name: Priority
            type: single_select
            options:
              - name: P0
                color: RED
              - name: P1
                color: ORANGE
        views:
          - name: Board
            layout: board
            group_by: Status
          - name: Table
            layout: table
            fields: [Title, Status, Priority]
`
	dir := t.TempDir()
	path := filepath.Join(dir, "projects.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	org, ok := cfg.Orgs["kubevirt"]
	if !ok {
		t.Fatal("expected org kubevirt")
	}
	if len(org.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(org.Projects))
	}
	proj := org.Projects[0]
	if proj.Title != "CI Board" {
		t.Errorf("expected title 'CI Board', got %q", proj.Title)
	}
	if len(proj.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(proj.Fields))
	}
	if len(proj.Views) != 2 {
		t.Errorf("expected 2 views, got %d", len(proj.Views))
	}
}

func TestLoad_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name: "missing title",
			content: `
orgs:
  test:
    projects:
      - description: "no title"
`,
		},
		{
			name: "invalid visibility",
			content: `
orgs:
  test:
    projects:
      - title: "test"
        visibility: "internal"
`,
		},
		{
			name: "invalid field type",
			content: `
orgs:
  test:
    projects:
      - title: "test"
        fields:
          - name: F1
            type: unknown
`,
		},
		{
			name: "invalid view layout",
			content: `
orgs:
  test:
    projects:
      - title: "test"
        views:
          - name: V1
            layout: kanban
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "projects.yaml")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(path)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
