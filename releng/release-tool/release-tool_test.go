package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/config"

	"github.com/google/go-github/v32/github"
)

func standardCleanup(r *releaseData) {
	_ = os.RemoveAll(r.cacheDir)
}

func standardSetup() releaseData {
	repo := "fake-repo"
	org := "fake-org"
	token := "fake-token"
	cacheDir, err := os.MkdirTemp("/tmp", "release-tool-unit-test")
	if err != nil {
		panic(err)
	}

	truePtr := true
	falsePtr := false
	idPtr := int64(0)

	v1Branch := "release-0.1"

	v1 := "v0.1.0"
	v1CreatedAt := &github.Timestamp{
		Time: time.Date(2020, time.January, 1, 7, 0, 0, 0, time.UTC),
	}

	v1RC0 := "v0.1.0-rc.1"
	v1RC0CreatedAt := &github.Timestamp{
		Time: time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC),
	}

	now := time.Date(2020, time.February, 1, 1, 0, 0, 0, time.UTC)

	releases := []*github.RepositoryRelease{
		{
			TagName:    &v1,
			CreatedAt:  v1CreatedAt,
			Draft:      &falsePtr,
			Prerelease: &falsePtr,
			Assets:     []*github.ReleaseAsset{{ID: &idPtr}},
		},
		{
			TagName:    &v1RC0,
			CreatedAt:  v1RC0CreatedAt,
			Draft:      &falsePtr,
			Prerelease: &truePtr,
			Assets:     []*github.ReleaseAsset{{ID: &idPtr}},
		},
	}
	branches := []*github.Branch{
		{
			Name: &v1Branch,
		},
	}

	blockerListCache := make(map[string]*blockerListCacheEntry)
	blockerListCache["release-blocker/master"] = &blockerListCacheEntry{}
	blockerListCache["release-blocker/release-0.1"] = &blockerListCacheEntry{}
	blockerListCache["release-blocker/release-0.2"] = &blockerListCacheEntry{}

	r := releaseData{
		org:              org,
		repo:             repo,
		allReleases:      releases,
		allBranches:      branches,
		blockerListCache: blockerListCache,
		now:              now,
		gitUser:          "fake-user",
		gitEmail:         "fake-email@fake.fake",
		repoUrl:          fmt.Sprintf("https://%s@github.com/%s/%s.git", token, org, repo),
		infraUrl:         fmt.Sprintf("https://%s@github.com/kubevirt/project-infra.git", token),
		repoDir:          fmt.Sprintf("%s/%s/https-%s", cacheDir, org, repo),
		infraDir:         fmt.Sprintf("%s/%s/https-%s", cacheDir, "kubevirt", "project-infra"),
		dryRun:           true,
		cacheDir:         cacheDir,
	}

	return r
}

func TestAutoRelease(t *testing.T) {

	var tests = []struct {
		name           string
		now            time.Time
		expectedBranch string
		expectedTag    string
		hasBlocker     bool
	}{
		{
			name:           "Expect new branch and rc",
			now:            time.Date(2020, time.February, 1, 1, 0, 0, 0, time.UTC),
			expectedBranch: "release-0.2",
			expectedTag:    "v0.2.0-rc.0",
		},
		{
			name:           "Expect new branch to be blocked",
			now:            time.Date(2020, time.February, 1, 1, 0, 0, 0, time.UTC),
			expectedBranch: "release-0.2",
			expectedTag:    "v0.2.0-rc.0",
			hasBlocker:     true,
		},
		{
			name:           "Expect no new branch or rc",
			now:            time.Date(2020, time.January, 31, 1, 0, 0, 0, time.UTC),
			expectedBranch: "",
			expectedTag:    "",
		},
	}

	for _, tc := range tests {
		t.Logf("test case %s", tc.name)

		r := standardSetup()
		defer standardCleanup(&r)
		r.now = tc.now

		if tc.hasBlocker {
			open := "open"
			id := int64(1)
			num := int(1)
			url := "someurl"
			title := "sometitle"
			r.blockerListCache["release-blocker/master"] = &blockerListCacheEntry{
				allBlockerIssues: []*github.Issue{
					{
						ID:     &id,
						Number: &num,
						State:  &open,
						URL:    &url,
						Title:  &title,
					},
				},
			}
		}

		err := r.autoDetectData("monthly", 7)

		if err != nil {
			t.Errorf("got unexpected error %s", err)
		} else if tc.expectedBranch != r.newBranch {
			t.Errorf("Expected branch [%s] and got [%s]", tc.expectedBranch, r.newBranch)
		} else if tc.expectedTag != r.tag {
			t.Errorf("Expected tag [%s] and got [%s]", tc.expectedTag, r.tag)
		}

		if r.newBranch != "" {
			err := r.verifyBranch()
			if err != nil {
				t.Errorf("got unexpected error %s", err)
			}

			blocked, err := r.isBranchBlocked("master")
			if err != nil {
				t.Errorf("got unexpected error %s", err)
			}

			if tc.hasBlocker != blocked {
				t.Errorf("expected blocker [%t] but got [%t]", tc.hasBlocker, blocked)
			}

		}
	}

}

