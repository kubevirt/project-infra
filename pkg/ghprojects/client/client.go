package client

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub GraphQL API for Projects V2 operations.
type Client struct {
	gql *githubv4.Client
}

// New creates a new Client using the provided OAuth2 token.
func New(token string) *Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	return &Client{gql: githubv4.NewClient(httpClient)}
}

// GetOrgID returns the node ID for a GitHub organization.
func (c *Client) GetOrgID(ctx context.Context, login string) (string, error) {
	var q queryOrgID
	vars := map[string]interface{}{
		"login": githubv4.String(login),
	}
	if err := c.gql.Query(ctx, &q, vars); err != nil {
		return "", fmt.Errorf("querying org ID for %s: %w", login, err)
	}
	id, ok := q.Organization.ID.(string)
	if !ok {
		return "", fmt.Errorf("unexpected org ID type for %s: %T", login, q.Organization.ID)
	}
	return id, nil
}

// ListProjects returns all Projects V2 for an organization.
func (c *Client) ListProjects(ctx context.Context, org string) ([]ProjectV2, error) {
	var projects []ProjectV2
	var cursor *githubv4.String

	for {
		var q queryOrgProjects
		vars := map[string]interface{}{
			"login": githubv4.String(org),
			"first": githubv4.Int(50),
			"after": cursor,
		}
		if err := c.gql.Query(ctx, &q, vars); err != nil {
			return nil, fmt.Errorf("listing projects for %s: %w", org, err)
		}

		for _, n := range q.Organization.ProjectsV2.Nodes {
			projects = append(projects, ProjectV2{
				ID:               string(n.ID),
				Number:           int(n.Number),
				Title:            string(n.Title),
				ShortDescription: string(n.ShortDescription),
				Public:           bool(n.Public),
			})
		}

		if !bool(q.Organization.ProjectsV2.PageInfo.HasNextPage) {
			break
		}
		cursor = &q.Organization.ProjectsV2.PageInfo.EndCursor
	}
	return projects, nil
}

// GetProjectFields returns all fields for a project.
func (c *Client) GetProjectFields(ctx context.Context, projectID string) ([]ProjectV2Field, error) {
	var fields []ProjectV2Field
	var cursor *githubv4.String

	for {
		var q queryProjectFields
		vars := map[string]interface{}{
			"projectID": githubv4.ID(projectID),
			"first":     githubv4.Int(50),
			"after":     cursor,
		}
		if err := c.gql.Query(ctx, &q, vars); err != nil {
			return nil, fmt.Errorf("listing fields for project %s: %w", projectID, err)
		}

		for _, n := range q.Node.ProjectV2.Fields.Nodes {
			f := ProjectV2Field{}
			switch n.TypeName {
			case "ProjectV2SingleSelectField":
				f.ID = string(n.ProjectV2SingleSelectField.ID)
				f.Name = string(n.ProjectV2SingleSelectField.Name)
				f.Type = string(n.ProjectV2SingleSelectField.DataType)
				for _, o := range n.ProjectV2SingleSelectField.Options {
					f.Options = append(f.Options, ProjectV2FieldOption{
						ID:          string(o.ID),
						Name:        string(o.Name),
						Description: string(o.Description),
						Color:       string(o.Color),
					})
				}
			case "ProjectV2IterationField":
				f.ID = string(n.ProjectV2IterationField.ID)
				f.Name = string(n.ProjectV2IterationField.Name)
				f.Type = string(n.ProjectV2IterationField.DataType)
			default:
				f.ID = string(n.ProjectV2FieldCommon.ID)
				f.Name = string(n.ProjectV2FieldCommon.Name)
				f.Type = string(n.ProjectV2FieldCommon.DataType)
			}
			fields = append(fields, f)
		}

		if !bool(q.Node.ProjectV2.Fields.PageInfo.HasNextPage) {
			break
		}
		cursor = &q.Node.ProjectV2.Fields.PageInfo.EndCursor
	}
	return fields, nil
}

// GetProjectViews returns all views for a project.
func (c *Client) GetProjectViews(ctx context.Context, projectID string) ([]ProjectV2View, error) {
	var views []ProjectV2View
	var cursor *githubv4.String

	for {
		var q queryProjectViews
		vars := map[string]interface{}{
			"projectID": githubv4.ID(projectID),
			"first":     githubv4.Int(50),
			"after":     cursor,
		}
		if err := c.gql.Query(ctx, &q, vars); err != nil {
			return nil, fmt.Errorf("listing views for project %s: %w", projectID, err)
		}

		for _, n := range q.Node.ProjectV2.Views.Nodes {
			views = append(views, ProjectV2View{
				ID:     string(n.ID),
				Name:   string(n.Name),
				Layout: string(n.Layout),
			})
		}

		if !bool(q.Node.ProjectV2.Views.PageInfo.HasNextPage) {
			break
		}
		cursor = &q.Node.ProjectV2.Views.PageInfo.EndCursor
	}
	return views, nil
}

