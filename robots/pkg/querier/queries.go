package querier

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"

	"github.com/google/go-github/github"
)

var SemVerRegex = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)$`)
var SemVerRegexFull = regexp.MustCompile(`^v([0-9]+)\.([0-9]+)\.([0-9]+)(-(alpha|beta|rc)\.[0-9]+)?$`)
var SemVerMajorRegex = regexp.MustCompile(`^[v]?([0-9]+)$`)
var SemVerMinorRegex = regexp.MustCompile(`^[v]?([0-9]+)\.([0-9]+)$`)

func ValidReleases(releases []*github.RepositoryRelease) (validReleases []*github.RepositoryRelease) {
	for _, release := range releases {
		if release.PublishedAt != nil {
			if !SemVerRegex.MatchString(*release.TagName) {
				continue
			}
			validReleases = append(validReleases, release)
		}
	}

	sort.Slice(validReleases, func(i, j int) bool {
		iRel := ParseRelease(validReleases[i])
		jRel := ParseRelease(validReleases[j])
		return iRel.Compare(jRel) > 0
	})
	return validReleases
}

func LatestRelease(releases []*github.RepositoryRelease) *github.RepositoryRelease {
	validReleases := ValidReleases(releases)
	if len(validReleases) == 0 {
		return nil
	}
	return validReleases[0]
}

func LastPatchOf(major uint, minor uint, releases []*github.RepositoryRelease) *github.RepositoryRelease {
	validReleases := ValidReleases(releases)

	for i := 0; i < len(validReleases); i++ {
		matches := SemVerRegex.FindStringSubmatch(*validReleases[i].TagName)
		releaseMajor, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Panicln(err)
		}
		if releaseMajor < int(major) {
			return nil
		}
		if releaseMajor == int(major) {
			releaseMinor, err := strconv.Atoi(matches[2])
			if err != nil {
				log.Panicln(err)
			}
			if releaseMinor < int(minor) {
				return nil
			}
			if releaseMinor == int(minor) {
				return validReleases[i]
			}
		}
	}
	return nil
}

func LastThreeMinor(major uint, releases []*github.RepositoryRelease) (minors []*github.RepositoryRelease) {
	validReleases := ValidReleases(releases)

	if len(validReleases) == 0 {
		return nil
	}

	var lastMinor string

	for i := 0; i < len(validReleases); i++ {
		matches := SemVerRegex.FindStringSubmatch(*validReleases[i].TagName)
		releaseMajor, err := strconv.Atoi(matches[1])
		if err != nil {
			log.Panicln(err)
		}
		if releaseMajor > int(major) {
			continue
		}
		if releaseMajor < int(major) {
			break
		}
		if matches[2] != lastMinor {
			minors = append(minors, validReleases[i])
		}
		if len(minors) == 3 {
			break
		}
		lastMinor = matches[2]
	}
	return minors
}

type SemVer struct {
	Major string
	Minor string
	Patch string
}

func (i *SemVer) MajorInt() int {
	v, _ := strconv.Atoi(i.Major)
	return v
}

func (i *SemVer) MinorInt() int {
	v, _ := strconv.Atoi(i.Minor)
	return v
}

func (i *SemVer) PatchInt() int {
	v, _ := strconv.Atoi(i.Patch)
	return v
}

func (i *SemVer) Compare(j *SemVer) int {
	if i.MajorInt() != j.MajorInt() {
		if i.MajorInt() > j.MajorInt() {
			return 1
		}
		return -1
	}
	if i.MinorInt() != j.MinorInt() {
		if i.MinorInt() > j.MinorInt() {
			return 1
		}
		return -1
	}
	if i.PatchInt() != j.PatchInt() {
		if i.PatchInt() > j.PatchInt() {
			return 1
		}
		return -1
	}
	return 0
}

func (i *SemVer) CompareMajorMinor(j *SemVer) int {
	if i.MajorInt() != j.MajorInt() {
		if i.MajorInt() > j.MajorInt() {
			return 1
		}
		return -1
	}
	if i.MinorInt() != j.MinorInt() {
		if i.MinorInt() > j.MinorInt() {
			return 1
		}
		return -1
	}
	return 0
}

func (i *SemVer) String() string {
	return fmt.Sprintf("%s.%s.%s", i.Major, i.Minor, i.Patch)
}

func ParseRelease(release *github.RepositoryRelease) *SemVer {
	matches := SemVerRegex.FindStringSubmatch(*release.TagName)
	return &SemVer{
		Major: matches[1],
		Minor: matches[2],
		Patch: matches[3],
	}
}

func ParseReleaseFull(release *github.RepositoryRelease) *SemVer {
	matches := SemVerRegexFull.FindStringSubmatch(*release.TagName)
	return &SemVer{
		Major: matches[1],
		Minor: matches[2],
		Patch: matches[3],
	}
}
