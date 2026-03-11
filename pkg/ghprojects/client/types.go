package client

import "github.com/shurcooL/githubv4"

// ProjectV2 represents a GitHub Project V2 as returned by the GraphQL API.
type ProjectV2 struct {
	ID               string
	Number           int
	Title            string
	ShortDescription string
	Public           bool
	Fields           []ProjectV2Field
	Views            []ProjectV2View
}

// ProjectV2Field represents a field on a GitHub Project V2.
type ProjectV2Field struct {
	ID      string
	Name    string
	Type    string // TEXT, NUMBER, DATE, SINGLE_SELECT, ITERATION
	Options []ProjectV2FieldOption
}

// ProjectV2FieldOption represents an option on a single-select or iteration field.
type ProjectV2FieldOption struct {
	ID          string
	Name        string
	Description string
	Color       string
}

// ProjectV2View represents a view on a GitHub Project V2.
type ProjectV2View struct {
	ID     string
	Name   string
	Layout string // BOARD_LAYOUT, TABLE_LAYOUT, ROADMAP_LAYOUT
}

// graphQL query/mutation types for shurcooL/githubv4

type queryOrgID struct {
	Organization struct {
		ID githubv4.ID
	} `graphql:"organization(login: $login)"`
}

type queryOrgProjects struct {
	Organization struct {
		ProjectsV2 struct {
			Nodes []struct {
				ID               githubv4.String
				Number           githubv4.Int
				Title            githubv4.String
				ShortDescription githubv4.String
				Public           githubv4.Boolean
			}
			PageInfo struct {
				HasNextPage githubv4.Boolean
				EndCursor   githubv4.String
			}
		} `graphql:"projectsV2(first: $first, after: $after)"`
	} `graphql:"organization(login: $login)"`
}