func TestAutoPromoteRC(t *testing.T) {
	v2RC0 := "v0.2.0-rc.0"
	v2Branch := "release-0.2"
	v2 := "v0.2.0"
	v2RC0CreatedAt := &github.Timestamp{
		Time: time.Date(2020, time.February, 1, 1, 0, 0, 0, time.UTC),
	}

	var tests = []struct {
		name                    string
		now                     time.Time
		expectPromotion         bool
		hasClosedBlockerPR      bool
		hasOpenBlockerIssue     bool
		expectNewRC             bool
		hasOutdatedBlockerPR    bool
		hasOutdatedBlockerIssue bool
	}{
		{
			name:               "wait for promotion",
			now:                time.Date(2020, time.February, 7, 1, 0, 0, 0, time.UTC),
			expectPromotion:    false,
			hasClosedBlockerPR: false,
			expectNewRC:        false,
		},
		{
			name:               "should promote",
			now:                time.Date(2020, time.February, 8, 1, 0, 0, 0, time.UTC),
			expectPromotion:    true,
			hasClosedBlockerPR: false,
			expectNewRC:        false,
		},
		{
			name:                 "should promote with old blocker PR that closed before release",
			now:                  time.Date(2020, time.February, 8, 1, 0, 0, 0, time.UTC),
			expectPromotion:      true,
			hasClosedBlockerPR:   false,
			expectNewRC:          false,
			hasOutdatedBlockerPR: true,
		},
		{
			name:                    "should promote with old blocker ISSUE that closed before release",
			now:                     time.Date(2020, time.February, 8, 1, 0, 0, 0, time.UTC),
			expectPromotion:         true,
			hasClosedBlockerPR:      false,
			expectNewRC:             false,
			hasOutdatedBlockerIssue: true,
		},
		{
			name:               "should block promotion due to blocker PR and create new RC",
			now:                time.Date(2020, time.February, 8, 1, 0, 0, 0, time.UTC),
			expectPromotion:    false,
			hasClosedBlockerPR: true,
			expectNewRC:        true,
		},
		{
			name:               "should create new RC once blocker PRs are closed",
			now:                time.Date(2020, time.February, 2, 1, 0, 0, 0, time.UTC),
			expectPromotion:    false,
			hasClosedBlockerPR: true,
			expectNewRC:        true,
		},
		{
			name:                "should block promotion due to blocker ISSUE and prevent new RC",
			now:                 time.Date(2020, time.February, 8, 1, 0, 0, 0, time.UTC),
			expectPromotion:     false,
			hasClosedBlockerPR:  false,
			expectNewRC:         false,
			hasOpenBlockerIssue: true,
		},
	}

	for _, tc := range tests {

		t.Logf("test case %s", tc.name)

		r := standardSetup()
		defer standardCleanup(&r)
		r.allReleases = append(r.allReleases, &github.RepositoryRelease{
			TagName:   &v2RC0,
			CreatedAt: v2RC0CreatedAt,
		})
		r.allBranches = append(r.allBranches, &github.Branch{Name: &v2Branch})

		if tc.hasClosedBlockerPR {
			state := "closed"
			id := int64(1)
			num := int(1)
			url := "someurl"
			title := "sometitle"
			r.blockerListCache["release-blocker/release-0.2"] = &blockerListCacheEntry{
				allBlockerPRs: []*github.PullRequest{
					{
						ID:       &id,
						Number:   &num,
						State:    &state,
						URL:      &url,
						Title:    &title,
						ClosedAt: &r.now,
					},
				},
			}
		}

		if tc.hasOutdatedBlockerPR {
			state := "closed"
			id := int64(1)
			num := int(1)
			url := "someurl"
			title := "sometitle"
			closedAt := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)

			r.blockerListCache["release-blocker/release-0.2"] = &blockerListCacheEntry{
				allBlockerPRs: []*github.PullRequest{
					{
						ID:       &id,
						Number:   &num,
						State:    &state,
						URL:      &url,
						Title:    &title,
						ClosedAt: &closedAt,
					},
				},
			}
		}

		if tc.hasOpenBlockerIssue {
			open := "open"
			id := int64(1)
			num := int(1)
			url := "someurl"
			title := "sometitle"
			r.blockerListCache["release-blocker/release-0.2"] = &blockerListCacheEntry{
				allBlockerIssues: []*github.Issue{
					{
						ID:     &id,
						Number: &num,
						State:  &open,
						URL:    &url,
						Title:  &title,
					},
				},
			}
		}

		if tc.hasOutdatedBlockerIssue {
			state := "closed"
			id := int64(1)
			num := int(1)
			url := "someurl"
			title := "sometitle"
			closedAt := time.Date(2020, time.January, 1, 1, 0, 0, 0, time.UTC)
			r.blockerListCache["release-blocker/release-0.2"] = &blockerListCacheEntry{
				allBlockerIssues: []*github.Issue{
					{
						ID:       &id,
						Number:   &num,
						State:    &state,
						URL:      &url,
						Title:    &title,
						ClosedAt: &closedAt,
					},
				},
			}
		}

		r.now = tc.now

		err := r.autoDetectData("monthly", 7)

		if err != nil {
			t.Errorf("got unexpected error %s", err)
		} else if tc.expectPromotion && r.promoteRC != v2RC0 {
			t.Errorf("expected to autopromote RC")
		} else if !tc.expectPromotion && r.promoteRC != "" {
			t.Errorf("did not expect to autopromote RC")
		} else if tc.expectNewRC && r.tag != "v0.2.0-rc.1" {
			t.Errorf("expected new RC")
		} else if !tc.expectNewRC && r.tag != "" {
			t.Errorf("did not expect new RC")
		} else if r.newBranch != "" {
			t.Errorf("did not expect new branch")
		}

		if r.tag != "" {
			err := r.verifyTag()
			if err != nil {
				t.Errorf("got unexpected error %s", err)
			} else if r.tagBranch != v2Branch {
				t.Errorf("expected to use branch %s but got %s", v2Branch, r.tagBranch)
			}
		}

		if tc.expectPromotion {
			err := r.verifyPromoteRC()
			if err != nil {
				t.Errorf("got unexpected error %s", err)
			} else if r.tag != v2 {
				t.Errorf("Expected promotion of rc to tag %s but got %s", v2, r.tag)
			}
		}
	}
}

