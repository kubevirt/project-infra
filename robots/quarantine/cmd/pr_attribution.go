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
 * Copyright the KubeVirt Authors.
 *
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/pkg/searchci"
)

const maxProwJobBytes = 8 << 20

var (
	directPullPattern = regexp.MustCompile(`pr-logs/pull/([^/]+)/(\d+)/`)
	prowJobClient     = &http.Client{Timeout: 15 * time.Second}

	// fetchProwJob reads a build's prowjob.json. Tests replace it.
	fetchProwJob = func(rawURL string) ([]byte, error) {
		resp, err := prowJobClient.Get(rawURL)
		if err != nil {
			return nil, fmt.Errorf("could not fetch prowjob %q: %w", rawURL, err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("could not fetch prowjob %q: %s", rawURL, resp.Status)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, maxProwJobBytes))
		if err != nil {
			return nil, fmt.Errorf("could not read prowjob %q: %w", rawURL, err)
		}
		return body, nil
	}

	// pullsForBatchBuild returns the pull requests exercised by a Tide batch build.
	// Tests replace it.
	pullsForBatchBuild = defaultPullsForBatchBuild
)

type prowJobPulls struct {
	Spec struct {
		Refs struct {
			Pulls []struct {
				Number int `json:"number"`
			} `json:"pulls"`
		} `json:"refs"`
	} `json:"spec"`
}

type buildAttribution struct {
	prs      map[int]struct{}
	resolved bool
}

func defaultPullsForBatchBuild(buildURL string) ([]int, error) {
	jobURL, err := prowJobURL(buildURL)
	if err != nil {
		return nil, err
	}
	body, err := fetchProwJob(jobURL)
	if err != nil {
		return nil, err
	}
	return pullNumbersFromProwJob(body)
}

func prowJobURL(buildURL string) (string, error) {
	parsed, err := url.Parse(buildURL)
	if err != nil {
		return "", fmt.Errorf("could not parse build URL %q: %w", buildURL, err)
	}
	path := strings.Trim(parsed.Path, "/")
	switch {
	case strings.Contains(path, "view/gs/"):
		path = path[strings.Index(path, "view/gs/")+len("view/gs/"):]
	case strings.Contains(path, "view/gcs/"):
		path = path[strings.Index(path, "view/gcs/")+len("view/gcs/"):]
	case parsed.Host == "storage.googleapis.com":
	default:
		return "", fmt.Errorf("build URL %q is not a Prow GCS view", buildURL)
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return "", fmt.Errorf("build URL %q has no GCS object path", buildURL)
	}
	if strings.HasSuffix(path, "/prowjob.json") {
		return "https://storage.googleapis.com/" + path, nil
	}
	return "https://storage.googleapis.com/" + path + "/prowjob.json", nil
}

func pullNumbersFromProwJob(body []byte) ([]int, error) {
	var job prowJobPulls
	if err := json.Unmarshal(body, &job); err != nil {
		return nil, fmt.Errorf("could not decode prowjob: %w", err)
	}
	var numbers []int
	for _, pull := range job.Spec.Refs.Pulls {
		if pull.Number > 0 {
			numbers = append(numbers, pull.Number)
		}
	}
	return numbers, nil
}

func directPullNumber(buildURL string) (int, bool) {
	match := directPullPattern.FindStringSubmatch(buildURL)
	if match == nil || match[1] == "batch" {
		return 0, false
	}
	number, err := strconv.Atoi(match[2])
	if err != nil || number <= 0 {
		return 0, false
	}
	return number, true
}

func isTideBatchBuild(buildURL string) bool {
	return strings.Contains(buildURL, "pr-logs/pull/batch/")
}

// attributeBuild maps a search.ci build to the pull requests it ran.
// A direct presubmit resolves to that PR. A Tide batch resolves to spec.refs.pulls.
// Unresolved builds are not independent corroboration.
func attributeBuild(buildURL string) (map[int]struct{}, bool) {
	if number, ok := directPullNumber(buildURL); ok {
		return map[int]struct{}{number: {}}, true
	}
	if !isTideBatchBuild(buildURL) {
		log.Warnf("cannot attribute job build %q to a pull request", buildURL)
		return nil, false
	}
	numbers, err := pullsForBatchBuild(buildURL)
	if err != nil {
		log.Warnf("cannot resolve Tide batch pulls for %q: %v", buildURL, err)
		return nil, false
	}
	pulls := make(map[int]struct{}, len(numbers))
	for _, number := range numbers {
		if number > 0 {
			pulls[number] = struct{}{}
		}
	}
	if len(pulls) == 0 {
		log.Warnf("Tide batch %q lists no pull requests", buildURL)
		return nil, false
	}
	return pulls, true
}

// impactDominatedBySinglePR reports whether the recent failures in an impact
// collapse to one pull request. maxAge is the same window as the recent-failure
// filter. minIndependent is how many attributed failures must omit the pull
// request that appears most often. Values below 1 are treated as 1.
//
// Failures where no pull request appears more than once are already spread
// across pull requests. They are kept when at least minIndependent of them
// were attributed, which preserves a flake that failed once on each of two PRs.
func impactDominatedBySinglePR(impact searchci.Impact, maxAge time.Duration, minIndependent int) (dominated bool, pullNumber, covered, resolved int) {
	var builds []buildAttribution
	for _, build := range impact.BuildURLs {
		if build.Interval > maxAge {
			continue
		}
		prs, ok := attributeBuild(build.URL)
		builds = append(builds, buildAttribution{prs: prs, resolved: ok})
	}
	return singlePRDominated(builds, minIndependent)
}

func singlePRDominated(builds []buildAttribution, minIndependent int) (dominated bool, pullNumber, covered, resolved int) {
	if minIndependent < 1 {
		minIndependent = 1
	}
	var known []buildAttribution
	for _, build := range builds {
		if build.resolved && len(build.prs) > 0 {
			known = append(known, build)
		}
	}
	if len(known) == 0 {
		return true, 0, 0, 0
	}

	coverage := make(map[int]int)
	for _, build := range known {
		for pr := range build.prs {
			coverage[pr]++
		}
	}

	type candidate struct {
		pr          int
		covered     int
		unexplained int
	}
	best := candidate{unexplained: len(known) + 1}
	for pr, count := range coverage {
		unexplained := 0
		for _, build := range known {
			if _, ok := build.prs[pr]; !ok {
				unexplained++
			}
		}
		better := count > best.covered ||
			(count == best.covered && unexplained < best.unexplained) ||
			(count == best.covered && unexplained == best.unexplained && (best.pr == 0 || pr < best.pr))
		if better {
			best = candidate{pr: pr, covered: count, unexplained: unexplained}
		}
	}

	if best.covered <= 1 && len(known) >= minIndependent {
		return false, best.pr, best.covered, len(known)
	}
	if best.unexplained < minIndependent {
		return true, best.pr, best.covered, len(known)
	}
	return false, best.pr, best.covered, len(known)
}
