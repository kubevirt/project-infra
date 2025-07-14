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

package searchci

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

const (
	ThreeDays    = TimeRange("72h")
	FourteenDays = TimeRange("336h")

	excludePeriodic = "periodic-.*"
)

type TimeRange string

type Impact struct {
	URL          string
	Percent      float64
	URLToDisplay string
}

var (
	impactScrapeRegex *regexp.Regexp
	logger            *log.Entry
	serviceURL        = "https://search.ci.kubevirt.io"
)

func init() {
	logger = log.WithField("module", "searchci")
	impactScrapeRegex = regexp.MustCompile(`<a target="_blank" href="(https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/[^"]+)">.*[0-9]+% of failures match = ([0-9\.]+)% impact`)
}

// ScrapeImpacts scrapes results that are relevant for quarantining from search.ci.kubevirt.io
func ScrapeImpacts(testNameSubstring string, timeRange TimeRange) ([]Impact, error) {
	scrapeResultURL := NewScrapeURL(testNameSubstring, timeRange)
	logger.Debugf("scraping search.ci for test %q with URL %q", testNameSubstring, scrapeResultURL)
	resp, err := http.Get(scrapeResultURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get search.ci results from %s: %w", scrapeResultURL, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search.ci result from %s: %w", scrapeResultURL, err)
	}
	return FilterImpacts(ScrapeImpact(string(body)), timeRange), nil
}

func NewScrapeURL(testNameSubstring string, timeRange TimeRange) string {
	scrapeResultURL := fmt.Sprintf("%s/?search=%s&maxAge=%s&context=1&type=junit&name=&excludeName=%s&maxMatches=1&maxBytes=20971520&groupBy=job", serviceURL, escapeForQuery(testNameSubstring), timeRange, excludePeriodic)
	return scrapeResultURL
}

func escapeForQuery(testNameSubstring string) string {
	result := strings.ReplaceAll(testNameSubstring, "[", `\[`)
	result = url.QueryEscape(result)
	return result
}

func ScrapeImpact(body string) []Impact {
	var result []Impact
	impactSubmatches := impactScrapeRegex.FindAllStringSubmatch(string(body), -1)
	if impactSubmatches == nil {
		return nil
	}
	for _, submatch := range impactSubmatches {
		if len(submatch) < 3 {
			log.Fatal("no match")
		}
		impactPercent, err := strconv.ParseFloat(submatch[2], 64)
		if err != nil {
			log.WithError(err).Fatalf("failed to parse impact %s", submatch[2])
		}
		urlToDisplay := submatch[1][strings.LastIndex(submatch[1], "/")+1:]
		result = append(result, Impact{
			URL:          submatch[1],
			Percent:      impactPercent,
			URLToDisplay: urlToDisplay,
		})
	}
	return result
}

type FilterOpt func(i Impact) bool

func matchingTimeRange(timeRange TimeRange) func(i Impact) bool {
	switch timeRange {
	case ThreeDays:
		return func(i Impact) bool {
			return i.Percent >= 20.0
		}
	case FourteenDays:
		return func(i Impact) bool {
			return i.Percent >= 5.0
		}
	default:
		return func(i Impact) bool {
			log.Fatalf("unsupported timerange %q", timeRange)
			return false
		}
	}
}

func FilterImpacts(impacts []Impact, timeRange TimeRange) []Impact {
	return FilterImpactsBy(impacts, matchingTimeRange(timeRange))
}

// FilterImpactsBy filters all impacts for which all filterOpts apply, i.e. each of them returns true
func FilterImpactsBy(impacts []Impact, filterOpts ...FilterOpt) []Impact {
	if impacts == nil {
		return nil
	}
	var relevantImpacts []Impact
	for _, impact := range impacts {
		shouldKeep := true
		for _, filter := range filterOpts {
			shouldKeep = filter(impact)
			if !shouldKeep {
				break
			}
		}
		if shouldKeep {
			relevantImpacts = append(relevantImpacts, impact)
		}
	}
	return relevantImpacts
}