func TestCutNewBranch(t *testing.T) {

	expectedGitCommands := []string{}

	r := standardSetup()
	defer standardCleanup(&r)
	r.newBranch = "release-0.2"

	r.dryRun = false
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [clone https://fake-token@github.com/kubevirt/project-infra.git %s/kubevirt/https-project-infra]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/kubevirt/https-project-infra config user.name fake-user]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/kubevirt/https-project-infra config user.email fake-email@fake.fake]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [clone https://fake-token@github.com/fake-org/fake-repo.git %s/fake-org/https-fake-repo]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo config user.name fake-user]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo config user.email fake-email@fake.fake]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo checkout -b release-0.2]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo push https://fake-token@github.com/fake-org/fake-repo.git release-0.2]", r.cacheDir))

	seenGitCommands := []string{}
	// override gitCommand with mock function
	gitCommand = func(arg ...string) (string, error) {
		seenGitCommands = append(seenGitCommands, fmt.Sprintf("git %s", arg))
		return "", nil
	}

	err := r.cutNewBranch(false)
	if err != nil {
		t.Errorf("got unexpected error %s", err)
	} else if len(expectedGitCommands) != len(seenGitCommands) {
		t.Errorf("got unexpected git commands")
	}

	for i, entry := range seenGitCommands {
		if entry != expectedGitCommands[i] {
			t.Errorf("expected command %s and got %s", expectedGitCommands[i], entry)
		}
	}
}

func runProwGit(t *testing.T, args ...string) string {
	t.Helper()
	output, err := _gitCommand(args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return strings.TrimSpace(output)
}

func setupProwConfigTest(t *testing.T, owner string, handler http.HandlerFunc) (releaseData, string, string) {
	t.Helper()
	r := standardSetup()
	t.Cleanup(func() { standardCleanup(&r) })
	r.org, r.repo, r.newBranch, r.gitToken, r.dryRun = "kubevirt", "kubevirt", "release-1.7", "fake-token", false
	base := t.TempDir()
	upstream, fork := filepath.Join(base, "upstream.git"), filepath.Join(base, "fork.git")
	gitConfig := filepath.Join(base, "gitconfig")
	if err := os.WriteFile(gitConfig, []byte(fmt.Sprintf(`[url %q]
	insteadOf = https://fake-token@github.com/kubevirt/project-infra.git
[url %q]
	insteadOf = https://fake-token@github.com/%s/project-infra.git
[protocol "https"]
	allow = never
[user]
	name = fake-user
	email = fake-email@fake.fake
`, upstream, fork, owner)), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	oldGitCommand, oldConfigForkerPath := gitCommand, configForkerPath
	t.Cleanup(func() { gitCommand, configForkerPath = oldGitCommand, oldConfigForkerPath })
	gitCommand = _gitCommand
	configForkerPath = filepath.Join(base, "config-forker")
	if err := os.WriteFile(configForkerPath, []byte(`#!/bin/sh
set -eu
while [ "$#" -gt 0 ]; do
  case "$1" in --job-config) input="$2"; shift;; --output) output="$2"; shift;; esac
  shift
done
cp "$input" "$output"
`), 0700); err != nil {
		t.Fatal(err)
	}
	seed := filepath.Join(base, "seed")
	jobDir := filepath.Join(seed, "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt")
	if err := os.MkdirAll(jobDir, 0755); err != nil {
		t.Fatal(err)
	}
	job, err := os.ReadFile("testdata/jobconfig/kubevirt-presubmits-1.6.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jobDir, "kubevirt-presubmits.yaml"), job, 0644); err != nil {
		t.Fatal(err)
	}
	runProwGit(t, "-C", seed, "init", "-b", "main")
	runProwGit(t, "-C", seed, "add", ".")
	runProwGit(t, "-C", seed, "commit", "-m", "seed")
	runProwGit(t, "clone", "--bare", seed, upstream)
	runProwGit(t, "clone", "--bare", upstream, fork)
	if err := r.checkoutProjectInfra(); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	r.githubClient = github.NewClient(server.Client())
	r.githubClient.BaseURL, err = url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	return r, upstream, fork
}

func TestForkProwJobs(t *testing.T) {
	for _, tc := range []struct {
		name         string
		override     string
		owner        string
		createStatus int
	}{
		{name: "authenticated user", owner: "Bhup-GitHUB", createStatus: http.StatusCreated},
		{name: "override and failed PR retry", override: "RELEASE-TEAM", owner: "Release-Team", createStatus: http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			branch := "kubevirt_kubevirt_release-1.7_configs"
			outputConfig := "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits-1.7.yaml"
			prCalls := 0
			r, upstream, fork := setupProwConfigTest(t, tc.owner, func(w http.ResponseWriter, req *http.Request) {
				switch req.Method + " " + req.URL.Path {
				case "GET /user":
					if tc.override != "" {
						t.Error("explicit fork owner must not require authenticated user lookup")
					}
					fmt.Fprintf(w, `{"login":%q}`, tc.owner)
				case "GET /repos/" + cmp.Or(tc.override, tc.owner) + "/project-infra":
					if tc.override == "" {
						fmt.Fprint(w, `{"fork":true,"parent":{"full_name":"kubevirt/project-infra"}}`)
					} else {
						fmt.Fprintf(w, `{"fork":true,"owner":{"login":%q},"parent":{"full_name":"another/project-infra"},"source":{"full_name":"kubevirt/project-infra"}}`, tc.owner)
					}
				case "POST /repos/kubevirt/project-infra/pulls":
					var payload map[string]any
					if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
						t.Error(err)
					}
					want := map[string]any{
						"title": "Release configs for kubevirt/kubevirt release branch release-1.7",
						"body":  "adds new release configs", "head": tc.owner + ":" + branch,
						"base": "main", "maintainer_can_modify": false,
					}
					if !reflect.DeepEqual(payload, want) {
						t.Errorf("PR payload = %#v, want %#v", payload, want)
					}
					prCalls++
					status := tc.createStatus
					if prCalls > 1 {
						status = http.StatusUnprocessableEntity
					}
					w.WriteHeader(status)
					fmt.Fprint(w, `{}`)
				case "GET /repos/kubevirt/project-infra/pulls":
					query := req.URL.Query()
					if query.Get("state") != "open" || query.Get("head") != tc.owner+":"+branch || query.Get("base") != "main" {
						t.Errorf("unexpected PR lookup: %s", req.URL.RawQuery)
					}
					fmt.Fprintf(w, `[{"state":"open","head":{"label":%q},"base":{"ref":"main"}}]`, tc.owner+":"+branch)
				default:
					t.Errorf("unexpected GitHub request: %s %s", req.Method, req.URL)
					http.Error(w, "unexpected request", http.StatusInternalServerError)
				}
			})
			r.forkOwner = tc.override
			if tc.override == "" {
				// Git emits diagnostics even when the fork has no matching ref.
				t.Setenv("GIT_TRACE", "1")
			} else {
				// As on GitHub, both spellings resolve to the same fork.
				runProwGit(t, "config", "--global", "--add", "url."+fork+".insteadOf", "https://fake-token@github.com/"+tc.override+"/project-infra.git")
			}
			err := r.forkProwJobs()
			t.Setenv("GIT_TRACE", "0")
			if (err != nil) != (tc.createStatus != http.StatusCreated) {
				t.Fatalf("forkProwJobs error = %v, PR response status = %d", err, tc.createStatus)
			}
			if runProwGit(t, "ls-remote", "--heads", upstream, "refs/heads/"+branch) != "" {
				t.Fatal("config branch was pushed to upstream")
			}
			if runProwGit(t, "ls-remote", "--heads", fork, "refs/heads/"+branch) == "" {
				t.Fatal("config branch was not pushed to the fork")
			}
			message := runProwGit(t, "--git-dir", fork, "log", "-1", "--format=%B", branch)
			if !strings.Contains(message, "Signed-off-by: fake-user <fake-email@fake.fake>") {
				t.Fatalf("config commit is not signed off: %s", message)
			}
			collaborator := filepath.Join(t.TempDir(), "collaborator")
			runProwGit(t, "clone", "--branch", branch, fork, collaborator)
			generated := runProwGit(t, "-C", collaborator, "show", "HEAD:"+outputConfig)
			edited := strings.Replace(generated, "max_concurrency: 4", "max_concurrency: 3", 1)
			if edited == generated {
				t.Fatal("fixture has no concurrency setting to edit")
			}
			if err := os.WriteFile(filepath.Join(collaborator, outputConfig), []byte(edited+"\n"), 0644); err != nil {
				t.Fatal(err)
			}
			runProwGit(t, "-C", collaborator, "commit", "-am", "adjust fork config")
			runProwGit(t, "-C", collaborator, "push", "origin", branch)
			r.infraDir = filepath.Join(t.TempDir(), "fresh-cache")
			if err := r.checkoutProjectInfra(); err != nil {
				t.Fatal(err)
			}
			if err := r.forkProwJobs(); err != nil {
				t.Fatalf("retry with existing fork branch and PR: %v", err)
			}
			if got := runProwGit(t, "-C", r.infraDir, "show", "HEAD:"+outputConfig); got != edited {
				t.Fatal("retry did not preserve the fork's config edit")
			}
			if runProwGit(t, "-C", r.infraDir, "config", "--get", "remote.origin.url") != r.infraUrl {
				t.Fatal("upstream checkout origin was replaced with the fork")
			}
		})
	}
}

func TestForkProwJobsInvalidFork(t *testing.T) {
	for _, tc := range []struct {
		name, owner, login, body, wantError string
		status                              int
	}{
		{name: "missing", owner: "Bhup-GitHUB", status: http.StatusNotFound, wantError: "a writable fork of kubevirt/project-infra is required"},
		{name: "inaccessible", owner: "Bhup-GitHUB", status: http.StatusForbidden, wantError: "a writable fork of kubevirt/project-infra is required"},
		{name: "not a fork", owner: "Bhup-GitHUB", body: `{"fork":false}`, wantError: "not a fork of kubevirt/project-infra"},
		{name: "unrelated fork", owner: "Bhup-GitHUB", body: `{"fork":true,"parent":{"full_name":"other/project-infra"}}`, wantError: "not a fork of kubevirt/project-infra"},
		{name: "upstream override", owner: "KuBeViRt", wantError: "fork owner must not be kubevirt"},
		{name: "upstream login", login: "kubevirt", wantError: "fork owner must not be kubevirt"},
		{name: "user lookup fails", status: http.StatusUnauthorized, wantError: "looking up project-infra fork owner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, fork := setupProwConfigTest(t, "Bhup-GitHUB", func(w http.ResponseWriter, req *http.Request) {
				switch req.Method + " " + req.URL.Path {
				case "GET /user":
					if tc.status != 0 {
						w.WriteHeader(tc.status)
					}
					fmt.Fprintf(w, `{"login":%q}`, tc.login)
				case "GET /repos/Bhup-GitHUB/project-infra":
					if tc.status != 0 {
						w.WriteHeader(tc.status)
					}
					fmt.Fprint(w, tc.body)
				default:
					t.Errorf("unexpected GitHub request: %s %s", req.Method, req.URL)
					http.Error(w, "unexpected request", http.StatusInternalServerError)
				}
			})
			r.forkOwner = tc.owner
			if err := r.forkProwJobs(); err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("forkProwJobs error = %v, want %q", err, tc.wantError)
			}
			if runProwGit(t, "ls-remote", "--heads", fork, "refs/heads/kubevirt_kubevirt_release-1.7_configs") != "" {
				t.Fatal("invalid fork was modified")
			}
		})
	}
}

