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
 * Copyright The KubeVirt Authors.
 * Copyright 2017 The Kubernetes Authors.
 *
 */

package main

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"sigs.k8s.io/prow/pkg/github"
)

type fakeClient struct {
	comments        map[int][]github.IssueComment
	createdComments map[int][]string
	issues          []github.Issue
	pullRequests    map[int]*github.PullRequest
	combinedStatus  map[string]*github.CombinedStatus
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		comments:        map[int][]github.IssueComment{},
		createdComments: map[int][]string{},
		pullRequests:    map[int]*github.PullRequest{},
		combinedStatus:  map[string]*github.CombinedStatus{},
	}
}

func (f *fakeClient) CreateComment(owner, repo string, number int, comment string) error {
	f.createdComments[number] = append(f.createdComments[number], comment)
	return nil
}

func (f *fakeClient) ListIssueComments(org, repo string, number int) ([]github.IssueComment, error) {
	return f.comments[number], nil
}

func (f *fakeClient) FindIssues(query, sort string, asc bool) ([]github.Issue, error) {
	return f.issues, nil
}

func (f *fakeClient) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	pr, ok := f.pullRequests[number]
	if !ok {
		return nil, fmt.Errorf("PR %d not found", number)
	}
	return pr, nil
}

func (f *fakeClient) GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error) {
	cs, ok := f.combinedStatus[ref]
	if !ok {
		return nil, fmt.Errorf("combined status for %s not found", ref)
	}
	return cs, nil
}

func TestParseHTMLURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		org  string
		repo string
		num  int
		fail bool
	}{
		{
			name: "normal issue",
			url:  "https://github.com/org/repo/issues/1234",
			org:  "org",
			repo: "repo",
			num:  1234,
		},
		{
			name: "normal pull",
			url:  "https://github.com/pull-org/pull-repo/pull/5555",
			org:  "pull-org",
			repo: "pull-repo",
			num:  5555,
		},
		{
			name: "different host",
			url:  "ftp://gitlab.whatever/org/repo/issues/6666",
			org:  "org",
			repo: "repo",
			num:  6666,
		},
		{
			name: "string issue",
			url:  "https://github.com/org/repo/issues/future",
			fail: true,
		},
		{
			name: "weird issue",
			url:  "https://gubernator.k8s.io/build/kubernetes-ci-logs/logs/ci-kubernetes-e2e-gci-gce/11947/",
			fail: true,
		},
	}

	for _, tc := range cases {
		org, repo, num, err := parseHTMLURL(tc.url)
		if err != nil && !tc.fail {
			t.Errorf("%s: should not have produced error: %v", tc.name, err)
		} else if err == nil && tc.fail {
			t.Errorf("%s: failed to produce an error", tc.name)
		} else {
			if org != tc.org {
				t.Errorf("%s: org %s != expected %s", tc.name, org, tc.org)
			}
			if repo != tc.repo {
				t.Errorf("%s: repo %s != expected %s", tc.name, repo, tc.repo)
			}
			if num != tc.num {
				t.Errorf("%s: num %d != expected %d", tc.name, num, tc.num)
			}
		}
	}
}

func TestMakeQuery(t *testing.T) {
	cases := []struct {
		name       string
		query      string
		archived   bool
		closed     bool
		locked     bool
		dur        time.Duration
		expected   []string
		unexpected []string
	}{
		{
			name:       "basic query",
			query:      "hello world",
			expected:   []string{"hello world"},
			unexpected: []string{"updated:"},
		},
		{
			name:     "basic duration",
			query:    "hello",
			dur:      1 * time.Hour,
			expected: []string{"hello", "updated:<"},
		},
		{
			name:       "weird characters not escaped",
			query:      "oh yeah!@#$&*()",
			expected:   []string{"!", "@", "#", " "},
			unexpected: []string{"%", "+"},
		},
		{
			name:     "linebreaks are replaced by whitespaces",
			query:    "label:foo\nlabel:bar",
			expected: []string{"label:foo label:bar"},
		},
	}

	for _, tc := range cases {
		actual := makeQuery(tc.query, tc.dur)
		for _, e := range tc.expected {
			if !strings.Contains(actual, e) {
				t.Errorf("%s: could not find %s in %s", tc.name, e, actual)
			}
		}
		for _, u := range tc.unexpected {
			if strings.Contains(actual, u) {
				t.Errorf("%s: should not have found %s in %s", tc.name, u, actual)
			}
		}
	}
}

func TestInitRequiredPresubmits(t *testing.T) {
	cases := []struct {
		name     string
		required bool
	}{
		{
			"pull-kubevirt-e2e-kind-sriov",
			true,
		},
		{
			"pull-kubevirt-fuzz",
			false,
		},
	}

	err := initPresubmitRequiredMap("testdata")
	if err != nil {
		t.Errorf("failed to init map: %v", err)
	}

	for _, tc := range cases {
		_, actual := presubmitRequiredMap[tc.name]
		if tc.required != actual {
			if tc.required {
				t.Errorf("%s should be required but isn't", tc.name)
			} else {
				t.Errorf("%s should NOT be required but is", tc.name)
			}
		}
	}
}

