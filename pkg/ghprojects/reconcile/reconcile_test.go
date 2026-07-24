package reconcile

import (
	"context"
	"testing"

	"kubevirt.io/project-infra/pkg/ghprojects/client"
	"kubevirt.io/project-infra/pkg/ghprojects/config"
)

type fakeClient struct {
	orgID    string
	projects []client.ProjectV2
	fields   []client.ProjectV2Field
	views    []client.ProjectV2View

	createdProjects []string
	updatedProjects []string
	createdFields   []string
	updatedFields   []string
	deletedFields   []string
	createdViews    []string
	updatedViews    []string
	updatedLayouts  []string
	deletedViews    []string
}

func (f *fakeClient) GetOrgID(_ context.Context, _ string) (string, error) {
	return f.orgID, nil
}

func (f *fakeClient) ListProjects(_ context.Context, _ string) ([]client.ProjectV2, error) {
	return f.projects, nil
}

func (f *fakeClient) GetProjectFields(_ context.Context, _ string) ([]client.ProjectV2Field, error) {
	return f.fields, nil
}

func (f *fakeClient) GetProjectViews(_ context.Context, _ string) ([]client.ProjectV2View, error) {
	return f.views, nil
}

func (f *fakeClient) CreateProject(_ context.Context, _, title string) (*client.ProjectV2, error) {
	f.createdProjects = append(f.createdProjects, title)
	return &client.ProjectV2{ID: "new-id", Number: 1, Title: title}, nil
}

func (f *fakeClient) UpdateProject(_ context.Context, projectID string, title, _ *string, _ *bool) error {
	t := projectID
	if title != nil {
		t = *title
	}
	f.updatedProjects = append(f.updatedProjects, t)
	return nil
}

func (f *fakeClient) DeleteProject(_ context.Context, _ string) error {
	return nil
}

func (f *fakeClient) CreateField(_ context.Context, _, name, dataType string, _ []client.ProjectV2FieldOption) (*client.ProjectV2Field, error) {
	f.createdFields = append(f.createdFields, name)
	return &client.ProjectV2Field{ID: "field-new", Name: name, Type: dataType}, nil
}

func (f *fakeClient) UpdateField(_ context.Context, fieldID, name string) error {
	f.updatedFields = append(f.updatedFields, name)
	return nil
}

func (f *fakeClient) DeleteField(_ context.Context, fieldID string) error {
	f.deletedFields = append(f.deletedFields, fieldID)
	return nil
}

func (f *fakeClient) CreateView(_ context.Context, _, name, layout string) (*client.ProjectV2View, error) {
	f.createdViews = append(f.createdViews, name)
	return &client.ProjectV2View{ID: "view-new", Name: name, Layout: layout}, nil
}

func (f *fakeClient) UpdateView(_ context.Context, viewID string, name, layout *string) error {
	n := viewID
	if name != nil {
		n = *name
	}
	f.updatedViews = append(f.updatedViews, n)
	if layout != nil {
		f.updatedLayouts = append(f.updatedLayouts, *layout)
	}
	return nil
}

func (f *fakeClient) DeleteView(_ context.Context, viewID string) error {
	f.deletedViews = append(f.deletedViews, viewID)
	return nil
}

func TestReconcile_DryRun(t *testing.T) {
	fc := &fakeClient{orgID: "org-123"}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{Title: "My Board", Visibility: "public"},
				},
			},
		},
	}

	opts := Options{Confirm: false, FixProjects: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdProjects) != 0 {
		t.Error("expected no projects created in dry-run mode")
	}
}

func TestReconcile_CreateProject(t *testing.T) {
	fc := &fakeClient{orgID: "org-123"}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{Title: "My Board", Visibility: "public"},
				},
			},
		},
	}

	opts := Options{Confirm: true, FixProjects: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdProjects) != 1 || fc.createdProjects[0] != "My Board" {
		t.Errorf("expected 1 project created with title 'My Board', got %v", fc.createdProjects)
	}
}

func TestReconcile_UpdateProject(t *testing.T) {
	fc := &fakeClient{
		orgID: "org-123",
		projects: []client.ProjectV2{
			{ID: "proj-1", Title: "My Board", ShortDescription: "old", Public: false},
		},
	}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{Title: "My Board", Description: "new desc", Visibility: "public"},
				},
			},
		},
	}

	opts := Options{Confirm: true, FixProjects: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdProjects) != 0 {
		t.Error("expected no projects created")
	}
	if len(fc.updatedProjects) != 1 {
		t.Errorf("expected 1 project updated, got %d", len(fc.updatedProjects))
	}
}

