package main

import (
	"github.com/google/go-github/github"
	"testing"
)

func Test_checkAnyLabelExists(t *testing.T) {
	type args struct {
		prToCheck     *github.PullRequest
		labelsToCheck []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "does not have any label to check",
			args: args{
				prToCheck:     &github.PullRequest{
					Labels:              []*github.Label{
						label("whatever"),
						label("else"),
						label("kind/enhancement"),
						label("release-note-none"),
					},
				},
				labelsToCheck: []string{
					"lgtm",
					"approved",
					"kind/bug",
				},
			},
			want: false,
		},
		{
			name: "has two of three labels to check",
			args: args{
				prToCheck:     &github.PullRequest{
					Labels:              []*github.Label{
						label("lgtm"),
						label("approved"),
						label("needs-rebase"),
						label("release-note-none"),
					},
				},
				labelsToCheck: []string{
					"lgtm",
					"approved",
					"kind/bug",
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkAnyLabelExists(tt.args.prToCheck, tt.args.labelsToCheck); got != tt.want {
				t.Errorf("checkAnyLabelExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func label(labelText string) *github.Label {
	return &github.Label{
		Name: &labelText,
	}
}
