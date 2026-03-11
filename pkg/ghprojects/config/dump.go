package config

import (
	"context"
	"fmt"

	"kubevirt.io/project-infra/pkg/ghprojects/client"
)

// Dump exports the current GitHub Projects V2 state for an organization as config.
func Dump(ctx context.Context, c client.ProjectsClient, orgName string) (*FullConfig, error) {
	projects, err := c.ListProjects(ctx, orgName)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	orgProjects := OrgProjects{
		Projects: make([]Project, 0, len(projects)),
	}

	for _, p := range projects {
		proj := Project{
			Title:       p.Title,
			Description: p.ShortDescription,
		}
		if p.Public {
			proj.Visibility = "public"
		} else {
			proj.Visibility = "private"
		}

		fields, err := c.GetProjectFields(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("listing fields for project %q: %w", p.Title, err)
		}
		for _, f := range fields {
			ft, ok := graphQLTypeToConfigType(f.Type)
			if !ok {
				continue
			}
			cf := Field{
				Name: f.Name,
				Type: ft,
			}
			for _, o := range f.Options {
				cf.Options = append(cf.Options, FieldOption{
					Name:        o.Name,
					Description: o.Description,
					Color:       o.Color,
				})
			}
			proj.Fields = append(proj.Fields, cf)
		}

		views, err := c.GetProjectViews(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("listing views for project %q: %w", p.Title, err)
		}
		for _, v := range views {
			vl, ok := graphQLLayoutToConfigLayout(v.Layout)
			if !ok {
				continue
			}
			proj.Views = append(proj.Views, View{
				Name:   v.Name,
				Layout: vl,
			})
		}

		orgProjects.Projects = append(orgProjects.Projects, proj)
	}

	return &FullConfig{
		Orgs: map[string]OrgProjects{
			orgName: orgProjects,
		},
	}, nil
}

func graphQLTypeToConfigType(t string) (FieldType, bool) {
	switch t {
	case "TEXT":
		return FieldTypeText, true
	case "NUMBER":
		return FieldTypeNumber, true
	case "DATE":
		return FieldTypeDate, true
	case "SINGLE_SELECT":
		return FieldTypeSingleSelect, true
	case "ITERATION":
		return FieldTypeIteration, true
	default:
		return "", false
	}
}

func graphQLLayoutToConfigLayout(l string) (ViewLayout, bool) {
	switch l {
	case "BOARD_LAYOUT":
		return ViewLayoutBoard, true
	case "TABLE_LAYOUT":
		return ViewLayoutTable, true
	case "ROADMAP_LAYOUT":
		return ViewLayoutRoadmap, true
	default:
		return "", false
	}
}