func TestReconcile_NoOpWhenInSync(t *testing.T) {
	fc := &fakeClient{
		orgID: "org-123",
		projects: []client.ProjectV2{
			{ID: "proj-1", Title: "My Board", ShortDescription: "desc", Public: true},
		},
	}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{Title: "My Board", Description: "desc", Visibility: "public"},
				},
			},
		},
	}

	opts := Options{Confirm: true, FixProjects: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdProjects) != 0 {
		t.Error("expected no projects created")
	}
	if len(fc.updatedProjects) != 0 {
		t.Error("expected no projects updated")
	}
}

func TestReconcile_CreateField(t *testing.T) {
	fc := &fakeClient{
		orgID: "org-123",
		projects: []client.ProjectV2{
			{ID: "proj-1", Title: "My Board", Public: true},
		},
	}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{
						Title:      "My Board",
						Visibility: "public",
						Fields: []config.Field{
							{Name: "Status", Type: config.FieldTypeSingleSelect, Options: []config.FieldOption{
								{Name: "Todo", Color: "GRAY"},
								{Name: "Done", Color: "GREEN"},
							}},
						},
					},
				},
			},
		},
	}

	opts := Options{Confirm: true, FixProjects: true, FixFields: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdFields) != 1 || fc.createdFields[0] != "Status" {
		t.Errorf("expected field 'Status' created, got %v", fc.createdFields)
	}
}

func TestReconcile_FieldAlreadyExists(t *testing.T) {
	fc := &fakeClient{
		orgID: "org-123",
		projects: []client.ProjectV2{
			{ID: "proj-1", Title: "My Board", Public: true},
		},
		fields: []client.ProjectV2Field{
			{ID: "f-1", Name: "Status", Type: "SINGLE_SELECT"},
		},
	}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{
						Title:      "My Board",
						Visibility: "public",
						Fields: []config.Field{
							{Name: "Status", Type: config.FieldTypeSingleSelect},
						},
					},
				},
			},
		},
	}

	opts := Options{Confirm: true, FixProjects: true, FixFields: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdFields) != 0 {
		t.Error("expected no fields created when field already exists")
	}
}

func TestReconcile_CreateView(t *testing.T) {
	fc := &fakeClient{
		orgID: "org-123",
		projects: []client.ProjectV2{
			{ID: "proj-1", Title: "My Board", Public: true},
		},
	}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{
						Title:      "My Board",
						Visibility: "public",
						Views: []config.View{
							{Name: "Kanban", Layout: config.ViewLayoutBoard},
						},
					},
				},
			},
		},
	}

	opts := Options{Confirm: true, FixProjects: true, FixViews: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdViews) != 1 || fc.createdViews[0] != "Kanban" {
		t.Errorf("expected view 'Kanban' created, got %v", fc.createdViews)
	}
}

func TestReconcile_UpdateViewLayout(t *testing.T) {
	fc := &fakeClient{
		orgID: "org-123",
		projects: []client.ProjectV2{
			{ID: "proj-1", Title: "My Board", Public: true},
		},
		views: []client.ProjectV2View{
			{ID: "v-1", Name: "Kanban", Layout: "TABLE_LAYOUT"},
		},
	}
	cfg := &config.FullConfig{
		Orgs: map[string]config.OrgProjects{
			"test-org": {
				Projects: []config.Project{
					{
						Title:      "My Board",
						Visibility: "public",
						Views: []config.View{
							{Name: "Kanban", Layout: config.ViewLayoutBoard},
						},
					},
				},
			},
		},
	}

	opts := Options{Confirm: true, FixProjects: true, FixViews: true}
	if err := Reconcile(context.Background(), fc, cfg, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fc.createdViews) != 0 {
		t.Error("expected no views created")
	}
	if len(fc.updatedViews) != 1 {
		t.Errorf("expected 1 view updated, got %d", len(fc.updatedViews))
	}
	if len(fc.updatedLayouts) != 1 || fc.updatedLayouts[0] != "BOARD_LAYOUT" {
		t.Errorf("expected layout update to BOARD_LAYOUT, got %v", fc.updatedLayouts)
	}
}
