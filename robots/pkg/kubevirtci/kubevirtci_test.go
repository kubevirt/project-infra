package kubevirtci

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-github/github"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

func TestReadExistingProviders(t *testing.T) {
	tests := []struct {
		name     string
		want     []querier.SemVer
		existing []querier.SemVer
		wantErr  bool
	}{
		{
			name: "should load all existing versions",
			existing: []querier.SemVer{
				newSemVer("2", "2", "4"),
				newSemVer("1", "2", "3"),
				newSemVer("1", "3", "4"),
				newSemVer("1", "4", "5"),
				newSemVer("1", "5", "6"),
			},
			want: []querier.SemVer{
				newSemVer("2", "2", "4"),
				newSemVer("1", "5", "6"),
				newSemVer("1", "4", "5"),
				newSemVer("1", "3", "4"),
				newSemVer("1", "2", "3"),
			},
		},
		{
			name: "should sort this correctly",
			existing: []querier.SemVer{
				newSemVer("1", "2", "1"),
				newSemVer("1", "3", "2"),
				newSemVer("1", "9", "4"),
				newSemVer("1", "16", "5"),
			},
			want: []querier.SemVer{
				newSemVer("1", "16", "5"),
				newSemVer("1", "9", "4"),
				newSemVer("1", "3", "2"),
				newSemVer("1", "2", "1"),
			},
		},
		{
			name: "should ignore strange folders",
			existing: []querier.SemVer{
				newSemVer("v1x", "2", "3"),
				newSemVer("1", "3-multus", "4"),
				newSemVer("v1", "4", "5"),
				newSemVer("1", "5", "6"),
			},
			want: []querier.SemVer{
				newSemVer("1", "5", "6"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "prefix")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(dir)
			createProviderEnv(dir, tt.existing)
			got, err := ReadExistingProviders(dir)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadExistingProviders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadExistingProviders() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func createProviderEnv(dir string, releases []querier.SemVer) {
	for _, release := range releases {
		createRelease(dir, release)
	}
}

func createRelease(dir string, semver querier.SemVer) {
	path := filepath.Join(dir, fmt.Sprintf("%s.%s", semver.Major, semver.Minor))
	err := os.Mkdir(path, os.ModePerm)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(filepath.Join(path, "version"), []byte(fmt.Sprintf("%s.%s.%s", semver.Major, semver.Minor, semver.Patch)), os.ModePerm)
	if err != nil {
		panic(err)
	}
	// emulate discoverable content
	err = ioutil.WriteFile(filepath.Join(path, fmt.Sprintf("%s.%s.%s", semver.Major, semver.Minor, semver.Patch)), []byte(fmt.Sprintf("%s.%s.%s", semver.Major, semver.Minor, semver.Patch)), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func newSemVer(major string, minor string, patch string) querier.SemVer {
	return querier.SemVer{Major: major, Minor: minor, Patch: patch}
}

func TestBumpMinorReleaseOfProvider(t *testing.T) {
	tests := []struct {
		name           string
		upstreamMinors []*github.RepositoryRelease
		existing       []querier.SemVer
		wanted         []querier.SemVer
		wantErr        bool
	}{
		{
			upstreamMinors: []*github.RepositoryRelease{
				release("v1.2.3", true),
				release("v1.3.4", true),
				release("v1.5.6", true),
			},
			existing: []querier.SemVer{
				newSemVer("1", "2", "1"),
				newSemVer("1", "3", "2"),
				newSemVer("1", "4", "4"),
				newSemVer("1", "5", "5"),
			},
			wanted: []querier.SemVer{
				newSemVer("1", "5", "6"),
				newSemVer("1", "4", "4"),
				newSemVer("1", "3", "4"),
				newSemVer("1", "2", "3"),
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "prefix")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(dir)
			createProviderEnv(dir, tt.existing)
			if err := BumpMinorReleaseOfProvider(dir, tt.upstreamMinors); (err != nil) != tt.wantErr {
				t.Errorf("BumpMinorReleaseOfProvider() error = %v, wantErr %v", err, tt.wantErr)
			}
			got, err := ReadExistingProviders(dir)
			if !reflect.DeepEqual(got, tt.wanted) {
				t.Errorf("ReadExistingProviders() got = %v, want %v", got, tt.wanted)
			}
		})
	}
}

func TestEnsureProviderExists(t *testing.T) {
	tests := []struct {
		name     string
		release  *github.RepositoryRelease
		existing []querier.SemVer
		wanted   []querier.SemVer
		wantErr  bool
	}{
		{
			release: release("v1.10.3", true),
			existing: []querier.SemVer{
				newSemVer("1", "2", "1"),
				newSemVer("1", "3", "2"),
				newSemVer("1", "9", "4"),
				newSemVer("1", "16", "5"),
			},
			wanted: []querier.SemVer{
				newSemVer("1", "16", "5"),
				newSemVer("1", "10", "3"),
				newSemVer("1", "9", "4"),
				newSemVer("1", "3", "2"),
				newSemVer("1", "2", "1"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "prefix")
			if err != nil {
				panic(err)
			}
			defer os.RemoveAll(dir)
			createProviderEnv(dir, tt.existing)
			if err := EnsureProviderExists(dir, tt.release); (err != nil) != tt.wantErr {
				t.Errorf("EnsureProviderExists() error = %v, wantErr %v", err, tt.wantErr)
			}
			got, err := ReadExistingProviders(dir)
			if !reflect.DeepEqual(got, tt.wanted) {
				t.Errorf("ReadExistingProviders() got = %v, want %v", got, tt.wanted)
			}
		})
	}
}

func release(tagName string, released bool) *github.RepositoryRelease {
	release := &github.RepositoryRelease{
		TagName: &tagName,
	}
	if released {
		release.PublishedAt = &github.Timestamp{time.Now()}
	}
	return release
}