// CreateProject creates a new Project V2 in an organization.
func (c *Client) CreateProject(ctx context.Context, orgID, title string) (*ProjectV2, error) {
	var m mutationCreateProject
	input := createProjectV2Input{
		OwnerID: githubv4.ID(orgID),
		Title:   githubv4.String(title),
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return nil, fmt.Errorf("creating project %q: %w", title, err)
	}
	return &ProjectV2{
		ID:     string(m.CreateProjectV2.ProjectV2.ID),
		Number: int(m.CreateProjectV2.ProjectV2.Number),
		Title:  title,
	}, nil
}

// UpdateProject updates a project's metadata.
func (c *Client) UpdateProject(ctx context.Context, projectID string, title, description *string, public *bool) error {
	var m mutationUpdateProject
	input := updateProjectV2Input{
		ProjectID: githubv4.ID(projectID),
	}
	if title != nil {
		t := githubv4.String(*title)
		input.Title = &t
	}
	if description != nil {
		d := githubv4.String(*description)
		input.ShortDescription = &d
	}
	if public != nil {
		p := githubv4.Boolean(*public)
		input.Public = &p
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("updating project %s: %w", projectID, err)
	}
	return nil
}

// DeleteProject deletes a project.
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	var m mutationDeleteProject
	input := deleteProjectV2Input{
		ProjectID: githubv4.ID(projectID),
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("deleting project %s: %w", projectID, err)
	}
	return nil
}

// CreateField creates a custom field on a project.
func (c *Client) CreateField(ctx context.Context, projectID, name, dataType string, options []ProjectV2FieldOption) (*ProjectV2Field, error) {
	var m mutationCreateField
	input := createProjectV2FieldInput{
		ProjectID: githubv4.ID(projectID),
		DataType:  githubv4.String(dataType),
		Name:      githubv4.String(name),
	}
	for _, o := range options {
		input.SingleSelectOptions = append(input.SingleSelectOptions, createProjectV2SingleSelectOption{
			Name:        githubv4.String(o.Name),
			Description: githubv4.String(o.Description),
			Color:       githubv4.String(o.Color),
		})
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return nil, fmt.Errorf("creating field %q on project %s: %w", name, projectID, err)
	}
	result := &ProjectV2Field{
		Name: name,
		Type: dataType,
	}
	// Extract ID from whichever fragment matched.
	if id := string(m.CreateProjectV2Field.ProjectV2Field.ProjectV2SingleSelectField.ID); id != "" {
		result.ID = id
	} else {
		result.ID = string(m.CreateProjectV2Field.ProjectV2Field.ProjectV2FieldCommon.ID)
	}
	return result, nil
}

// UpdateField updates a field's name.
func (c *Client) UpdateField(ctx context.Context, fieldID, name string) error {
	var m mutationUpdateField
	n := githubv4.String(name)
	input := updateProjectV2FieldInput{
		FieldID: githubv4.ID(fieldID),
		Name:    &n,
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("updating field %s: %w", fieldID, err)
	}
	return nil
}

// DeleteField deletes a custom field from a project.
func (c *Client) DeleteField(ctx context.Context, fieldID string) error {
	var m mutationDeleteField
	input := deleteProjectV2FieldInput{
		FieldID: githubv4.ID(fieldID),
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("deleting field %s: %w", fieldID, err)
	}
	return nil
}

// CreateView creates a view on a project.
func (c *Client) CreateView(ctx context.Context, projectID, name, layout string) (*ProjectV2View, error) {
	var m mutationCreateView
	input := createProjectV2ViewInput{
		ProjectID: githubv4.ID(projectID),
		Name:      githubv4.String(name),
		Layout:    githubv4.String(layout),
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return nil, fmt.Errorf("creating view %q on project %s: %w", name, projectID, err)
	}
	return &ProjectV2View{
		ID:     string(m.CreateProjectV2View.ProjectV2View.ID),
		Name:   string(m.CreateProjectV2View.ProjectV2View.Name),
		Layout: string(m.CreateProjectV2View.ProjectV2View.Layout),
	}, nil
}

// UpdateView updates a view's name and/or layout.
func (c *Client) UpdateView(ctx context.Context, viewID string, name, layout *string) error {
	var m mutationUpdateView
	input := updateProjectV2ViewInput{
		ViewID: githubv4.ID(viewID),
	}
	if name != nil {
		n := githubv4.String(*name)
		input.Name = &n
	}
	if layout != nil {
		l := githubv4.String(*layout)
		input.Layout = &l
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("updating view %s: %w", viewID, err)
	}
	return nil
}

// DeleteView deletes a view from a project.
func (c *Client) DeleteView(ctx context.Context, viewID string) error {
	var m mutationDeleteView
	input := deleteProjectV2ViewInput{
		ViewID: githubv4.ID(viewID),
	}
	if err := c.gql.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("deleting view %s: %w", viewID, err)
	}
	return nil
}
