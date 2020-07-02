package main

import (
	"testing"
	"time"

	"github.com/google/go-github/v32/github"
)

func standardSetup() releaseData {
	repo := "fake-repo"
	org := "fake-org"

	truePtr := true
	falsePtr := false
	idPtr := int64(0)

	v1Branch := "release-0.1"

	v1 := "v0.1.0"
	v1CreatedAt := &github.Timestamp{
		Time: time.Date(2020, time.January, 1, 7, 0, 0, 0, time.UTC),
	}

	v1RC1 := "v0.1.0-rc.1"
	v1RC1CreatedAt := &github.Timestamp{
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
			TagName:    &v1RC1,
			CreatedAt:  v1RC1CreatedAt,
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
	}

	return r
}

func TestAutoRelease(t *testing.T) {

	var tests = []struct {
		name           string
		now            time.Time
		expectedBranch string
		expectedTag    string
	}{
		{
			name:           "Expect new branch and rc",
			now:            time.Date(2020, time.February, 1, 1, 0, 0, 0, time.UTC),
			expectedBranch: "release-0.2",
			expectedTag:    "v0.2.0-rc.1",
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
		r.now = tc.now

		err := r.autoDetectData("monthly", 7)

		if err != nil {
			t.Errorf("got unexpected error %s", err)
		} else if tc.expectedBranch != r.newBranch {
			t.Errorf("Expected branch [%s] and got [%s]", tc.expectedBranch, r.newBranch)
		} else if tc.expectedTag != r.tag {
			t.Errorf("Expected tag [%s] and got [%s]", tc.expectedTag, r.tag)
		}
	}

}

func TestAutoPromoteRC(t *testing.T) {
	v2RC1 := "v0.2.0-rc.1"
	v2Branch := "release-0.2"
	v2RC1CreatedAt := &github.Timestamp{
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
		r.allReleases = append(r.allReleases, &github.RepositoryRelease{
			TagName:   &v2RC1,
			CreatedAt: v2RC1CreatedAt,
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
		} else if tc.expectPromotion && r.promoteRC != v2RC1 {
			t.Errorf("expected to autopromote RC")
		} else if !tc.expectPromotion && r.promoteRC != "" {
			t.Errorf("did not expect to autopromote RC")
		} else if tc.expectNewRC && r.tag != "v0.2.0-rc.2" {
			t.Errorf("expected new RC")
		} else if !tc.expectNewRC && r.tag != "" {
			t.Errorf("did not expect new RC")
		} else if r.newBranch != "" {
			t.Errorf("did not expect new branch")
		}
	}
}