func TestForkProwJobsWithoutFork(t *testing.T) {
	for _, name := range []string{"dry run", "no template", "already on main"} {
		t.Run(name, func(t *testing.T) {
			r, upstream, fork := setupProwConfigTest(t, "Bhup-GitHUB", func(w http.ResponseWriter, req *http.Request) {
				t.Errorf("unexpected GitHub request: %s %s", req.Method, req.URL)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			})
			outputConfig := "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits-1.7.yaml"
			switch name {
			case "dry run":
				r.dryRun = true
			case "no template":
				r.repo = "no-template"
			case "already on main":
				if err := os.WriteFile(filepath.Join(r.infraDir, outputConfig), []byte("existing config\n"), 0644); err != nil {
					t.Fatal(err)
				}
				runProwGit(t, "-C", r.infraDir, "add", outputConfig)
				runProwGit(t, "-C", r.infraDir, "commit", "-m", "config already on main")
				runProwGit(t, "-C", r.infraDir, "push", "origin", "main")
			}
			if err := r.forkProwJobs(); err != nil {
				t.Fatal(err)
			}
			for _, remote := range []string{upstream, fork} {
				if runProwGit(t, "ls-remote", "--heads", remote, "refs/heads/kubevirt_kubevirt_release-1.7_configs") != "" {
					t.Fatalf("unexpected config branch on %s", remote)
				}
			}
			if name == "dry run" {
				jobConfig, err := config.ReadJobConfig(filepath.Join(r.infraDir, outputConfig))
				if err != nil {
					t.Fatalf("dry run did not generate a local config: %v", err)
				}
				if !slices.ContainsFunc(jobConfig.PresubmitsStatic["kubevirt/kubevirt"], func(job config.Presubmit) bool {
					return job.Name == "pull-kubevirt-verify-rpm-urls-1.7"
				}) {
					t.Fatal("generated config is missing the RPM URLs presubmit")
				}
			} else if runProwGit(t, "-C", r.infraDir, "branch", "--show-current") != "main" {
				t.Fatal("skipped config generation changed the branch")
			}
		})
	}
}

