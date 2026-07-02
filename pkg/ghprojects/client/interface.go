package client

import "context"

// ProjectsClient defines the interface for GitHub Projects V2 operations.
// This enables mocking for tests and the reconciler.
type ProjectsClient interface {
	GetOrgID(ctx context.Context, login string) (string, error)
	ListProjects(ctx context.Context, org string) ([]ProjectV2, error)
	GetProjectFields(ctx context.Context, projectID string) ([]ProjectV2Field, error)
	GetProjectViews(ctx context.Context, projectID string) ([]ProjectV2View, error)
	CreateProject(ctx context.Context, orgID, title string) (*ProjectV2, error)
	UpdateProject(ctx context.Context, projectID string, title, description *string, public *bool) error
	DeleteProject(ctx context.Context, projectID string) error
	CreateField(ctx context.Context, projectID, name, dataType string, options []ProjectV2FieldOption) (*ProjectV2Field, error)
	UpdateField(ctx context.Context, fieldID, name string) error
	DeleteField(ctx context.Context, fieldID string) error
	CreateView(ctx context.Context, projectID, name, layout string) (*ProjectV2View, error)
	UpdateView(ctx context.Context, viewID string, name, layout *string) error
	DeleteView(ctx context.Context, viewID string) error
}

// Compile-time check that Client implements ProjectsClient.
var _ ProjectsClient = (*Client)(nil)
