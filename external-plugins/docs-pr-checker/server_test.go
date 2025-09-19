package main

import (
	"testing"

	"github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/external-plugins/docs-pr-checker/fakegithub"
	"sigs.k8s.io/prow/pkg/github"
)

func newTestServer(fakeGitHubClient *fakegithub.FakeClient) *Server {
	return &Server{
		ghc: fakeGitHubClient,
		log: logrus.NewEntry(logrus.New()),
	}
}

func TestCheckAndUpdateDocsPRStatus(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		initialLabels   []string
		expectedAdd     []string
		expectedRemove  []string
		expectComment   bool
		expectedComment string
	}{
		{
			name:          "no docs-pr, no labels",
			body:          "Some PR description",
			initialLabels: []string{},
			expectedAdd:   []string{labelDocsPRRequired},
			expectComment: false,
		},
		{
			name:          "docs-pr NONE, no labels",
			body:          "Some PR description\n\n```docs-pr\nNONE\n```",
			initialLabels: []string{},
			expectedAdd:   []string{labelDocsPRNone},
			expectComment: false,
		},
		{
			name:          "docs-pr #1234, no labels",
			body:          "Some PR description\n\n```docs-pr\n#1234\n```",
			initialLabels: []string{},
			expectedAdd:   []string{labelDocsPR},
			expectComment: false,
		},
		{
			name:           "no docs-pr, has docs-pr label",
			body:           "Some PR description",
			initialLabels:  []string{labelDocsPR},
			expectedAdd:    []string{labelDocsPRRequired},
			expectedRemove: []string{labelDocsPR},
			expectComment:  false,
		},
		{
			name:           "docs-pr NONE, has docs-pr-required label",
			body:           "Some PR description\n\n```docs-pr\nNONE\n```",
			initialLabels:  []string{labelDocsPRRequired},
			expectedAdd:    []string{labelDocsPRNone},
			expectedRemove: []string{labelDocsPRRequired},
			expectComment:  false,
		},
		{
			name:           "docs-pr #1234, has docs-pr-none label",
			body:           "Some PR description\n\n```docs-pr\n#1234\n```",
			initialLabels:  []string{labelDocsPRNone},
			expectedAdd:    []string{labelDocsPR},
			expectedRemove: []string{labelDocsPRNone},
			expectComment:  false,
		},
		{
			name:            "invalid docs-pr value",
			body:            "Some PR description\n\n```docs-pr\ninvalid\n```",
			initialLabels:   []string{},
			expectedAdd:     []string{labelDocsPRRequired},
			expectComment:   true,
			expectedComment: "Invalid `docs-pr` value: `invalid`. Please use `NONE` or a valid PR reference (e.g., `#123`, `repo#123`, `org/repo#123`).",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeGitHubClient := fakegithub.NewFakeClient()
			s := newTestServer(fakeGitHubClient)
			org := "kubevirt"
			repo := "project-infra"
			prNumber := 123

			fakeGitHubClient.SetLabels(org, repo, prNumber, tt.initialLabels)

			pr := &github.PullRequest{
				Body:   tt.body,
				Number: prNumber,
			}

			err := s.checkAndUpdateDocsPRStatus(s.log, org, repo, pr)
			if err != nil {
				t.Fatalf("checkAndUpdateDocsPRStatus() returned error: %v", err)
			}

			for _, label := range tt.expectedAdd {
				if !fakeGitHubClient.HasLabel(org, repo, prNumber, label) {
					t.Errorf("expected to add label %q, but it was not added", label)
				}
			}

			for _, label := range tt.expectedRemove {
				if !fakeGitHubClient.HasRemovedLabel(org, repo, prNumber, label) {
					t.Errorf("expected to remove label %q, but it was not removed", label)
				}
			}

			comments := fakeGitHubClient.GetComments(org, repo, prNumber)
			if tt.expectComment {
				if len(comments) == 0 {
					t.Errorf("expected a comment to be created, but none was")
				} else if comments[0] != tt.expectedComment {
					t.Errorf("expected comment %q, but got %q", tt.expectedComment, comments[0])
				}
			} else if len(comments) > 0 {
				t.Errorf("expected no comments to be created, but got %v", comments)
			}
		})
	}
}

func TestExtractDocsPRValue(t *testing.T) {
	s := newTestServer(nil)

	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
		{
			name:     "body with docs-pr NONE",
			body:     "Some PR description\n\n```docs-pr\nNONE\n```",
			expected: "NONE",
		},
		{
			name:     "body with docs-pr number",
			body:     "Some PR description\n\n```docs-pr\n#1234\n```",
			expected: "#1234",
		},
		{
			name:     "body with empty docs-pr",
			body:     "Some PR description\n\n```docs-pr\n\n```",
			expected: "",
		},
		{
			name:     "body with whitespace in docs-pr",
			body:     "Some PR description\n\n```docs-pr\n  #1234  \n```",
			expected: "#1234",
		},
		{
			name:     "body without docs-pr section",
			body:     "Some PR description without docs-pr section",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.extractDocsPRValue(tt.body)
			if result != tt.expected {
				t.Errorf("extractDocsPRValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestUpdateDocsPRInBody(t *testing.T) {
	s := newTestServer(nil)

	tests := []struct {
		name     string
		body     string
		newValue string
		expected string
	}{
		{
			name:     "add docs-pr to empty body",
			body:     "",
			newValue: "NONE",
			expected: "\n```docs-pr\nNONE\n```",
		},
		{
			name:     "replace existing docs-pr",
			body:     "Some description\n\n```docs-pr\n#1234\n```\n\nMore text",
			newValue: "NONE",
			expected: "Some description\n\nMore text\n```docs-pr\nNONE\n```",
		},
		{
			name:     "add docs-pr to body without existing section",
			body:     "Some PR description",
			newValue: "#5678",
			expected: "Some PR description\n```docs-pr\n#5678\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.updateDocsPRInBody(tt.body, tt.newValue)
			if result != tt.expected {
				t.Errorf("updateDocsPRInBody() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHasLabel(t *testing.T) {
	s := newTestServer(nil)

	labels := []github.Label{
		{Name: "bug"},
		{Name: "enhancement"},
		{Name: "docs-pr"},
	}

	tests := []struct {
		name      string
		labelName string
		expected  bool
	}{
		{
			name:      "label exists",
			labelName: "docs-pr",
			expected:  true,
		},
		{
			name:      "label does not exist",
			labelName: "do-not-merge/docs-pr-required",
			expected:  false,
		},
		{
			name:      "empty label name",
			labelName: "",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.hasLabel(labels, tt.labelName)
			if result != tt.expected {
				t.Errorf("hasLabel() = %t, want %t", result, tt.expected)
			}
		})
	}
}
