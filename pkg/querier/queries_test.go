package querier

import (
	"testing"
	"time"

	"github.com/google/go-github/github"
)

func TestReleaseValidator(t *testing.T) {
	tests := []struct {
		name     string
		given    []*github.RepositoryRelease
		expected []*github.RepositoryRelease
	}{
		{
			name: "releases should be sorted",
			given: []*github.RepositoryRelease{
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v2.3.2", true),
				release("v1.3.4", true),
				release("v1.3.3", true),
				release("v1.3.5", true),
				release("v3.3.1", true),
				release("v3.3.0", true),
				release("v3.3.1", true),
			},
			expected: []*github.RepositoryRelease{
				release("v3.3.1", true),
				release("v3.3.1", true),
				release("v3.3.0", true),
				release("v2.3.2", true),
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v1.3.5", true),
				release("v1.3.4", true),
				release("v1.3.3", true),
			},
		},
		{
			name: "tags should be sorted out",
			given: []*github.RepositoryRelease{
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v2.3.2", true),
				release("v1.3.4", false),
			},
			expected: []*github.RepositoryRelease{
				release("v2.3.2", true),
				release("v2.3.1", true),
				release("v2.3.0", true),
			},
		},
		{
			name: "shold sort out invalid tags",
			given: []*github.RepositoryRelease{
				release("v2.3.1", true),
				release("vv2.3.0", true),
				release("v2.3.3", true),
				release("v1.3.4-rc1", true),
			},
			expected: []*github.RepositoryRelease{
				release("v2.3.3", true),
				release("v2.3.1", true),
			},
		},
		{
			name:     "it should be able to handle no releases",
			given:    []*github.RepositoryRelease{},
			expected: []*github.RepositoryRelease{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processed := ValidReleases(test.given)
			for i, r := range processed {
				if *r.TagName != *test.expected[i].TagName {
					t.Errorf("Expected %s to equal %s", *r.TagName, *test.expected[i].TagName)
				}
			}
		})
	}
}

func TestLatestReleaseFinder(t *testing.T) {
	tests := []struct {
		name     string
		given    []*github.RepositoryRelease
		expected *github.RepositoryRelease
	}{
		{
			name: "the latest release should be found",
			given: []*github.RepositoryRelease{
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v2.3.2", true),
				release("v1.3.4", true),
				release("v1.3.3", true),
				release("v1.3.5", true),
				release("v3.3.1", true),
				release("v3.3.0", true),
				release("v3.3.1", true),
			},
			expected: release("v3.3.1", true),
		},
		{
			name: "not released tags should be ignored",
			given: []*github.RepositoryRelease{
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v2.3.2", true),
				release("v15.3.4", false),
			},
			expected: release("v2.3.2", true),
		},
		{
			name:     "it should be able to handle no releases",
			given:    []*github.RepositoryRelease{},
			expected: nil,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := LatestRelease(test.given)
			if r == nil && r != test.expected {
				t.Errorf("Expect both releases to be nil")
			} else if r != nil && test.expected != nil && *r.TagName != *test.expected.TagName {
				t.Errorf("Expected %s to equal %s", *r.TagName, *test.expected.TagName)
			}
		})
	}
}

func TestLatestPatchOf(t *testing.T) {
	tests := []struct {
		name     string
		given    []*github.RepositoryRelease
		expected *github.RepositoryRelease
	}{
		{
			name: "the latest release should be found",
			given: []*github.RepositoryRelease{
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v2.3.2", true),
				release("v1.3.4", true),
				release("v1.3.3", true),
				release("v1.3.5", true),
				release("v3.3.1", true),
				release("v3.3.0", true),
				release("v3.3.1", true),
			},
			expected: release("v2.3.2", true),
		},
		{
			name: "not released tags should be ignored",
			given: []*github.RepositoryRelease{
				release("v1.3.1", true),
				release("v1.3.1", true),
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v2.3.2", true),
				release("v2.4.2", true),
				release("v2.5.2", true),
				release("v2.3.4", false),
			},
			expected: release("v2.3.2", true),
		},
		{
			name: "it should handle situations without major or minor given",
			given: []*github.RepositoryRelease{
				release("v1.3.1", true),
				release("v2.2.1", true),
			},
			expected: nil,
		},
		{
			name: "the latest release should be found",
			given: []*github.RepositoryRelease{
				release("v2.3.1", true),
				release("v2.3.0", true),
				release("v2.3.2", true),
				release("v1.3.4", true),
				release("v1.3.3", true),
				release("v1.3.5", true),
				release("v3.3.1", true),
				release("v3.3.0", true),
				release("v3.3.1", true),
			},
			expected: release("v2.3.2", true),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := LastPatchOf(2, 3, test.given)
			if r == nil && r != test.expected {
				t.Errorf("Expect both releases to be nil")
			} else if r != nil && test.expected != nil && *r.TagName != *test.expected.TagName {
				t.Errorf("Expected %s to equal %s", *r.TagName, *test.expected.TagName)
			}
		})
	}
}

func TestLastThreeMinorReleases(t *testing.T) {
	tests := []struct {
		name     string
		given    []*github.RepositoryRelease
		expected []*github.RepositoryRelease
	}{
		{
			name: "releases should be sorted",
			given: []*github.RepositoryRelease{
				release("v2.2.1", true),
				release("v2.3.0", true),
				release("v2.4.2", true),
				release("v1.5.4", true),
				release("v1.6.3", true),
				release("v1.7.5", true),
			},
			expected: []*github.RepositoryRelease{
				release("v2.4.2", true),
				release("v2.3.0", true),
				release("v2.2.1", true),
			},
		},
		{
			name:     "it should be able to handle no releases",
			given:    []*github.RepositoryRelease{},
			expected: []*github.RepositoryRelease{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			processed := LastThreeMinor(2, test.given)
			for i, r := range processed {
				if *r.TagName != *test.expected[i].TagName {
					t.Errorf("Expected %s to equal %s", *r.TagName, *test.expected[i].TagName)
				}
			}
		})
	}
}

func release(tagName string, released bool) *github.RepositoryRelease {
	release := &github.RepositoryRelease{
		TagName: &tagName,
	}
	if released {
		release.PublishedAt = &github.Timestamp{Time: time.Now()}
	}
	return release
}
