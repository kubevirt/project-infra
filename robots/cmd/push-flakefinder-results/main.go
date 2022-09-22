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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

const (
	dateRange24h  = "024h"
	dateRange168h = "168h"
	dateRange672h = "672h"
)

const (
	testName = iota
	testLane
	severity
	failed
	succeeded
	skipped
)

var dateRangeAllowedValues = map[string]struct{}{
	dateRange24h:  {},
	dateRange168h: {},
	dateRange672h: {},
}

func main() {
	var pushgatewayURL, dateRange, org, repo string
	flag.StringVar(&pushgatewayURL, "pushgateway-url", "http://localhost:8080/", "pushgateway url to push values to")
	flag.StringVar(&dateRange, "date-range", dateRange24h, "daterange of the report (one of 024h, 168h, 672h)")
	flag.StringVar(&org, "github-org", "kubevirt", "github org")
	flag.StringVar(&repo, "github-repo", "kubevirt", "github repo")
	flag.Parse()

	if _, exists := dateRangeAllowedValues[dateRange]; !exists {
		log.Fatalf("Value %q not allowed for range, allowed values: %v", dateRange, dateRangeAllowedValues)
	}

	pusher := push.New(pushgatewayURL, fmt.Sprintf("flakefinder_results_%s", dateRange))

	yesterday := time.Now().Add(-24 * time.Hour)
	reportCSVURL := fmt.Sprintf("https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/%s/%s/flakefinder-%s-%s.csv", org, repo, yesterday.Format("2006-01-02"), dateRange)
	log.Printf("fetching report %q", reportCSVURL)
	response, err := http.DefaultClient.Get(reportCSVURL)
	if err != nil {
		log.Fatalf("error fetching report %q", reportCSVURL)
	}
	if response.StatusCode != http.StatusOK {
		log.Fatalf("error %s fetching report %q", response.Status, reportCSVURL)
	}
	defer response.Body.Close()

	csvReader := csv.NewReader(response.Body)
	values, err := csvReader.ReadAll()
	if err != nil {
		log.Fatalf("error reading csv %q", reportCSVURL)
	}
	for index, row := range values {
		// skip headers
		if index == 0 {
			continue
		}

		labels := prometheus.Labels{
			"org":        org,
			"repo":       repo,
			"date_range": dateRange,
			"test_name":  row[testName],
			"test_lane":  row[testLane],
			"severity":   row[severity],
		}

		timesTestFailed, err := strconv.Atoi(row[failed])
		if err != nil {
			log.Fatalf("cannot convert %s to int", row[failed])
		}
		testFailedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
			"flakefinder",
			"report",
			"e2e_test_failed",
			"number of failed tests on presubmit jobs for all merged PRs observed over the report period",
			labels,
		})
		testFailedGauge.Set(float64(timesTestFailed))
		pusher.Collector(testFailedGauge)

		timesTestSucceeded, err := strconv.Atoi(row[succeeded])
		if err != nil {
			log.Fatalf("cannot convert %s to int", row[succeeded])
		}
		testSucceededGauge := prometheus.NewGauge(prometheus.GaugeOpts{
			"flakefinder",
			"report",
			"e2e_test_succeeded",
			"number of succeeded tests on presubmit job for merged PRs observed over the report period",
			labels,
		})
		testSucceededGauge.Set(float64(timesTestSucceeded))
		pusher.Collector(testSucceededGauge)

		timesTestSkipped, err := strconv.Atoi(row[skipped])
		if err != nil {
			log.Fatalf("cannot convert %s to int", row[skipped])
		}
		testSkippedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
			"flakefinder",
			"report",
			"e2e_test_skipped",
			"number of skipped tests on presubmit job for merged PRs observed over the report period",
			labels,
		})
		testSkippedGauge.Set(float64(timesTestSkipped))
		pusher.Collector(testSkippedGauge)
	}
	pusher.Push()
}
