/*
 * Copyright 2021 The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package release

import (
	"github.com/google/go-github/github"

	"kubevirt.io/project-infra/robots/pkg/querier"
)

// GetLatestMinorReleases returns the list of all latest minor releases in descending order.
// Input is expected to be a list of all releases sorted descending
func GetLatestMinorReleases(releases []*querier.SemVer) (latestMinorReleases []*querier.SemVer) {
	for _, release := range releases {
		if len(latestMinorReleases) == 0 || release.Major < latestMinorReleases[len(latestMinorReleases)-1].Major || release.Minor < latestMinorReleases[len(latestMinorReleases)-1].Minor {
			latestMinorReleases = append(latestMinorReleases, release)
		}
	}
	return
}

func Release(version string) *github.RepositoryRelease {
	result := github.RepositoryRelease{}
	result.TagName = &version
	return &result
}

func AsSemVers(releases []*github.RepositoryRelease) []*querier.SemVer {
	semVers := make([]*querier.SemVer, 0, len(releases))
	for _, theRelease := range releases {
		semVers = append(semVers, querier.ParseRelease(theRelease))
	}
	return semVers
}
