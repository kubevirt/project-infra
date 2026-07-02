package reconcile

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"kubevirt.io/project-infra/pkg/ghprojects/client"
	"kubevirt.io/project-infra/pkg/ghprojects/config"
)

// Options controls reconciliation behavior.
type Options struct {
	Confirm     bool // If false, dry-run only (log what would happen).
	FixProjects bool // Create/update projects.
	FixFields   bool // Create missing custom fields.
	FixViews    bool // Create/update views.
}

// Reconcile compares the desired config against the actual GitHub state
// and makes changes to converge them.
func Reconcile(ctx context.Context, c client.ProjectsClient, cfg *config.FullConfig, opts Options) error {
	for orgName, org := range cfg.Orgs {
		if err := reconcileOrg(ctx, c, orgName, org, opts); err != nil {
			return fmt.Errorf("org %s: %w", orgName, err)
		}
	}
	return nil
}

func reconcileOrg(ctx context.Context, c client.ProjectsClient, orgName string, org config.OrgProjects, opts Options) error {
	orgID, err := c.GetOrgID(ctx, orgName)
	if err != nil {
		return err
	}

	existing, err := c.ListProjects(ctx, orgName)
	if err != nil {
		return err
	}

	existingByTitle := make(map[string]*client.ProjectV2, len(existing))
	for i := range existing {
		existingByTitle[existing[i].Title] = &existing[i]
	}

	for _, desired := range org.Projects {
		actual, found := existingByTitle[desired.Title]
		if !found {
			if err := createProject(ctx, c, orgID, desired, opts); err != nil {
				return err
			}
			continue
		}

		if err := updateProject(ctx, c, actual, desired, opts); err != nil {
			return err
		}
	}

	return nil
}

func createProject(ctx context.Context, c client.ProjectsClient, orgID string, desired config.Project, opts Options) error {
	logrus.Infof("project %q: would create", desired.Title)

	if !opts.Confirm || !opts.FixProjects {
		return nil
	}

	proj, err := c.CreateProject(ctx, orgID, desired.Title)
	if err != nil {
		return err
	}
	logrus.Infof("project %q: created with ID %s", desired.Title, proj.ID)

	// Set description and visibility if specified.
	if desired.Description != "" || desired.Visibility != "" {
		var desc *string
		if desired.Description != "" {
			desc = &desired.Description
		}
		public := visibilityToPublic(desired.Visibility)
		if err := c.UpdateProject(ctx, proj.ID, nil, desc, public); err != nil {
			return fmt.Errorf("setting metadata for new project %q: %w", desired.Title, err)
		}
	}

	if opts.FixFields {
		if err := reconcileFields(ctx, c, proj.ID, desired.Title, nil, desired.Fields, opts); err != nil {
			return err
		}
	}
	if opts.FixViews {
		if err := reconcileViews(ctx, c, proj.ID, desired.Title, nil, desired.Views, opts); err != nil {
			return err
		}
	}

	return nil
}

func updateProject(ctx context.Context, c client.ProjectsClient, actual *client.ProjectV2, desired config.Project, opts Options) error {
	var needsUpdate bool
	var title, desc *string
	var public *bool

	if actual.Title != desired.Title {
		needsUpdate = true
		title = &desired.Title
	}
	if desired.Description != "" && actual.ShortDescription != desired.Description {
		needsUpdate = true
		desc = &desired.Description
	}
	if p := visibilityToPublic(desired.Visibility); p != nil && *p != actual.Public {
		needsUpdate = true
		public = p
	}

	if needsUpdate {
		logrus.Infof("project %q: would update metadata", desired.Title)
		if opts.Confirm && opts.FixProjects {
			if err := c.UpdateProject(ctx, actual.ID, title, desc, public); err != nil {
				return err
			}
			logrus.Infof("project %q: updated metadata", desired.Title)
		}
	}

	if opts.FixFields {
		fields, err := c.GetProjectFields(ctx, actual.ID)
		if err != nil {
			return err
		}
		if err := reconcileFields(ctx, c, actual.ID, desired.Title, fields, desired.Fields, opts); err != nil {
			return err
		}
	}

	if opts.FixViews {
		views, err := c.GetProjectViews(ctx, actual.ID)
		if err != nil {
			return err
		}
		if err := reconcileViews(ctx, c, actual.ID, desired.Title, views, desired.Views, opts); err != nil {
			return err
		}
	}

	return nil
}

