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
 * Copyright 2020 Red Hat, Inc.
 *
 */
package handler

import (
	v1 "k8s.io/api/core/v1"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
)

func TestShouldPerformOnEvent(t *testing.T) {
	cases := []struct {
		Name     string
		Event    github.PullRequestEvent
		Expected bool
	}{
		{
			"Opened PR",
			github.PullRequestEvent{Action: "opened"},
			true,
		},
		{
			"Edited PR",
			github.PullRequestEvent{Action: "edited"},
			true,
		},
		{
			"Synchronzed PR",
			github.PullRequestEvent{Action: "synchronize"},
			true,
		},
		{
			"Empty action",
			github.PullRequestEvent{Action: ""},
			false,
		},
		{
			"Closed PR",
			github.PullRequestEvent{Action: "closed"},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := shouldPerformOnEventAction(&tc.Event)
			assert.Equal(t, tc.Expected, result)
		})
	}
}

func TestFileByPattern(t *testing.T) {
	cases := []struct {
		name, pattern   string
		input, expected []string
	}{
		{
			"1 match",
			"presubmits/*/*-presubmits.yaml",
			[]string{
				"path/to/file-1",
				"presubmits/repo-1/repo-1-presubmits.yaml",
				"file-2",
			},
			[]string{
				"presubmits/repo-1/repo-1-presubmits.yaml",
			},
		},
		{
			"no matches",
			"presubmits/*/*-presubmits.yaml",
			[]string{
				"path/to/file-1",
				"file-2",
			},
			nil,
		},
		{
			"empty pattern",
			"",
			[]string{
				"path/to/file-1",
				"file-2",
			},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := filterByPattern(tc.input, tc.pattern)
			assert.Nil(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestSquashPresubmits(t *testing.T) {
	cases := []struct {
		Name               string
		OriginalPresubmits []config.Presubmit
		ModifiedPresubmits []config.Presubmit
		ExpectedPresubmits []config.Presubmit
	}{
		{
			Name: "No presubmit spec changes",
			OriginalPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name:           "p1",
						MaxConcurrency: 1,
					},
				},
			},
			ModifiedPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name:           "p1",
						MaxConcurrency: 2,
					},
				},
			},
			ExpectedPresubmits: []config.Presubmit{},
		},
		{
			Name: "Added new presubmit",
			OriginalPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
			},
			ModifiedPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p2",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
			},
			ExpectedPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p2",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
			},
		},
		{
			Name: "Modified spec",
			OriginalPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
			},
			ModifiedPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
						Spec: &v1.PodSpec{
							HostNetwork: true,
						},
					},
				},
			},
			ExpectedPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
						Spec: &v1.PodSpec{
							HostNetwork: true,
						},
					},
				},
			},
		},
		{
			Name: "Modified spec on some, no changes on some",
			OriginalPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p2",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p3",
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p4",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
			},
			ModifiedPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p2",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p3",
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p4",
						Spec: &v1.PodSpec{
							HostNetwork: true,
						},
					},
				},
			},
			ExpectedPresubmits: []config.Presubmit{
				{
					JobBase: config.JobBase{
						Name: "p1",
						Spec: &v1.PodSpec{
							HostNetwork: false,
						},
					},
				},
				{
					JobBase: config.JobBase{
						Name: "p4",
						Spec: &v1.PodSpec{
							HostNetwork: true,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := squashPresubmits(tc.OriginalPresubmits, tc.ModifiedPresubmits)
			assert.Equal(t, tc.ExpectedPresubmits, result)
		})
	}
}