func TestExtractPresubmitName(t *testing.T) {
	cases := []struct {
		name      string
		targetURL string
		expected  string
	}{
		{
			name:      "prow target URL",
			targetURL: "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/14804/pull-kubevirt-build-s390x/1234567890",
			expected:  "pull-kubevirt-build-s390x",
		},
		{
			name:      "no match",
			targetURL: "https://coveralls.io/builds/abc",
			expected:  "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := extractPresubmitName(tc.targetURL)
			if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestDeterministicJobPattern(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{"pull-kubevirt-build", true},
		{"pull-kubevirt-build-arm64", true},
		{"pull-kubevirt-build-s390x", true},
		{"pull-kubevirt-build-cs10", true},
		{"pull-kubevirt-generate", true},
		{"pull-kubevirt-e2e-k8s-1.33-sig-compute", false},
		{"pull-kubevirt-e2e-kind-sriov", false},
		{"pull-kubevirt-unit-test", false},
		{"pull-kubevirt-verify-go-mod", false},
		{"pull-kubevirt-code-lint", false},
		{"pull-kubevirt-goveralls", false},
		{"pull-kubevirt-fossa", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := deterministicJobPattern.MatchString(tc.name)
			if actual != tc.expected {
				t.Errorf("deterministicJobPattern.MatchString(%q) = %v, want %v", tc.name, actual, tc.expected)
			}
		})
	}
}

func prowTargetURL(jobName string) string {
	return "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/1/" + jobName + "/1234567890"
}

func TestNotFitForRetest(t *testing.T) {
	// Set up the required presubmit map for tests
	savedMap := presubmitRequiredMap
	defer func() { presubmitRequiredMap = savedMap }()

	presubmitRequiredMap = map[string]struct{}{
		"pull-kubevirt-build":                      {},
		"pull-kubevirt-build-s390x":                {},
		"pull-kubevirt-generate":                   {},
		"pull-kubevirt-verify-go-mod":              {},
		"pull-kubevirt-unit-test":                  {},
		"pull-kubevirt-e2e-k8s-1.33-sig-compute":  {},
		"pull-kubevirt-e2e-k8s-1.34-sig-compute":  {},
		"pull-kubevirt-e2e-k8s-1.35-sig-compute":  {},
		"pull-kubevirt-e2e-k8s-1.33-sig-operator": {},
		"pull-kubevirt-e2e-k8s-1.34-sig-operator": {},
		"pull-kubevirt-e2e-k8s-1.35-sig-operator": {},
		"pull-kubevirt-e2e-k8s-1.33-sig-storage":  {},
		"pull-kubevirt-e2e-k8s-1.34-sig-storage":  {},
		"pull-kubevirt-e2e-k8s-1.35-sig-storage":  {},
	}

	cases := []struct {
		name            string
		failedRequired  []string
		allStatuses     []github.Status
		expectNotFit    bool
		expectSubstring string
	}{
		{
			name:           "deterministic build failure (PR 14804 pattern)",
			failedRequired: []string{"pull-kubevirt-build-s390x", "pull-kubevirt-generate"},
			allStatuses: []github.Status{
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-build-s390x")},
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-generate")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-compute")},
			},
			expectNotFit:    true,
			expectSubstring: "Deterministic jobs failed",
		},
		{
			name:           "e2e SIG lane fails on all k8s versions (PR 17243 pattern)",
			failedRequired: []string{"pull-kubevirt-e2e-k8s-1.33-sig-operator", "pull-kubevirt-e2e-k8s-1.34-sig-operator"},
			allStatuses: []github.Status{
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-operator")},
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-operator")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-compute")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-compute")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-build")},
			},
			expectNotFit:    true,
			expectSubstring: "E2E lane(s) failing on all k8s versions: sig-operator",
		},
		{
			name:           "e2e failure on some but not all k8s versions (flake)",
			failedRequired: []string{"pull-kubevirt-e2e-k8s-1.33-sig-compute"},
			allStatuses: []github.Status{
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-compute")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-compute")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.35-sig-compute")},
			},
			expectNotFit: false,
		},
		{
			name:           "e2e failure on single k8s version only (no multi-version data)",
			failedRequired: []string{"pull-kubevirt-e2e-k8s-1.33-sig-storage"},
			allStatuses: []github.Status{
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-storage")},
			},
			expectNotFit: false,
		},
		{
			name:           "multiple e2e SIG lanes failing on all versions",
			failedRequired: []string{"pull-kubevirt-e2e-k8s-1.33-sig-operator", "pull-kubevirt-e2e-k8s-1.34-sig-operator", "pull-kubevirt-e2e-k8s-1.33-sig-storage", "pull-kubevirt-e2e-k8s-1.34-sig-storage"},
			allStatuses: []github.Status{
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-operator")},
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-operator")},
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-storage")},
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-storage")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-compute")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-compute")},
			},
			expectNotFit:    true,
			expectSubstring: "sig-operator",
		},
		{
			name:           "mixed deterministic and e2e failures prefers deterministic reason",
			failedRequired: []string{"pull-kubevirt-build", "pull-kubevirt-e2e-k8s-1.33-sig-operator", "pull-kubevirt-e2e-k8s-1.34-sig-operator"},
			allStatuses: []github.Status{
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-build")},
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-operator")},
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-operator")},
			},
			expectNotFit:    true,
			expectSubstring: "Deterministic jobs failed",
		},
		{
			name:           "non-build non-e2e failure does not trigger deterministic skip",
			failedRequired: []string{"pull-kubevirt-verify-go-mod"},
			allStatuses: []github.Status{
				{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-verify-go-mod")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-compute")},
				{State: "success", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-compute")},
			},
			expectNotFit: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason := notFitForRetest(tc.failedRequired, tc.allStatuses)
			if tc.expectNotFit {
				if reason == "" {
					t.Error("expected not fit for retest, but got empty reason")
				}
				if tc.expectSubstring != "" && !strings.Contains(reason, tc.expectSubstring) {
					t.Errorf("expected reason to contain %q, got %q", tc.expectSubstring, reason)
				}
			} else {
				if reason != "" {
					t.Errorf("expected fit for retest, but got reason: %q", reason)
				}
			}
		})
	}
}