func TestCutNewBranchRejectsUnpublishedProwConfig(t *testing.T) {
	for _, state := range []string{"untracked", "staged", "local commit"} {
		t.Run(state, func(t *testing.T) {
			r, upstream, _ := setupProwConfigTest(t, "Bhup-GitHUB", func(w http.ResponseWriter, req *http.Request) {
				t.Errorf("unexpected GitHub request: %s %s", req.Method, req.URL)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			})
			// Give the release branch a local destination so a skipped config
			// would let the release proceed instead of failing on network access.
			r.repoUrl = upstream
			outputConfig := "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits-1.7.yaml"
			if err := os.WriteFile(filepath.Join(r.infraDir, outputConfig), []byte("unpublished config\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if state != "untracked" {
				runProwGit(t, "-C", r.infraDir, "add", outputConfig)
			}
			if state == "local commit" {
				runProwGit(t, "-C", r.infraDir, "commit", "-m", "config only on local main")
			}
			t.Setenv("GIT_TRACE", "1")
			err := r.cutNewBranch(false)
			t.Setenv("GIT_TRACE", "0")
			if runProwGit(t, "ls-remote", "--heads", upstream, "refs/heads/"+r.newBranch) != "" {
				t.Error("release branch was cut without publishing its Prow config")
			}
			if err == nil || !strings.Contains(err.Error(), "not on upstream main") || !strings.Contains(err.Error(), "--clean-cache=true") {
				t.Fatalf("cutNewBranch error = %v, want an unpublished config cache error", err)
			}
		})
	}
}

func TestForkProwJobsUnmatchedPR(t *testing.T) {
	for _, tc := range []struct {
		name, pull string
	}{
		{name: "validation error", pull: `[]`},
		{name: "upstream head", pull: `[{"state":"open","head":{"label":"kubevirt:kubevirt_kubevirt_release-1.7_configs"},"base":{"ref":"main"}}]`},
		{name: "different base", pull: `[{"state":"open","head":{"label":"Bhup-GitHUB:kubevirt_kubevirt_release-1.7_configs"},"base":{"ref":"release-1.7"}}]`},
		{name: "closed PR", pull: `[{"state":"closed","head":{"label":"Bhup-GitHUB:kubevirt_kubevirt_release-1.7_configs"},"base":{"ref":"main"}}]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _, _ := setupProwConfigTest(t, "Bhup-GitHUB", func(w http.ResponseWriter, req *http.Request) {
				switch req.Method + " " + req.URL.Path {
				case "GET /repos/Bhup-GitHUB/project-infra":
					fmt.Fprint(w, `{"fork":true,"parent":{"full_name":"kubevirt/project-infra"}}`)
				case "POST /repos/kubevirt/project-infra/pulls":
					w.WriteHeader(http.StatusUnprocessableEntity)
					fmt.Fprint(w, `{"message":"Validation Failed"}`)
				case "GET /repos/kubevirt/project-infra/pulls":
					fmt.Fprint(w, tc.pull)
				default:
					t.Errorf("unexpected GitHub request: %s %s", req.Method, req.URL)
					http.Error(w, "unexpected request", http.StatusInternalServerError)
				}
			})
			r.forkOwner = "Bhup-GitHUB"
			if err := r.forkProwJobs(); err == nil || !strings.Contains(err.Error(), "Validation Failed") {
				t.Fatalf("unmatched PR must not suppress the creation error: %v", err)
			}
		})
	}
}

func TestForkProwJobsGitFailure(t *testing.T) {
	for _, name := range []string{"inaccessible fork ref", "concurrent update"} {
		t.Run(name, func(t *testing.T) {
			r, upstream, fork := setupProwConfigTest(t, "Bhup-GitHUB", func(w http.ResponseWriter, req *http.Request) {
				if req.Method == "GET" && req.URL.Path == "/repos/Bhup-GitHUB/project-infra" {
					fmt.Fprint(w, `{"fork":true,"parent":{"full_name":"kubevirt/project-infra"}}`)
					return
				}
				t.Errorf("unexpected GitHub request: %s %s", req.Method, req.URL)
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			})
			r.forkOwner = "Bhup-GitHUB"
			branch := "kubevirt_kubevirt_release-1.7_configs"
			var concurrentCommit string
			if name == "inaccessible fork ref" {
				if err := os.RemoveAll(fork); err != nil {
					t.Fatal(err)
				}
			} else {
				collaborator := filepath.Join(t.TempDir(), "collaborator")
				runProwGit(t, "clone", fork, collaborator)
				if err := os.WriteFile(filepath.Join(collaborator, "fork-edit.txt"), []byte("concurrent fork edit\n"), 0644); err != nil {
					t.Fatal(err)
				}
				runProwGit(t, "-C", collaborator, "add", ".")
				runProwGit(t, "-C", collaborator, "commit", "-m", "concurrent fork edit")
				concurrentCommit = runProwGit(t, "-C", collaborator, "rev-parse", "HEAD")
				gitCommand = func(args ...string) (string, error) {
					if len(args) > 2 && args[2] == "push" {
						runProwGit(t, "-C", collaborator, "push", fork, "HEAD:refs/heads/"+branch)
					}
					return _gitCommand(args...)
				}
			}
			if err := r.forkProwJobs(); err == nil {
				t.Fatal("git failure must stop config publishing")
			}
			if name == "inaccessible fork ref" {
				if runProwGit(t, "-C", r.infraDir, "branch", "--show-current") != "main" {
					t.Fatal("failed ref lookup was treated as an absent branch")
				}
			} else if runProwGit(t, "--git-dir", fork, "rev-parse", branch) != concurrentCommit {
				t.Fatal("concurrent fork edit was overwritten")
			}
			if runProwGit(t, "ls-remote", "--heads", upstream, "refs/heads/"+branch) != "" {
				t.Fatal("git failure modified upstream")
			}
		})
	}
}

func TestNewTag(t *testing.T) {

	expectedGitCommands := []string{}

	r := standardSetup()
	defer standardCleanup(&r)
	r.tag = "v0.2.0"
	r.tagBranch = "release-0.2"

	r.dryRun = false
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [clone https://fake-token@github.com/fake-org/fake-repo.git %s/fake-org/https-fake-repo]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo config user.name fake-user]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo config user.email fake-email@fake.fake]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo checkout release-0.2]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo pull origin release-0.2]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo tag -s v0.2.0 -F %s/fake-org/https-fake-repo/v0.2.0-release-notes.txt]", r.cacheDir, r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/fake-org/https-fake-repo push https://fake-token@github.com/fake-org/fake-repo.git v0.2.0]", r.cacheDir))

	seenGitCommands := []string{}
	// override gitCommand with mock function
	gitCommand = func(arg ...string) (string, error) {
		seenGitCommands = append(seenGitCommands, fmt.Sprintf("git %s", arg))
		return "", nil
	}

	err := r.cutNewTag()
	if err != nil {
		t.Errorf("got unexpected error %s", err)
	} else if len(expectedGitCommands) != len(seenGitCommands) {
		t.Errorf("got unexpected git commands")
	}

	for i, entry := range seenGitCommands {
		if entry != expectedGitCommands[i] {
			t.Errorf("expected command %s and got %s", expectedGitCommands[i], entry)
		}
	}
}

func TestGenerateReleaseNotesSkipsBots(t *testing.T) {
	r := standardSetup()
	defer standardCleanup(&r)

	r.tag = "v0.2.0"
	r.tagBranch = "release-0.2"
	r.previousTag = "v0.1.0"

	err := os.MkdirAll(r.repoDir, 0755)
	if err != nil {
		t.Fatalf("failed to create repoDir: %s", err)
	}

	gitCommand = func(arg ...string) (string, error) {
		for _, a := range arg {
			switch a {
			case "log":
				return "abc1234 some commit message\n", nil
			case "shortlog":
				return "     5\tAlice <alice@example.com>\n     3\tkubevirt-bot <bot@kubevirt.io>\n     2\tBob <bob@example.com>\n     1\tkubevirt-prow <prow@kubevirt.io>\n", nil
			case "diff":
				return "10 files changed, 200 insertions(+), 50 deletions(-)", nil
			}
		}
		return "", nil
	}

	err = r.generateReleaseNotes()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	content, err := os.ReadFile(r.releaseNotesFile)
	if err != nil {
		t.Fatalf("failed to read release notes: %s", err)
	}

	notes := string(content)
	if strings.Contains(notes, "kubevirt-bot") {
		t.Error("release notes should not contain kubevirt-bot")
	}
	if strings.Contains(notes, "kubevirt-prow") {
		t.Error("release notes should not contain kubevirt-prow")
	}
	if !strings.Contains(notes, "Alice") {
		t.Error("release notes should contain Alice")
	}
	if !strings.Contains(notes, "Bob") {
		t.Error("release notes should contain Bob")
	}
}

func TestDoesBranchExist(t *testing.T) {
	// the lookup must use the requested branch, not newBranch: the
	// --add-rpm-urls-job dispatch runs with an empty newBranch
	r := standardSetup()
	defer standardCleanup(&r)
	r.newBranch = ""

	exists, err := r.doesBranchExist("release-0.1")
	if err != nil {
		t.Fatalf("got unexpected error %s", err)
	}
	if !exists {
		t.Error("expected release-0.1 to exist")
	}

	exists, err = r.doesBranchExist("release-0.2")
	if err != nil {
		t.Fatalf("got unexpected error %s", err)
	}
	if exists {
		t.Error("expected release-0.2 to not exist")
	}
}

func TestIsValidReleaseBranch(t *testing.T) {
	var tests = []struct {
		branch  string
		isValid bool
	}{
		{branch: "release-1.10", isValid: true},
		{branch: "release-0.30", isValid: true},
		{branch: "main", isValid: false},
		{branch: "release-1", isValid: false},
		{branch: "release-1.10.1", isValid: false},
		{branch: "release-1.10-hotfix", isValid: false},
	}
	for _, tc := range tests {
		if got := isValidReleaseBranch(tc.branch); got != tc.isValid {
			t.Errorf("expected %q -> %t, got %t", tc.branch, tc.isValid, got)
		}
	}
}

func TestRPMURLsPresubmit(t *testing.T) {
	const image = "quay.io/kubevirtci/bootstrap:v20990101-abcdef12"
	job, jobText, err := rpmURLsPresubmit("kubevirt", "kubevirt", "1.10", image)
	if err != nil {
		t.Fatalf("got unexpected error %s", err)
	}
	if job.Name != "pull-kubevirt-verify-rpm-urls-1.10" {
		t.Errorf("unexpected job name %s", job.Name)
	}
	// the branch must be the plain branch name: the rehearse plugin
	// passes it verbatim to git as the ref to clone, so it must be a
	// valid refspec, not an anchored regex
	if !slices.Equal(job.Branches, []string{"release-1.10"}) {
		t.Errorf("unexpected job branches %v", job.Branches)
	}
	if job.Context != "pull-kubevirt-verify-rpm-urls" {
		t.Errorf("unexpected job context %s", job.Context)
	}
	if !job.AlwaysRun {
		t.Error("expected always_run to be true")
	}
	if job.RunIfChanged != "" {
		t.Errorf("unexpected run_if_changed %s", job.RunIfChanged)
	}
	// the job must run make rpm-mirror-checker in the bootstrap image
	// it was rendered with
	if job.Spec == nil || len(job.Spec.Containers) != 1 || job.Spec.Containers[0].Image != image {
		t.Errorf("expected the job to run in %s, got %+v", image, job.Spec)
	}
	// the rendered item must be a single insertable yaml list entry
	if !strings.HasPrefix(jobText, "- always_run: true") {
		t.Errorf("unexpected rendered job text:\n%s", jobText)
	}
}

func TestAppendRPMURLsJob(t *testing.T) {
	const (
		org     = "fake-org"
		repo    = "fake-repo"
		version = "0.2"
	)
	var tests = []struct {
		name              string
		existingConfig    string
		expectedErr       string // expected substring of the error, empty when no error is expected
		expectedJobs      map[string][]string
		expectedPeriodics []string
		expectedContains  []string // substrings expected in the updated config text
		// expected image of the appended rpm urls job, asserted when
		// non-empty
		expectedBootstrapImage string
	}{
		{
			name: "appends to the existing job list",
			existingConfig: `presubmits:
  fake-org/fake-repo:
  # TODO: operational comment that must survive the append
  - name: pull-fake-repo-e2e
    branches:
    - release-0.2
    always_run: true
    spec:
      containers:
      - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12
`,
			expectedJobs: map[string][]string{
				"fake-org/fake-repo": {"pull-fake-repo-e2e", "pull-fake-repo-verify-rpm-urls-0.2"},
			},
			// the append must not rewrite the existing config: comments
			// and formatting survive
			expectedContains: []string{
				"# TODO: operational comment that must survive the append",
				"    always_run: true\n",
			},
			expectedBootstrapImage: "quay.io/kubevirtci/bootstrap:v20990101-abcdef12",
		},
		{
			// the job inherits the bootstrap image of the repo's
			// presubmits instead of pinning a tag that no automation
			// updates
			name: "inherits the bootstrap image from the repo's presubmits",
			existingConfig: `presubmits:
  fake-org/fake-repo:
  - name: pull-fake-repo-e2e
    branches:
    - release-0.2
    always_run: true
    spec:
      containers:
      - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12
`,
			expectedJobs: map[string][]string{
				"fake-org/fake-repo": {"pull-fake-repo-e2e", "pull-fake-repo-verify-rpm-urls-0.2"},
			},
			expectedBootstrapImage: "quay.io/kubevirtci/bootstrap:v20990101-abcdef12",
		},
		{
			// the job is inserted before the next repo key
			name: "inserts the job before the next repo key",
			existingConfig: `presubmits:
  fake-org/fake-repo:
  - name: pull-fake-repo-e2e
    always_run: true
    spec:
      containers:
      - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12
  other-org/other-repo:
  - name: pull-other-repo-e2e
    always_run: true
`,
			expectedJobs: map[string][]string{
				"fake-org/fake-repo":   {"pull-fake-repo-e2e", "pull-fake-repo-verify-rpm-urls-0.2"},
				"other-org/other-repo": {"pull-other-repo-e2e"},
			},
		},
		{
			// the job is inserted before the top-level key that follows
			// the presubmits section
			name: "inserts the job before the top-level key that follows the presubmits section",
			existingConfig: `presubmits:
  fake-org/fake-repo:
  - name: pull-fake-repo-e2e
    always_run: true
    spec:
      containers:
      - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12
periodics:
- name: periodic-job
  interval: 1h
`,
			expectedJobs: map[string][]string{
				"fake-org/fake-repo": {"pull-fake-repo-e2e", "pull-fake-repo-verify-rpm-urls-0.2"},
			},
			// the periodic must survive the append untouched
			expectedPeriodics: []string{"periodic-job"},
			expectedContains:  []string{"periodics:\n- name: periodic-job\n"},
		},
		{
			name: "adds a new repo key when the config has presubmits for other repos only",
			existingConfig: `presubmits:
  other-org/other-repo:
  - name: pull-other-repo-e2e
    branches:
    - release-0.2
    always_run: true
    spec:
      containers:
      - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12
`,
			expectedJobs: map[string][]string{
				"other-org/other-repo": {"pull-other-repo-e2e"},
				"fake-org/fake-repo":   {"pull-fake-repo-verify-rpm-urls-0.2"},
			},
			// with no presubmits of its own the repo inherits the
			// bootstrap image of the other repos in the config
			expectedBootstrapImage: "quay.io/kubevirtci/bootstrap:v20990101-abcdef12",
		},
		{
			// repository keys indented by four spaces are valid yaml: the
			// new repo key must match the indentation of its siblings
			name: "matches the indentation of the sibling repo keys",
			existingConfig: `presubmits:
    other-org/other-repo:
    - name: pull-other-repo-e2e
      branches:
      - release-0.2
      always_run: true
      spec:
        containers:
        - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12
`,
			expectedJobs: map[string][]string{
				"fake-org/fake-repo": {"pull-fake-repo-verify-rpm-urls-0.2"},
			},
			expectedContains: []string{"\n    fake-org/fake-repo:\n"},
		},
		{
			// a flow style job list cannot be extended with a block style
			// item: the append must refuse instead of producing invalid
			// yaml
			name:           "refuses to extend a flow style job list",
			existingConfig: "presubmits:\n  fake-org/fake-repo: [{name: pull-fake-repo-e2e, spec: {containers: [{image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12}]}}]\n",
			expectedErr:    "flow style",
		},
		{
			// the config-forker emits empty sections as flow mappings when
			// a repo has no forkable presubmits: with no presubmit at all
			// there is no bootstrap image to inherit, and the append must
			// refuse instead of pinning an image that nothing keeps
			// current
			name:           "refuses to append without a bootstrap image to inherit",
			existingConfig: "periodics: []\npostsubmits: {}\npresubmits: {}\n",
			expectedErr:    "no presubmit runs in a quay.io/kubevirtci/bootstrap:* image",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "presubmits.yaml")
			if err := os.WriteFile(configPath, []byte(tc.existingConfig), 0666); err != nil {
				t.Fatalf("failed to seed the job config: %s", err)
			}

			added, err := appendRPMURLsJob(configPath, org, repo, version)
			if tc.expectedErr != "" {
				if err == nil {
					t.Fatal("expected an error")
				}
				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Errorf("expected the error to contain %q, got %s", tc.expectedErr, err)
				}
				if added {
					t.Error("expected no job to be added")
				}
				// the config must be left untouched
				updated, readErr := os.ReadFile(configPath)
				if readErr != nil {
					t.Fatalf("failed to read the job config: %s", readErr)
				}
				if string(updated) != tc.existingConfig {
					t.Errorf("expected the config to be left untouched, got:\n%s", updated)
				}
				return
			}
			if err != nil {
				t.Fatalf("got unexpected error %s", err)
			}
			if !added {
				t.Error("expected the job to be added")
			}

			// appending again must be a no-op
			added, err = appendRPMURLsJob(configPath, org, repo, version)
			if err != nil {
				t.Fatalf("got unexpected error on second call %s", err)
			}
			if added {
				t.Error("expected the second call to be a no-op")
			}

			jobConfig, err := config.ReadJobConfig(configPath)
			if err != nil {
				t.Fatalf("failed to read the updated job config: %s", err)
			}
			for repoKey, expectedNames := range tc.expectedJobs {
				names := []string{}
				for _, presubmit := range jobConfig.PresubmitsStatic[repoKey] {
					names = append(names, presubmit.Name)
				}
				if !slices.Equal(names, expectedNames) {
					t.Errorf("expected jobs %v for %s, got %v", expectedNames, repoKey, names)
				}
			}
			if tc.expectedBootstrapImage != "" {
				image := ""
				for _, presubmit := range jobConfig.PresubmitsStatic[org+"/"+repo] {
					if presubmit.Name == "pull-fake-repo-verify-rpm-urls-0.2" &&
						presubmit.Spec != nil && len(presubmit.Spec.Containers) == 1 {
						image = presubmit.Spec.Containers[0].Image
					}
				}
				if image != tc.expectedBootstrapImage {
					t.Errorf("expected the rpm urls job to run in %s, got %s", tc.expectedBootstrapImage, image)
				}
			}
			if len(tc.expectedPeriodics) > 0 {
				names := []string{}
				for _, periodic := range jobConfig.Periodics {
					names = append(names, periodic.Name)
				}
				if !slices.Equal(names, tc.expectedPeriodics) {
					t.Errorf("expected periodics %v, got %v", tc.expectedPeriodics, names)
				}
			}

			updated, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read the updated job config: %s", err)
			}
			for _, substring := range tc.expectedContains {
				if !strings.Contains(string(updated), substring) {
					t.Errorf("expected the updated config to contain %q, got:\n%s", substring, updated)
				}
			}
		})
	}
}

func TestAddRPMURLsJob(t *testing.T) {
	r := standardSetup()
	defer standardCleanup(&r)

	outputConfig := fmt.Sprintf("github/ci/prow-deploy/files/jobs/%s/%s/%s-presubmits-%s.yaml", r.org, r.repo, r.repo, "0.2")
	fullOutputConfig := filepath.Join(r.infraDir, outputConfig)
	if err := os.MkdirAll(filepath.Dir(fullOutputConfig), 0755); err != nil {
		t.Fatalf("failed to create the job config dir: %s", err)
	}
	existingConfig := []byte(`presubmits:
  fake-org/fake-repo:
  - name: pull-fake-repo-e2e
    branches:
    - release-0.2
    always_run: true
    spec:
      containers:
      - image: quay.io/kubevirtci/bootstrap:v20990101-abcdef12
`)
	if err := os.WriteFile(fullOutputConfig, existingConfig, 0666); err != nil {
		t.Fatalf("failed to seed the job config: %s", err)
	}

	seenGitCommands := []string{}
	originalGitCommand := gitCommand
	t.Cleanup(func() { gitCommand = originalGitCommand })
	gitCommand = func(arg ...string) (string, error) {
		seenGitCommands = append(seenGitCommands, fmt.Sprintf("git %s", arg))
		return "", nil
	}

	if err := r.addRPMURLsJob("release-0.2"); err != nil {
		t.Fatalf("got unexpected error %s", err)
	}

	jobConfig, err := config.ReadJobConfig(fullOutputConfig)
	if err != nil {
		t.Fatalf("failed to read the updated job config: %s", err)
	}
	names := []string{}
	for _, presubmit := range jobConfig.PresubmitsStatic["fake-org/fake-repo"] {
		names = append(names, presubmit.Name)
	}
	expectedNames := []string{"pull-fake-repo-e2e", "pull-fake-repo-verify-rpm-urls-0.2"}
	if !slices.Equal(names, expectedNames) {
		t.Errorf("expected jobs %v, got %v", expectedNames, names)
	}

	// the job is committed on a dedicated branch, and only the job
	// config is committed: unrelated staged files must not end up in it
	expectedCommit := fmt.Sprintf("git [-C %s commit -s -m add rpm urls presubmit job for branch release-0.2 -- %s]", r.infraDir, outputConfig)
	sawCommit := false
	for _, command := range seenGitCommands {
		if command == expectedCommit {
			sawCommit = true
		}
	}
	if !sawCommit {
		t.Errorf("expected command %s, got %v", expectedCommit, seenGitCommands)
	}

	// adding the job again must be a no-op
	commandsBefore := len(seenGitCommands)
	if err := r.addRPMURLsJob("release-0.2"); err != nil {
		t.Fatalf("got unexpected error on second call: %s", err)
	}
	for _, name := range seenGitCommands[commandsBefore:] {
		if strings.Contains(name, "commit") {
			t.Errorf("second call must not commit anything, got %s", name)
		}
	}
}

func TestConfigureReleaseJob(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "convertJobConfig-*")
	if err != nil {
		t.Errorf("got unexpected error %s", err)
		return
	}
	defer func(path string) {
		err := os.RemoveAll(path)
		if err != nil {
			log.Warnf("failed to remove temp dir %s", tempDir)
		}
	}(tempDir)

	tempOutputFile := filepath.Join(tempDir, "kubevirt-presubmits-1.6.yaml")
	err = configureReleaseJob(
		"testdata/jobconfig/kubevirt-presubmits-1.6.yaml",
		tempOutputFile,
	)
	if err != nil {
		t.Errorf("got unexpected error %s", err)
		return
	}
	jobConfig, err := config.ReadJobConfig(tempOutputFile)
	if err != nil {
		t.Errorf("got unexpected error %s", err)
	}
	for _, presubmits := range jobConfig.PresubmitsStatic {
		for _, presubmit := range presubmits {
			if presubmit.RunBeforeMerge {
				t.Errorf("got unexpected RunBeforeMerge on job %s", presubmit.Name)
			}
			if _, ok := presubmit.Labels["preset-bazel-cache"]; ok {
				t.Errorf("got unexpected Label \"preset-bazel-cache\" on job %s", presubmit.Name)
			}
		}
	}
	if err != nil {
		t.Errorf("got unexpected error %s", err)
	}
}