func TestSquashPresubmitsStatic(t *testing.T) {
	cases := []struct {
		Name               string
		OriginalPresubmits map[string][]config.Presubmit
		ModifiedPresubmits map[string][]config.Presubmit
		ExpectedPresubmits map[string][]config.Presubmit
	}{
		{
			Name:               "Nothing changed",
			OriginalPresubmits: nil,
			ModifiedPresubmits: nil,
			ExpectedPresubmits: make(map[string][]config.Presubmit),
		},
		{
			Name: "New repo added",
			OriginalPresubmits: map[string][]config.Presubmit{
				"org/foo": {},
			},
			ModifiedPresubmits: map[string][]config.Presubmit{
				"org/foo": {},
				"org/new": {},
			},
			ExpectedPresubmits: map[string][]config.Presubmit{
				"org/new": {},
				"org/foo": {},
			},
		},
		{
			Name: "Repo deleted",
			OriginalPresubmits: map[string][]config.Presubmit{
				"org/foo":    {},
				"org/delete": {},
			},
			ModifiedPresubmits: map[string][]config.Presubmit{
				"org/foo": {},
			},
			ExpectedPresubmits: map[string][]config.Presubmit{
				"org/foo": {},
			},
		},
		{
			Name: "Repo presubmits modified",
			OriginalPresubmits: map[string][]config.Presubmit{
				"foo/dont-touch": {
					{
						JobBase: config.JobBase{
							Spec: &v1.PodSpec{
								ServiceAccountName: "sa",
							},
						},
					},
				},
				"foo/modify": {
					{
						JobBase: config.JobBase{
							Spec: &v1.PodSpec{
								ServiceAccountName: "sa",
							},
						},
					},
				},
			},
			ModifiedPresubmits: map[string][]config.Presubmit{
				"foo/dont-touch": {
					{
						JobBase: config.JobBase{
							Spec: &v1.PodSpec{
								ServiceAccountName: "sa",
							},
						},
					},
				},
				"foo/modify": {
					{
						JobBase: config.JobBase{
							Spec: &v1.PodSpec{
								ServiceAccountName: "other-sa",
							},
						},
					},
				},
			},
			ExpectedPresubmits: map[string][]config.Presubmit{
				"foo/dont-touch": {},
				"foo/modify": {
					{
						JobBase: config.JobBase{
							Spec: &v1.PodSpec{
								ServiceAccountName: "other-sa",
							},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := squashPresubmitsStatic(tc.OriginalPresubmits, tc.ModifiedPresubmits)
			assert.Equal(t, tc.ExpectedPresubmits, result)
		})
	}
}

func TestSquashConfigPaths(t *testing.T) {
	cases := []struct {
		Name            string
		OriginalConfigs jobConfigPath
		ModifiedConfigs jobConfigPath
		ExpectedConfigs []*config.Config
	}{
		{
			Name: "Nothing changed",
			OriginalConfigs: jobConfigPath{
				"some/path": {},
			},
			ModifiedConfigs: jobConfigPath{
				"some/path": {},
			},
			ExpectedConfigs: []*config.Config{},
		},
		{
			Name: "Config added",
			OriginalConfigs: jobConfigPath{
				"some/path": {},
			},
			ModifiedConfigs: jobConfigPath{
				"some/path": {},
				"new/config": {
					JobConfig: config.JobConfig{
						PresubmitsStatic: map[string][]config.Presubmit{
							"org/repo": {},
						},
					},
				},
			},
			ExpectedConfigs: []*config.Config{
				{
					JobConfig: config.JobConfig{
						PresubmitsStatic: map[string][]config.Presubmit{
							"org/repo": {},
						},
					},
				},
			},
		},
		{
			Name: "Static presubmits modified",
			OriginalConfigs: jobConfigPath{
				"some/path": {
					JobConfig: config.JobConfig{
						PresubmitsStatic: map[string][]config.Presubmit{
							"org/repo": {
								{
									JobBase: config.JobBase{
										Name: "p1",
										Spec: &v1.PodSpec{
											ServiceAccountName: "foo",
										},
									},
								},
							},
						},
					},
				},
			},
			ModifiedConfigs: jobConfigPath{
				"some/path": {
					JobConfig: config.JobConfig{
						PresubmitsStatic: map[string][]config.Presubmit{
							"org/repo": {
								{
									JobBase: config.JobBase{
										Name: "p1",
										Spec: &v1.PodSpec{
											ServiceAccountName: "foo-modified",
										},
									},
								},
							},
						},
					},
				},
			},
			ExpectedConfigs: []*config.Config{
				{
					JobConfig: config.JobConfig{
						PresubmitsStatic: map[string][]config.Presubmit{
							"org/repo": {
								{
									JobBase: config.JobBase{
										Name: "p1",
										Spec: &v1.PodSpec{
											ServiceAccountName: "foo-modified",
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			Name: "Nothing changed",
			OriginalConfigs: jobConfigPath{
				"some/path": {},
			},
			ModifiedConfigs: jobConfigPath{
				"some/path": {},
			},
			ExpectedConfigs: []*config.Config{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			result := squashConfigPaths(tc.OriginalConfigs, tc.ModifiedConfigs)
			assert.Equal(t, tc.ExpectedConfigs, result)
		})
	}
}

func TestAddRepoRef(t *testing.T) {
	cases := []struct {
		Name            string
		Repo            string
		ProwJobInput    prowapi.ProwJob
		ProwJobExpected prowapi.ProwJob
	}{
		{
			Name: "WorkDir exists",
			Repo: "foo/bar",
			ProwJobInput: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					ExtraRefs: []prowapi.Refs{
						{
							WorkDir: true,
						},
					},
				},
			},
			ProwJobExpected: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					ExtraRefs: []prowapi.Refs{
						{
							WorkDir: true,
						},
					},
				},
			},
		},
		{
			Name:         "Set new WorkDir",
			Repo:         "foo/bar",
			ProwJobInput: prowapi.ProwJob{},
			ProwJobExpected: prowapi.ProwJob{
				Spec: prowapi.ProwJobSpec{
					ExtraRefs: []prowapi.Refs{
						{
							Org:        "foo",
							Repo:       "bar",
							RepoLink:   "https://github.com/foo/bar",
							BaseRef:    "refs/heads/master",
							WorkDir:    true,
							CloneDepth: 50,
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			addRepoRef(&tc.ProwJobInput, tc.Repo)
			assert.Equal(t, tc.ProwJobExpected, tc.ProwJobInput)
		})
	}
}
