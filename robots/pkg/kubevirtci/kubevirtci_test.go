package kubevirtci

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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

func createEnsureProviderExistsEnv(tt struct {
	name            string
	release         *github.RepositoryRelease
	existing        []querier.SemVer
	wanted          []querier.SemVer
	wantedClusterUp []querier.SemVer
	wantErr         bool
}) (string, string) {
	providerDir, err := ioutil.TempDir("", "prefix")
	if err != nil {
		panic(err)
	}
	clusterUpDir, err := ioutil.TempDir("", "prefix")
	if err != nil {
		panic(err)
	}
	createProviderEnv(providerDir, tt.existing)
	createClusterUpEnv(clusterUpDir, tt.existing)
	return providerDir, clusterUpDir
}

func createProviderEnv(dir string, releases []querier.SemVer) {
	for _, release := range releases {
		createRelease(dir, release)
	}
}

func createRelease(dir string, semver querier.SemVer) {
	path := filepath.Join(dir, fmt.Sprintf("%s.%s", semver.Major, semver.Minor))
	err := os.Mkdir(path, os.ModePerm)
	if err != nil && !os.IsExist(err) {
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

func createClusterUpEnv(dir string, releases []querier.SemVer) {
	for _, release := range releases {
		createClusterUp(dir, release)
	}
}

func createClusterUp(dir string, semver querier.SemVer) {
	semVerMinor := fmt.Sprintf("%s.%s", semver.Major, semver.Minor)
	path := filepath.Join(dir, fmt.Sprintf("k8s-%s", semVerMinor))
	err := os.Mkdir(path, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	// emulate doc file with semver content
	err = ioutil.WriteFile(filepath.Join(path, "blah.md"), []byte(fmt.Sprintf("k8s-%s %s.1", semver.Major, semver.Minor)), os.ModePerm)
	if err != nil {
		panic(err)
	}
}

func newSemVer(major string, minor string, patch string) querier.SemVer {
	return querier.SemVer{Major: major, Minor: minor, Patch: patch}
}

func newSemVerMinor(major string, minor string) querier.SemVer {
	return querier.SemVer{Major: major, Minor: minor}
}

func readExistingClusterUpFolders(clusterUpDir string) ([]querier.SemVer, error) {
	semvers := []querier.SemVer{}
	fileinfo, err := ioutil.ReadDir(clusterUpDir)
	if err != nil {
		return nil, err
	}
	for _, file := range fileinfo {
		if file.IsDir() {
			semverPartOfDir := strings.TrimPrefix(file.Name(), "k8s-")
			if SemVerMinorRegex.MatchString(semverPartOfDir) {
				matches := SemVerMinorRegex.FindStringSubmatch(semverPartOfDir)
				semvers = append(semvers, querier.SemVer{Major: matches[1], Minor: matches[2]})
				// TODO check whether the docs contain wrong semvers
			}
		}
	}
	sort.Slice(semvers, func(i, j int) bool {
		return semvers[i].Compare(&semvers[j]) > 0
	})
	return semvers, nil
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

var testsEnsureProviderExists = []struct {
	name            string
	release         *github.RepositoryRelease
	existing        []querier.SemVer
	wanted          []querier.SemVer
	wantedClusterUp []querier.SemVer
	wantErr         bool
}{
	{
		name:    "expect a v1.10 provider to be added",
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
		wantedClusterUp: []querier.SemVer{
			newSemVerMinor("1", "16"),
			newSemVerMinor("1", "10"),
			newSemVerMinor("1", "9"),
			newSemVerMinor("1", "3"),
			newSemVerMinor("1", "2"),
		},
	},
	{
		name:    "expect the latest 1.16 provider not to be updated from 1.16.5 to 1.16.7 (already done by ensureLatestThreeMinor)",
		release: release("v1.16.7", true),
		existing: []querier.SemVer{
			newSemVer("1", "2", "1"),
			newSemVer("1", "3", "2"),
			newSemVer("1", "9", "4"),
			newSemVer("1", "16", "5"),
		},
		wanted: []querier.SemVer{
			newSemVer("1", "16", "5"),
			newSemVer("1", "9", "4"),
			newSemVer("1", "3", "2"),
			newSemVer("1", "2", "1"),
		},
		wantedClusterUp: []querier.SemVer{
			newSemVerMinor("1", "16"),
			newSemVerMinor("1", "9"),
			newSemVerMinor("1", "3"),
			newSemVerMinor("1", "2"),
		},
	},
}

func TestEnsureProviderExists(t *testing.T) {
	for _, tt := range testsEnsureProviderExists {
		t.Run(tt.name, func(t *testing.T) {
			providerDir, clusterUpDir := createEnsureProviderExistsEnv(tt)
			defer os.RemoveAll(providerDir)
			defer os.RemoveAll(clusterUpDir)
			if err := EnsureProviderExists(providerDir, clusterUpDir, tt.release); (err != nil) != tt.wantErr {
				t.Errorf("EnsureProviderExists() error = %v, wantErr %v", err, tt.wantErr)
			}
			got, err := ReadExistingProviders(providerDir)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, tt.wanted) {
				t.Errorf("ReadExistingProviders() got = %v, want %v", got, tt.wanted)
			}
		})
	}
}

func TestEnsureProviderExists_CopiesClusterUpDir(t *testing.T) {
	for _, tt := range testsEnsureProviderExists {
		t.Run(tt.name, func(t *testing.T) {
			providerDir, clusterUpDir := createEnsureProviderExistsEnv(tt)
			defer os.RemoveAll(providerDir)
			defer os.RemoveAll(clusterUpDir)
			if err := EnsureProviderExists(providerDir, clusterUpDir, tt.release); (err != nil) != tt.wantErr {
				t.Errorf("EnsureProviderExists() error = %v, wantErr %v", err, tt.wantErr)
			}
			got, err := readExistingClusterUpFolders(clusterUpDir)
			if err != nil {
				panic(err)
			}
			if !reflect.DeepEqual(got, tt.wantedClusterUp) {
				t.Errorf("ReadExistingProviders() got = %v, want %v", got, tt.wantedClusterUp)
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

type testScenario struct {
	dirsBefore         []string
	clusterUpDirsAfter []string
	providerDirsAfter  []string
}

func TestDropUnsupportedProviders(t *testing.T) {
	type args struct {
		scenario          testScenario
		providerDir       string
		clusterUpDir      string
		supportedReleases []*github.RepositoryRelease
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "no supported releases provokes error",
			args: args{
				scenario: testScenario{
					dirsBefore: []string{
						"cluster-provision/k8s/1.19",
						"cluster-provision/k8s/1.20",
						"cluster-provision/k8s/1.21",
						"cluster-up/cluster/k8s-1.19",
						"cluster-up/cluster/k8s-1.20",
						"cluster-up/cluster/k8s-1.21",
					},
				},
				providerDir:       "cluster-provision/k8s",
				clusterUpDir:      "cluster-up/cluster",
				supportedReleases: []*github.RepositoryRelease{},
			},
			wantErr: true,
		},
		{
			name: "only supported providers left",
			args: args{
				scenario: testScenario{
					dirsBefore: []string{
						"cluster-provision/k8s/1.19",
						"cluster-provision/k8s/1.20",
						"cluster-provision/k8s/1.21",
						"cluster-up/cluster/k8s-1.19",
						"cluster-up/cluster/k8s-1.20",
						"cluster-up/cluster/k8s-1.21",
					},
					providerDirsAfter: []string{
						"1.19",
						"1.20",
						"1.21",
					},
					clusterUpDirsAfter: []string{
						"k8s-1.19",
						"k8s-1.20",
						"k8s-1.21",
					},
				},
				providerDir:  "cluster-provision/k8s",
				clusterUpDir: "cluster-up/cluster",
				supportedReleases: []*github.RepositoryRelease{
					{
						TagName: strPointer("v1.19.3"),
					},
					{
						TagName: strPointer("v1.20.2"),
					},
					{
						TagName: strPointer("v1.21.1"),
					},
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported providers should get deleted",
			args: args{
				scenario: testScenario{
					dirsBefore: []string{
						"cluster-provision/k8s/1.19",
						"cluster-provision/k8s/1.20",
						"cluster-provision/k8s/1.21",
						"cluster-up/cluster/k8s-1.19",
						"cluster-up/cluster/k8s-1.20",
						"cluster-up/cluster/k8s-1.21",
					},
					providerDirsAfter: []string{
						"1.20",
						"1.21",
					},
					clusterUpDirsAfter: []string{
						"k8s-1.20",
						"k8s-1.21",
					},
				},
				providerDir:  "cluster-provision/k8s",
				clusterUpDir: "cluster-up/cluster",
				supportedReleases: []*github.RepositoryRelease{
					{
						TagName: strPointer("v1.20.2"),
					},
					{
						TagName: strPointer("v1.21.1"),
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "dropProviders-")
			panicOnErr(err)
			defer os.RemoveAll(tempDir)
			for _, dir := range tt.args.scenario.dirsBefore {
				panicOnErr(mkDirs(path.Join(tempDir, dir)))
			}

			targetClusterUpDir := path.Join(tempDir, tt.args.clusterUpDir)
			targetProviderDir := path.Join(tempDir, tt.args.providerDir)
			err = DropUnsupportedProviders(targetProviderDir, targetClusterUpDir, tt.args.supportedReleases)
			if (err != nil) != tt.wantErr {
				t.Fatalf("DropUnsupportedProviders() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil {
				checkForSuperfluousDirEntries(t, err, targetClusterUpDir, tt.args.scenario.clusterUpDirsAfter)
				checkForRequiredDirEntries(t, targetClusterUpDir, tt.args.scenario.clusterUpDirsAfter)
				checkForSuperfluousDirEntries(t, err, targetProviderDir, tt.args.scenario.providerDirsAfter)
				checkForRequiredDirEntries(t, targetProviderDir, tt.args.scenario.providerDirsAfter)
			}
		})
	}
}

func checkForSuperfluousDirEntries(t *testing.T, err error, pathToCheck string, listOfEntriesThatShouldExist []string) {
	dirEntries, err := os.ReadDir(pathToCheck)
	for _, entry := range dirEntries {
		found := false
		for _, clusterUpDir := range listOfEntriesThatShouldExist {
			if clusterUpDir == entry.Name() {
				found = true
			}
		}
		if !found {
			t.Errorf("DropUnsupportedProviders() error = dir %v should not exist", entry.Name())
		}
	}
}

func checkForRequiredDirEntries(t *testing.T, pathToCheck string, requiredEntries []string) {
	for _, entry := range requiredEntries {
		pathShouldExist := path.Join(pathToCheck, entry)
		_, err := os.Stat(pathShouldExist)
		if os.IsNotExist(err) {
			t.Errorf("DropUnsupportedProviders() error = dir %v should exist", pathShouldExist)
			continue
		}
		panicOnErr(err)
	}
}

func mkDirs(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		parent, _ := path.Split(dir)
		parent = strings.TrimSuffix(parent, "/")
		err := mkDirs(parent)
		if err != nil {
			return err
		}
		return os.Mkdir(dir, 0777)
	}
	return nil
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func strPointer(input string) *string {
	return &input
}
