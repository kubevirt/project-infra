/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 *
 */
package main

import (
	"testing"

	"github.com/google/go-github/github"
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
				prToCheck: &github.PullRequest{
					Labels: []*github.Label{
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
				prToCheck: &github.PullRequest{
					Labels: []*github.Label{
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
		{
			name: "has one out of three labels to check",
			args: args{
				prToCheck: &github.PullRequest{
					Labels: []*github.Label{
						label("lgtm"),
						label("approved"),
						label("needs-rebase"),
						label("release-note-none"),
					},
				},
				labelsToCheck: []string{
					"approved",
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
