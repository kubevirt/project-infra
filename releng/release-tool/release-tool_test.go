package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/google/go-github/v32/github"
)

func standardCleanup(r *releaseData) {
	os.RemoveAll(r.cacheDir)
}

func standardSetup() releaseData {
	repo := "fake-repo"
	org := "fake-org"
	token := "fake-token"
	cacheDir, err := ioutil.TempDir("/tmp", "release-tool-unit-test")
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
