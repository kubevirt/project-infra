package main

import (
	"fmt"
	"os"
	"path/filepath"
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
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/kubevirt/https-project-infra checkout -b fake-org_fake-repo_release-0.2_configs]", r.cacheDir))
	expectedGitCommands = append(expectedGitCommands, fmt.Sprintf("git [-C %s/kubevirt/https-project-infra pull origin fake-org_fake-repo_release-0.2_configs]", r.cacheDir))
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
	job, jobText, err := rpmURLsPresubmit("kubevirt", "kubevirt", "1.10")
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
	// the rendered item must be a single insertable yaml list entry
	if !strings.HasPrefix(jobText, "- always_run: true") {
		t.Errorf("unexpected rendered job text:\n%s", jobText)
	}
}

func TestAppendRPMURLsJob(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "presubmits.yaml")
	existingConfig := []byte(`presubmits:
  fake-org/fake-repo:
  # TODO: operational comment that must survive the append
  - name: pull-fake-repo-e2e
    branches:
    - release-0.2
    always_run: true
`)
	if err := os.WriteFile(configPath, existingConfig, 0666); err != nil {
		t.Fatalf("failed to seed the job config: %s", err)
	}

	added, err := appendRPMURLsJob(configPath, "fake-org", "fake-repo", "0.2")
	if err != nil {
		t.Fatalf("got unexpected error %s", err)
	}
	if !added {
		t.Error("expected the job to be added")
	}

	// appending again must be a no-op
	added, err = appendRPMURLsJob(configPath, "fake-org", "fake-repo", "0.2")
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
	names := []string{}
	for _, presubmit := range jobConfig.PresubmitsStatic["fake-org/fake-repo"] {
		names = append(names, presubmit.Name)
	}
	expectedNames := []string{"pull-fake-repo-e2e", "pull-fake-repo-verify-rpm-urls-0.2"}
	if !slices.Equal(names, expectedNames) {
		t.Errorf("expected jobs %v, got %v", expectedNames, names)
	}

	// the append must not rewrite the existing config: comments and
	// formatting survive
	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read the updated job config: %s", err)
	}
	if !strings.Contains(string(updated), "# TODO: operational comment that must survive the append") {
		t.Errorf("expected the config comment to survive the append, got:\n%s", updated)
	}
	if !strings.Contains(string(updated), "    always_run: true\n") {
		t.Errorf("expected the existing config formatting to survive the append, got:\n%s", updated)
	}
}

func TestAppendRPMURLsJobMissingRepo(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "presubmits.yaml")
	existingConfig := []byte(`presubmits:
  other-org/other-repo:
  - name: pull-other-repo-e2e
    branches:
    - release-0.2
    always_run: true
`)
	if err := os.WriteFile(configPath, existingConfig, 0666); err != nil {
		t.Fatalf("failed to seed the job config: %s", err)
	}

	added, err := appendRPMURLsJob(configPath, "fake-org", "fake-repo", "0.2")
	if err != nil {
		t.Fatalf("got unexpected error %s", err)
	}
	if !added {
		t.Error("expected the job to be added")
	}

	jobConfig, err := config.ReadJobConfig(configPath)
	if err != nil {
		t.Fatalf("failed to read the updated job config: %s", err)
	}
	names := []string{}
	for _, presubmit := range jobConfig.PresubmitsStatic["other-org/other-repo"] {
		names = append(names, presubmit.Name)
	}
	if !slices.Equal(names, []string{"pull-other-repo-e2e"}) {
		t.Errorf("expected jobs of the untouched repo %v", names)
	}
	names = []string{}
	for _, presubmit := range jobConfig.PresubmitsStatic["fake-org/fake-repo"] {
		names = append(names, presubmit.Name)
	}
	if !slices.Equal(names, []string{"pull-fake-repo-verify-rpm-urls-0.2"}) {
		t.Errorf("expected jobs of the appended repo %v", names)
	}
}