func TestLastCommentMatches(t *testing.T) {
	cases := []struct {
		name     string
		comments []github.IssueComment
		comment  string
		expected bool
	}{
		{
			name:     "no comments",
			comments: nil,
			comment:  "hello",
			expected: false,
		},
		{
			name: "last comment matches",
			comments: []github.IssueComment{
				{Body: "old comment"},
				{Body: "hello"},
			},
			comment:  "hello",
			expected: true,
		},
		{
			name: "last comment differs",
			comments: []github.IssueComment{
				{Body: "hello"},
				{Body: "something else"},
			},
			comment:  "hello",
			expected: false,
		},
		{
			name: "matches with whitespace differences",
			comments: []github.IssueComment{
				{Body: "  hello\n"},
			},
			comment:  "hello",
			expected: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newFakeClient()
			c.comments[1] = tc.comments
			actual := lastCommentMatches(c, "org", "repo", 1, tc.comment)
			if actual != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, actual)
			}
		})
	}
}

func TestRunSkipsDuplicateSkipComment(t *testing.T) {
	savedMap := presubmitRequiredMap
	defer func() { presubmitRequiredMap = savedMap }()

	presubmitRequiredMap = map[string]struct{}{
		"pull-kubevirt-e2e-k8s-1.33-sig-storage": {},
		"pull-kubevirt-e2e-k8s-1.34-sig-storage": {},
	}

	skipMsg := fmt.Sprintf(skipComment, "E2E lane(s) failing on all k8s versions: sig-storage")

	c := newFakeClient()
	c.issues = []github.Issue{
		{HTMLURL: "https://github.com/kubevirt/kubevirt/pull/1"},
	}
	c.pullRequests[1] = &github.PullRequest{
		Head: github.PullRequestBranch{SHA: "abc123"},
	}
	c.combinedStatus["abc123"] = &github.CombinedStatus{
		Statuses: []github.Status{
			{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-storage")},
			{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-storage")},
		},
	}
	c.comments[1] = []github.IssueComment{
		{Body: skipMsg},
	}

	err := run(c, "test query", "", false, comment, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.createdComments[1]) > 0 {
		t.Errorf("expected no new comments, got %d: %v", len(c.createdComments[1]), c.createdComments[1])
	}
}

func TestRunPostsSkipCommentWhenDifferent(t *testing.T) {
	savedMap := presubmitRequiredMap
	defer func() { presubmitRequiredMap = savedMap }()

	presubmitRequiredMap = map[string]struct{}{
		"pull-kubevirt-e2e-k8s-1.33-sig-storage": {},
		"pull-kubevirt-e2e-k8s-1.34-sig-storage": {},
	}

	c := newFakeClient()
	c.issues = []github.Issue{
		{HTMLURL: "https://github.com/kubevirt/kubevirt/pull/1"},
	}
	c.pullRequests[1] = &github.PullRequest{
		Head: github.PullRequestBranch{SHA: "abc123"},
	}
	c.combinedStatus["abc123"] = &github.CombinedStatus{
		Statuses: []github.Status{
			{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.33-sig-storage")},
			{State: "failure", TargetURL: prowTargetURL("pull-kubevirt-e2e-k8s-1.34-sig-storage")},
		},
	}
	c.comments[1] = []github.IssueComment{
		{Body: "some other comment"},
	}

	err := run(c, "test query", "", false, comment, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.createdComments[1]) != 1 {
		t.Errorf("expected 1 new comment, got %d", len(c.createdComments[1]))
	}
}