func reconcileFields(ctx context.Context, c client.ProjectsClient, projectID, key string, existing []client.ProjectV2Field, desired []config.Field, opts Options) error {
	existingByName := make(map[string]*client.ProjectV2Field, len(existing))
	for i := range existing {
		existingByName[existing[i].Name] = &existing[i]
	}

	for _, df := range desired {
		actual, found := existingByName[df.Name]
		if !found {
			logrus.Infof("project %s: would create field %q (type=%s)", key, df.Name, df.Type)
			if opts.Confirm {
				var fieldOpts []client.ProjectV2FieldOption
				for _, o := range df.Options {
					fieldOpts = append(fieldOpts, client.ProjectV2FieldOption{
						Name:        o.Name,
						Description: o.Description,
						Color:       o.Color,
					})
				}
				if _, err := c.CreateField(ctx, projectID, df.Name, configTypeToGraphQL(df.Type), fieldOpts); err != nil {
					return err
				}
				logrus.Infof("project %s: created field %q", key, df.Name)
			}
			continue
		}

		// Check if the field type matches — if not, we need to recreate it
		// since GitHub doesn't allow changing field types.
		if actual.Type != configTypeToGraphQL(df.Type) {
			logrus.Warnf("project %s: field %q type mismatch (have=%s, want=%s) — field type cannot be changed in place, skipping", key, df.Name, actual.Type, df.Type)
		}
	}

	return nil
}

func reconcileViews(ctx context.Context, c client.ProjectsClient, projectID, key string, existing []client.ProjectV2View, desired []config.View, opts Options) error {
	existingByName := make(map[string]*client.ProjectV2View, len(existing))
	for i := range existing {
		existingByName[existing[i].Name] = &existing[i]
	}

	for _, dv := range desired {
		actual, found := existingByName[dv.Name]
		ghLayout := configLayoutToGraphQL(dv.Layout)

		if !found {
			logrus.Infof("project %s: would create view %q (layout=%s)", key, dv.Name, dv.Layout)
			if opts.Confirm {
				if _, err := c.CreateView(ctx, projectID, dv.Name, ghLayout); err != nil {
					return err
				}
				logrus.Infof("project %s: created view %q", key, dv.Name)
			}
			continue
		}

		if actual.Layout != ghLayout {
			logrus.Infof("project %s: would update view %q layout (%s -> %s)", key, dv.Name, actual.Layout, ghLayout)
			if opts.Confirm {
				layout := ghLayout
				if err := c.UpdateView(ctx, actual.ID, nil, &layout); err != nil {
					return err
				}
				logrus.Infof("project %s: updated view %q", key, dv.Name)
			}
		}
	}

	return nil
}

func configTypeToGraphQL(t config.FieldType) string {
	switch t {
	case config.FieldTypeText:
		return "TEXT"
	case config.FieldTypeNumber:
		return "NUMBER"
	case config.FieldTypeDate:
		return "DATE"
	case config.FieldTypeSingleSelect:
		return "SINGLE_SELECT"
	case config.FieldTypeIteration:
		return "ITERATION"
	default:
		return string(t)
	}
}

func configLayoutToGraphQL(l config.ViewLayout) string {
	switch l {
	case config.ViewLayoutBoard:
		return "BOARD_LAYOUT"
	case config.ViewLayoutTable:
		return "TABLE_LAYOUT"
	case config.ViewLayoutRoadmap:
		return "ROADMAP_LAYOUT"
	default:
		return string(l)
	}
}

func visibilityToPublic(v string) *bool {
	switch v {
	case "public":
		b := true
		return &b
	case "private":
		b := false
		return &b
	default:
		return nil
	}
}
