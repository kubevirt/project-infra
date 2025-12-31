package main

import (
	"testing"

	"sigs.k8s.io/prow/pkg/github"
)

func TestExtractDocsPRValue(t *testing.T) {
	s := &Server{}
	
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name: "empty body",
			body: "",
			expected: "",
		},
		{
			name: "body with docs-pr NONE",
			body: "Some PR description\n\n```docs-pr\nNONE\n```",
			expected: "NONE",
		},
		{
			name: "body with docs-pr number",
			body: "Some PR description\n\n```docs-pr\n#1234\n```",
			expected: "#1234",
		},
		{
			name: "body with empty docs-pr",
			body: "Some PR description\n\n```docs-pr\n\n```",
			expected: "",
		},
		{
			name: "body with whitespace in docs-pr",
			body: "Some PR description\n\n```docs-pr\n  #1234  \n```",
			expected: "#1234",
		},
		{
			name: "body without docs-pr section",
			body: "Some PR description without docs-pr section",
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
	s := &Server{}
	
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
			expected: "\n\n```docs-pr\nNONE\n```",
		},
		{
			name:     "replace existing docs-pr",
			body:     "Some description\n\n```docs-pr\n#1234\n```\n\nMore text",
			newValue: "NONE",
			expected: "Some description\n\n```docs-pr\nNONE\n```\n\nMore text",
		},
		{
			name:     "add docs-pr to body without existing section",
			body:     "Some PR description",
			newValue: "#5678",
			expected: "Some PR description\n\n```docs-pr\n#5678\n```",
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
	s := &Server{}
	
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

