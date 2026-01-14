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
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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
	BuildURLs    []JobBuildURL
}

type JobBuildURL struct {
	URL      string
	Interval time.Duration
}

var (
	impactScrapeRegex = regexp.MustCompile(`(<tr><td.*<a target="_blank" href="(https://[a-z\.]+/job-history/kubevirt-prow/pr-logs/directory/[^"]+)">.*[0-9]+% of (failures|runs) match( = ([0-9\.]+)% impact)?</em></td></tr>|<tr class="row-match"><td[^\r\n]+href="(https://[a-z\.]+/view/[^"]+)">#[0-9]+</a>.*>([0-9]+) (minutes|hours|days) ago<.*</td></tr>)`)
	logger            = log.WithField("module", "searchci")
	serviceURL        = "https://search.ci.kubevirt.io"
)

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
	impactSubmatches := impactScrapeRegex.FindAllStringSubmatch(body, -1)
	for _, submatch := range impactSubmatches {
		jobHistoryURL := submatch[2]
		viewJobBuildURL := submatch[6]
		var err error
		switch {
		case strings.Contains(jobHistoryURL, "job-history"):
			impactPercent := 0.0
			action := submatch[3]
			if action == "failures" {
				impactPercentStr := submatch[5]
				impactPercent, err = strconv.ParseFloat(impactPercentStr, 64)
				if err != nil {
					log.WithError(err).Fatalf("unparseable impact %q", impactPercentStr)
				}
			}
			result = append(result, Impact{
				URL:          jobHistoryURL,
				Percent:      impactPercent,
				URLToDisplay: jobHistoryURL[strings.LastIndex(jobHistoryURL, "/")+1:],
			})
		case strings.Contains(viewJobBuildURL, "view"):
			timeAmountStr := submatch[7]
			timeAmount, err := strconv.Atoi(timeAmountStr)
			if err != nil {
				log.Fatalf("unparseable amount %q", timeAmountStr)
			}
			unitWord := submatch[8]
			var duration time.Duration
			switch {
			case unitWord == "minutes":
				duration = time.Minute * time.Duration(timeAmount)
			case unitWord == "hours":
				duration = time.Hour * time.Duration(timeAmount)
			case unitWord == "days":
				duration = time.Hour * 24 * time.Duration(timeAmount)
			}
			jobBuild := JobBuildURL{
				URL:      viewJobBuildURL,
				Interval: duration,
			}
			result[len(result)-1].BuildURLs = append(result[len(result)-1].BuildURLs, jobBuild)
		}
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
