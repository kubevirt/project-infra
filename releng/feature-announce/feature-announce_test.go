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
 * Copyright the KubeVirt Authors.
 */

package main

import (
	"reflect"
	"strings"
	"testing"
)

func Test_featureAnnouncer_extractReleaseNoteContent(t *testing.T) {
	type args struct {
		number int
		body   []string
	}
	tests := []struct {
		name    string
		args    args
		want    *ReleaseNote
		wantErr bool
	}{
		{
			name: "no release note",
			args: args{
				number: 1742,
				body:   strings.Split(`NONE`, "\n"),
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "release note NONE",
			args: args{
				number: 1742,
				body: []string{
					"```release-note",
					"NONE",
					"```",
				},
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "release note MEH",
			args: args{
				number: 1742,
				body: []string{
					"```release-note",
					"MEH",
					"```",
				},
			},
			want: &ReleaseNote{
				PullRequestNumber: 1742,
				GitHubHandle:      "someUser",
				ReleaseNote:       "MEH",
			},
			wantErr: false,
		},
		{
			name: "release note multiline",
			args: args{
				number: 1742,
				body: []string{
					"```release-note",
					"",
					" - documents steps to build the KubeVirt builder container",
					"",
					"```",
				},
			},
			want: &ReleaseNote{
				PullRequestNumber: 1742,
				GitHubHandle:      "someUser",
				ReleaseNote:       "- documents steps to build the KubeVirt builder container",
			},
			wantErr: false,
		},
		{
			name: "release note multiline 2",
			args: args{
				number: 1742,
				body: []string{
					"```release-note",
					`Added “adm” subcommand under “virtctl”, and “log-verbosity" subcommand under “adm”. The log-verbosity command is:
- To show the log verbosity of one or more components.
- To set the log verbosity of one or more components.
- To reset the log verbosity of all components (reset to the default verbosity (2)).`,
					"```",
				},
			},
			want: &ReleaseNote{
				PullRequestNumber: 1742,
				GitHubHandle:      "someUser",
				ReleaseNote: `Added “adm” subcommand under “virtctl”, and “log-verbosity" subcommand under “adm”. The log-verbosity command is:
- To show the log verbosity of one or more components.
- To set the log verbosity of one or more components.
- To reset the log verbosity of all components (reset to the default verbosity (2)).`,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &featureAnnouncer{}
			got, err := f.extractReleaseNoteContent(tt.args.number, tt.args.body, "someUser")
			if (err != nil) != tt.wantErr {
				t.Errorf("extractReleaseNoteContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractReleaseNoteContent() got = %v, want %v", got, tt.want)
			}
		})
	}
}
