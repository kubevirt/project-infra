package main

import (
	"testing"
)

func TestTemplateRegexParsing(t *testing.T) {
	s := &Server{}
	
	tests := []struct {
		name     string
		prBody   string
		expected string
	}{
		{
			name: "template with empty docs-pr field",
			prBody: `**What this PR does / why we need it**:
This PR adds a new feature.

**Which issue(s) this PR fixes**:
Fixes #1234

**Documentation update**:
<!-- Add your Docs PR number -->
` + "```docs-pr\n\n```" + `

**Special notes for your reviewer**:
Please review carefully.`,
			expected: "",
		},
		{
			name: "template with NONE in docs-pr field",
			prBody: `**What this PR does / why we need it**:
This PR adds a new feature.

**Documentation update**:
<!-- Add your Docs PR number -->
` + "```docs-pr\nNONE\n```" + `

**Special notes for your reviewer**:
Please review carefully.`,
			expected: "NONE",
		},
		{
			name: "template with PR number in docs-pr field",
			prBody: `**What this PR does / why we need it**:
This PR adds a new feature.

**Documentation update**:
` + "```docs-pr\n#5678\n```" + `

**Special notes for your reviewer**:
Please review carefully.`,
			expected: "#5678",
		},
		{
			name: "template with multiple lines in docs-pr field should take first line",
			prBody: `**Documentation update**:
` + "```docs-pr\n#1234\nextra text that should be ignored\n```",
			expected: "#1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.extractDocsPRValue(tt.prBody)
			if result != tt.expected {
				t.Errorf("extractDocsPRValue() = %q, want %q", result, tt.expected)
			}
		})
	}
}