func TestAppendRPMURLsJobInsertionPoints(t *testing.T) {
	var tests = []struct {
		name           string
		existingConfig string
		expectedJobs   map[string][]string
	}{
		{
			// the job is inserted before the next repo key
			name: "insert before the next repo",
			existingConfig: `presubmits:
  fake-org/fake-repo:
  - name: pull-fake-repo-e2e
    always_run: true
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
			name: "insert before the next top-level key",
			existingConfig: `presubmits:
  fake-org/fake-repo:
  - name: pull-fake-repo-e2e
    always_run: true
periodics:
- name: periodic-job
  interval: 1h
`,
			expectedJobs: map[string][]string{
				"fake-org/fake-repo": {"pull-fake-repo-e2e", "pull-fake-repo-verify-rpm-urls-0.2"},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			configPath := filepath.Join(tempDir, "presubmits.yaml")
			if err := os.WriteFile(configPath, []byte(tc.existingConfig), 0666); err != nil {
				t.Fatalf("failed to seed the job config: %s", err)
			}

			added, err := appendRPMURLsJob(configPath, "fake-org", "fake-repo", "0.2")
			if err != nil {
				t.Fatalf("got unexpected error %s", err)
			}
			if !added {
				t.Fatal("expected the job to be added")
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

			updated, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read the updated job config: %s", err)
			}
			if strings.Contains(tc.existingConfig, "periodic-job") {
				// the periodic must survive the append untouched
				if len(jobConfig.Periodics) != 1 || jobConfig.Periodics[0].Name != "periodic-job" {
					t.Errorf("expected the periodics to survive the append, got %v", jobConfig.Periodics)
				}
				if !strings.Contains(string(updated), "periodics:\n- name: periodic-job\n") {
					t.Errorf("expected the periodics to be untouched, got:\n%s", updated)
				}
			}
		})
	}
}

func TestAppendRPMURLsJobMissingRepoIndentedKeys(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "presubmits.yaml")
	// repository keys indented by four spaces are valid yaml: the new
	// repo key must match the indentation of its siblings
	existingConfig := []byte(`presubmits:
    other-org/other-repo:
    - name: pull-other-repo-e2e
      branches:
      - release-0.2
      always_run: true
`)
	if err := os.WriteFile(configPath, existingConfig, 0666); err != nil {
		t.Fatalf("failed to seed the job config: %s", err)
	}

	added, err := appendRPMURLsJob(configPath, "fake-org", "fake-repo", "0.2")
	if err != nil {
		t.Fatalf("got unexpected error %s", err)
	}
	if !added {
		t.Fatal("expected the job to be added")
	}

	// the result must still be valid yaml
	jobConfig, err := config.ReadJobConfig(configPath)
	if err != nil {
		t.Fatalf("failed to read the updated job config: %s", err)
	}
	names := []string{}
	for _, presubmit := range jobConfig.PresubmitsStatic["fake-org/fake-repo"] {
		names = append(names, presubmit.Name)
	}
	if !slices.Equal(names, []string{"pull-fake-repo-verify-rpm-urls-0.2"}) {
		t.Errorf("expected jobs of the appended repo %v", names)
	}

	updated, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read the updated job config: %s", err)
	}
	if !strings.Contains(string(updated), "\n    fake-org/fake-repo:\n") {
		t.Errorf("expected the new repo key to match the sibling indentation, got:\n%s", updated)
	}
}

func TestAppendRPMURLsJobFlowSequence(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "presubmits.yaml")
	// a flow style job list cannot be extended with a block style item:
	// the append must refuse instead of producing invalid yaml
	existingConfig := []byte("presubmits:\n  fake-org/fake-repo: [{name: pull-fake-repo-e2e}]\n")
	if err := os.WriteFile(configPath, existingConfig, 0666); err != nil {
		t.Fatalf("failed to seed the job config: %s", err)
	}

	_, err := appendRPMURLsJob(configPath, "fake-org", "fake-repo", "0.2")
	if err == nil {
		t.Fatal("expected an error for a flow style job list")
	}
	if !strings.Contains(err.Error(), "flow style") {
		t.Errorf("unexpected error %s", err)
	}

	// the config must be left untouched
	updated, readErr := os.ReadFile(configPath)
	if readErr != nil {
		t.Fatalf("failed to read the job config: %s", readErr)
	}
	if string(updated) != string(existingConfig) {
		t.Errorf("expected the config to be left untouched, got:\n%s", updated)
	}
}

func TestAppendRPMURLsJobEmptyPresubmits(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "presubmits.yaml")
	// the config-forker emits empty sections as flow mappings when a repo
	// has no forkable presubmits
	existingConfig := []byte(`periodics: []
postsubmits: {}
presubmits: {}
`)
	if err := os.WriteFile(configPath, existingConfig, 0666); err != nil {
		t.Fatalf("failed to seed the job config: %s", err)
	}

	added, err := appendRPMURLsJob(configPath, "fake-org", "fake-repo", "0.2")
	if err != nil {
		t.Fatalf("got unexpected error %s", err)
	}
	if !added {
		t.Error("expected the job to be added")
	}

	jobConfig, err := config.ReadJobConfig(configPath)
	if err != nil {
		t.Fatalf("failed to read the updated job config: %s", err)
	}
	names := []string{}
	for _, presubmit := range jobConfig.PresubmitsStatic["fake-org/fake-repo"] {
		names = append(names, presubmit.Name)
	}
	if !slices.Equal(names, []string{"pull-fake-repo-verify-rpm-urls-0.2"}) {
		t.Errorf("expected jobs %v", names)
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