type queryProjectFields struct {
	Node struct {
		ProjectV2 struct {
			Fields struct {
				Nodes []projectFieldNode
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"fields(first: $first, after: $after)"`
		} `graphql:"... on ProjectV2"`
	} `graphql:"node(id: $projectID)"`
}

type projectFieldNode struct {
	TypeName             string `graphql:"__typename"`
	ProjectV2FieldCommon `graphql:"... on ProjectV2Field"`
	ProjectV2SingleSelectField `graphql:"... on ProjectV2SingleSelectField"`
	ProjectV2IterationField    `graphql:"... on ProjectV2IterationField"`
}

type ProjectV2FieldCommon struct {
	ID       githubv4.String
	Name     githubv4.String
	DataType githubv4.String
}

type ProjectV2SingleSelectField struct {
	ID       githubv4.String
	Name     githubv4.String
	DataType githubv4.String
	Options  []struct {
		ID          githubv4.String
		Name        githubv4.String
		Description githubv4.String
		Color       githubv4.String
	}
}

type ProjectV2IterationField struct {
	ID       githubv4.String
	Name     githubv4.String
	DataType githubv4.String
}

type queryProjectViews struct {
	Node struct {
		ProjectV2 struct {
			Views struct {
				Nodes []struct {
					ID     githubv4.String
					Name   githubv4.String
					Layout githubv4.String
				}
				PageInfo struct {
					HasNextPage githubv4.Boolean
					EndCursor   githubv4.String
				}
			} `graphql:"views(first: $first, after: $after)"`
		} `graphql:"... on ProjectV2"`
	} `graphql:"node(id: $projectID)"`
}

type mutationCreateProject struct {
	CreateProjectV2 struct {
		ProjectV2 struct {
			ID     githubv4.String
			Number githubv4.Int
		}
	} `graphql:"createProjectV2(input: $input)"`
}

type mutationUpdateProject struct {
	UpdateProjectV2 struct {
		ProjectV2 struct {
			ID githubv4.String
		}
	} `graphql:"updateProjectV2(input: $input)"`
}

type mutationDeleteProject struct {
	DeleteProjectV2 struct {
		ProjectV2 struct {
			ID githubv4.String
		}
	} `graphql:"deleteProjectV2(input: $input)"`
}

type mutationCreateField struct {
	CreateProjectV2Field struct {
		ProjectV2Field struct {
			ProjectV2FieldCommon `graphql:"... on ProjectV2Field"`
			ProjectV2SingleSelectField `graphql:"... on ProjectV2SingleSelectField"`
		}
	} `graphql:"createProjectV2Field(input: $input)"`
}

type mutationDeleteField struct {
	DeleteProjectV2Field struct {
		ProjectV2Field struct {
			ProjectV2FieldCommon `graphql:"... on ProjectV2Field"`
		}
	} `graphql:"deleteProjectV2Field(input: $input)"`
}

type mutationUpdateField struct {
	UpdateProjectV2Field struct {
		ProjectV2Field struct {
			ProjectV2FieldCommon `graphql:"... on ProjectV2Field"`
		}
	} `graphql:"updateProjectV2Field(input: $input)"`
}

// Mutation input types — defined locally because the vendored githubv4
// library predates Projects V2.

type createProjectV2Input struct {
	OwnerID          githubv4.ID     `json:"ownerId"`
	Title            githubv4.String `json:"title"`
	ClientMutationID *githubv4.String `json:"clientMutationId,omitempty"`
}

type updateProjectV2Input struct {
	ProjectID        githubv4.ID      `json:"projectId"`
	Title            *githubv4.String  `json:"title,omitempty"`
	ShortDescription *githubv4.String  `json:"shortDescription,omitempty"`
	Public           *githubv4.Boolean `json:"public,omitempty"`
	ClientMutationID *githubv4.String  `json:"clientMutationId,omitempty"`
}

type deleteProjectV2Input struct {
	ProjectID        githubv4.ID     `json:"projectId"`
	ClientMutationID *githubv4.String `json:"clientMutationId,omitempty"`
}

type createProjectV2FieldInput struct {
	ProjectID        githubv4.ID                          `json:"projectId"`
	DataType         githubv4.String                      `json:"dataType"`
	Name             githubv4.String                      `json:"name"`
	SingleSelectOptions []createProjectV2SingleSelectOption `json:"singleSelectOptions,omitempty"`
	ClientMutationID *githubv4.String                     `json:"clientMutationId,omitempty"`
}

type createProjectV2SingleSelectOption struct {
	Name        githubv4.String `json:"name"`
	Description githubv4.String `json:"description"`
	Color       githubv4.String `json:"color"`
}

type updateProjectV2FieldInput struct {
	FieldID          githubv4.ID     `json:"fieldId"`
	Name             *githubv4.String `json:"name,omitempty"`
	ClientMutationID *githubv4.String `json:"clientMutationId,omitempty"`
}

type deleteProjectV2FieldInput struct {
	FieldID          githubv4.ID     `json:"fieldId"`
	ClientMutationID *githubv4.String `json:"clientMutationId,omitempty"`
}

type mutationCreateView struct {
	CreateProjectV2View struct {
		ProjectV2View struct {
			ID     githubv4.String
			Name   githubv4.String
			Layout githubv4.String
		}
	} `graphql:"createProjectV2View(input: $input)"`
}

type createProjectV2ViewInput struct {
	ProjectID        githubv4.ID     `json:"projectId"`
	Name             githubv4.String `json:"name"`
	Layout           githubv4.String `json:"layout,omitempty"`
	ClientMutationID *githubv4.String `json:"clientMutationId,omitempty"`
}

type mutationUpdateView struct {
	UpdateProjectV2View struct {
		ProjectV2View struct {
			ID githubv4.String
		}
	} `graphql:"updateProjectV2View(input: $input)"`
}

type updateProjectV2ViewInput struct {
	ViewID           githubv4.ID      `json:"viewId"`
	Name             *githubv4.String `json:"name,omitempty"`
	Layout           *githubv4.String `json:"layout,omitempty"`
	ClientMutationID *githubv4.String `json:"clientMutationId,omitempty"`
}

type mutationDeleteView struct {
	DeleteProjectV2View struct {
		ProjectV2View struct {
			ID githubv4.String
		}
	} `graphql:"deleteProjectV2View(input: $input)"`
}

type deleteProjectV2ViewInput struct {
	ViewID           githubv4.ID     `json:"viewId"`
	ClientMutationID *githubv4.String `json:"clientMutationId,omitempty"`
}
