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

package review

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	testdiff "github.com/andreyvit/diff"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/go-diff/diff"
	"sigs.k8s.io/prow/pkg/github"
)

func TestGuessReviewTypes(t *testing.T) {
	diffFilePaths := []string{
		"testdata/simple_bump-prow-job-images_sh.patch0",
		"testdata/simple_bump-prow-job-images_sh.patch1",
		"testdata/move_prometheus_stack.patch0",
		"testdata/move_prometheus_stack.patch1",
		"testdata/cdi_arm_release.patch0",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch0",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch1",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch2",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch3",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch4",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch00",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch01",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch02",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch03",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch04",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch05",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch06",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch07",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch08",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch09",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch10",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch11",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch12",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch13",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch14",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch15",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch16",
	}
	diffFilePathsToDiffs := map[string]*diff.FileDiff{}
	for _, diffFile := range diffFilePaths {
		bumpImagesDiffFile, err := os.ReadFile(diffFile)
		if err != nil {
			t.Errorf("failed to read diff: %v", err)
		}
		bumpFileDiffs, err := diff.ParseFileDiff(bumpImagesDiffFile)
		if err != nil {
			t.Errorf("failed to read diff: %v", err)
		}
		diffFilePathsToDiffs[diffFile] = bumpFileDiffs
	}
	type args struct {
		fileDiffs []*diff.FileDiff
	}
	tests := []struct {
		name string
		args args
		want []KindOfChange
	}{
		{
			name: "simple image bump should yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
					diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch1"],
				},
			},
			want: []KindOfChange{
				&ProwJobImageUpdate{
					relevantFileDiffs: []*diff.FileDiff{
						diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
						diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch1"],
					},
				},
			},
		},
		{
			name: "mixed with image bump should yield a partial change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch0"],
				},
			},
			want: []KindOfChange{
				&ProwJobImageUpdate{
					relevantFileDiffs: []*diff.FileDiff{
						diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
					},
				},
			},
		},
		{
			name: "non image bump (move_prometheus_stack) should not yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch0"],
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch1"],
				},
			},
			want: nil,
		},
		{
			name: "non image bump (cdi_arm_release) should not yield a change 2",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/cdi_arm_release.patch0"],
				},
			},
			want: nil,
		},
		{
			name: "non kubevirtci bump (eviction-strategy-doc) should not yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch0"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch1"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch2"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch3"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch4"],
				},
			},
			want: nil,
		},
		{
			name: "non kubevirtci bump (fix-containerdisks-migration) should not yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch00"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch01"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch02"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch03"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch04"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch05"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch06"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch07"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch08"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch09"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch10"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch11"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch12"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch13"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch14"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch15"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch16"],
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GuessReviewTypes(tt.args.fileDiffs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GuessReviewTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fields struct {
	l       *logrus.Entry
	org     string
	repo    string
	num     int
	user    string
	action  github.PullRequestEventAction
	dryRun  bool
	BaseSHA string
}

func newFields() fields {
	return fields{
		l:       newEntry(),
		org:     "",
		repo:    "",
		num:     0,
		user:    "",
		action:  "",
		dryRun:  false,
		BaseSHA: "",
	}
}
func TestReviewer_AttachReviewComments(t *testing.T) {

	type args struct {
		botReviewResults []BotReviewResult
		githubClient     FakeGHReviewClient
	}
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantErr            bool
		wantReviewComments []*FakeComment
	}{
		{
			name:   "basic comment",
			fields: newFields(),
			args: args{
				githubClient:     newGHReviewClient(),
				botReviewResults: []BotReviewResult{},
			},
			wantErr: false,
			wantReviewComments: []*FakeComment{
				{
					Org:    "",
					Repo:   "",
					Number: 0,
					Comment: `@pr-reviewer's review-bot says:



This PR satisfies all automated review criteria.

/lgtm
/approve

This PR does not require further manual action.

**Note: botreview (kubevirt/project-infra#3100) is a Work In Progress!**
`,
				},
			},
		},
		{
			name:   "review approved",
			fields: newFields(),
			args: args{
				githubClient: newGHReviewClient(),
				botReviewResults: []BotReviewResult{
					NewShouldNotMergeReviewResult("approved", "disapproved", "should not get merged at all reason"),
				},
			},
			wantErr: false,
			wantReviewComments: []*FakeComment{
				{
					Org:    "",
					Repo:   "",
					Number: 0,
					Comment: `@pr-reviewer's review-bot says:

approved

This PR satisfies all automated review criteria.

/lgtm
/approve

Holding this PR because:
* should not get merged at all reason

/hold

**Note: botreview (kubevirt/project-infra#3100) is a Work In Progress!**
`,
				},
			},
		},
		{
			name:   "one review not approved",
			fields: newFields(),
			args: args{
				githubClient: newGHReviewClient(),
				botReviewResults: []BotReviewResult{
					newReviewResultWithData(
						"approved",
						"disapproved",
						map[string][]*diff.Hunk{"test": {{Body: []byte("nil")}}},
						"should not get merged at all reason",
					),
				},
			},
			wantErr: false,
			wantReviewComments: []*FakeComment{
				{
					Org:    "",
					Repo:   "",
					Number: 0,
					Comment: `@pr-reviewer's review-bot says:

disapproved

<details>

_test_

~~~diff
nil
~~~

</details>


This PR does not satisfy at least one automated review criteria.

Holding this PR because:
* should not get merged at all reason

/hold

**Note: botreview (kubevirt/project-infra#3100) is a Work In Progress!**
`,
				},
			},
		},
		{
			name:   "one review without hunks, not approved",
			fields: newFields(),
			args: args{
				githubClient: newGHReviewClient(),
				botReviewResults: []BotReviewResult{
					newReviewResultWithData(
						"approved",
						"disapproved",
						map[string][]*diff.Hunk{
							"blah": nil,
						},
						"should not get merged at all reason",
					),
				},
			},
			wantErr: false,
			wantReviewComments: []*FakeComment{
				{
					Org:    "",
					Repo:   "",
					Number: 0,
					Comment: `@pr-reviewer's review-bot says:

disapproved

<details>

_blah_

</details>


This PR does not satisfy at least one automated review criteria.

Holding this PR because:
* should not get merged at all reason

/hold

**Note: botreview (kubevirt/project-infra#3100) is a Work In Progress!**
`,
				},
			},
		},
		{
			name:   "two reviews not approved",
			fields: newFields(),
			args: args{
				githubClient: newGHReviewClient(),
				botReviewResults: []BotReviewResult{
					newReviewResultWithData(
						"approved",
						"can't approve moo",
						map[string][]*diff.Hunk{"mehFile": {{Body: []byte("moo")}}},
						"should not get merged at all",
					),
					newReviewResultWithData(
						"approved",
						"will not approve meh",
						map[string][]*diff.Hunk{"mooFile": {{Body: []byte("meh")}}},
						"should not get merged in any case",
					),
				},
			},
			wantErr: false,
			wantReviewComments: []*FakeComment{
				{
					Org:    "",
					Repo:   "",
					Number: 0,
					Comment: `@pr-reviewer's review-bot says:

can't approve moo

<details>

_mehFile_

~~~diff
moo
~~~

</details>

will not approve meh

<details>

_mooFile_

~~~diff
meh
~~~

</details>


This PR does not satisfy at least one automated review criteria.

Holding this PR because:
* should not get merged at all
* should not get merged in any case

/hold

**Note: botreview (kubevirt/project-infra#3100) is a Work In Progress!**
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reviewer{
				l:       tt.fields.l,
				org:     tt.fields.org,
				repo:    tt.fields.repo,
				num:     tt.fields.num,
				user:    tt.fields.user,
				action:  tt.fields.action,
				dryRun:  tt.fields.dryRun,
				BaseSHA: tt.fields.BaseSHA,
			}
			if err := r.AttachReviewComments(tt.args.botReviewResults, &tt.args.githubClient); (err != nil) != tt.wantErr {
				t.Errorf("AttachReviewComments() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.githubClient.FakeComments, tt.wantReviewComments) {
				t.Errorf("AttachReviewComments() reviewComments = %v, wantReviewComments %v", tt.args.githubClient.FakeComments, tt.wantReviewComments)
				for i, comment := range tt.wantReviewComments {
					if len(tt.args.githubClient.FakeComments) <= i {
						t.Errorf("no comment on %d found", i)
						continue
					}
					t.Errorf("diff:\n%s", testdiff.LineDiff(tt.args.githubClient.FakeComments[i].Comment, comment.Comment))
				}
			}
		})
	}
}

func newEntry() *logrus.Entry {
	return logrus.NewEntry(logrus.StandardLogger())
}

type FakeComment struct {
	Org     string `json:"org,omitempty"`
	Repo    string `json:"repo,omitempty"`
	Number  int    `json:"number,omitempty"`
	Comment string `json:"comment,omitempty"`
}

func (f FakeComment) String() string {
	marshal, _ := json.Marshal(f)
	return string(marshal)
}

type FakeGHReviewClient struct {
	FakeComments []*FakeComment
}

func (f *FakeGHReviewClient) CreateComment(org, repo string, number int, comment string) error {
	f.FakeComments = append(f.FakeComments, &FakeComment{
		Org:     org,
		Repo:    repo,
		Number:  number,
		Comment: comment,
	})
	return nil
}

func (f *FakeGHReviewClient) BotUser() (*github.UserData, error) {
	return &github.UserData{
		Login: "pr-reviewer",
	}, nil
}

func newGHReviewClient() FakeGHReviewClient {
	return FakeGHReviewClient{}
}
