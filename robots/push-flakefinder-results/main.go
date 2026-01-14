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

	"kubevirt.io/project-infra/pkg/flakefinder"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

const (
	testName = iota
	testLane
	severity
	failed
	succeeded
	skipped
)

const (
	namespace = "flakefinder"
	subsystem = "report"
)

func main() {
	var pushgatewayURL, dateRange, org, repo string
	flag.StringVar(&pushgatewayURL, "pushgateway-url", "http://localhost:8080/", "pushgateway url to push values to")
	flag.StringVar(&dateRange, "date-range", flakefinder.DateRange24h, "daterange of the report (one of 024h, 168h, 672h)")
	flag.StringVar(&org, "github-org", "kubevirt", "github org")
	flag.StringVar(&repo, "github-repo", "kubevirt", "github repo")
	flag.Parse()

	pusher := push.New(pushgatewayURL, fmt.Sprintf("flakefinder_results_%s", dateRange))

	yesterday := time.Now().Add(-24 * time.Hour)
	fileType := "csv"
	reportCSVURL, err := flakefinder.GenerateReportURL(org, repo, yesterday, dateRange, fileType)
	if err != nil {
		log.Fatalf("failed to generate report url: %v", err)
	}
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

		pusher.Collector(
			newTestFailedGauge(labels, convertToIntOrDie(row[failed]))).Collector(
			newTestSucceededGauge(labels, convertToIntOrDie(row[succeeded]))).Collector(
			newTestSkippedGauge(labels, convertToIntOrDie(row[skipped])))
	}

	err = pusher.Push()
	if err != nil {
		log.Fatalf("push to %s failed: %v", pushgatewayURL, err)
	}
}

func newTestSucceededGauge(labels prometheus.Labels, timesTestSucceeded int) prometheus.Gauge {
	testSucceededGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "e2e_test_succeeded",
		Help:        "number of succeeded tests on presubmit job for merged PRs observed over the report period",
		ConstLabels: labels,
	})
	testSucceededGauge.Set(float64(timesTestSucceeded))
	return testSucceededGauge
}

func newTestFailedGauge(labels prometheus.Labels, timesTestFailed int) prometheus.Gauge {
	testFailedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "e2e_test_failed",
		Help:        "number of failed tests on presubmit jobs for all merged PRs observed over the report period",
		ConstLabels: labels,
	})
	testFailedGauge.Set(float64(timesTestFailed))
	return testFailedGauge
}

func newTestSkippedGauge(labels prometheus.Labels, timesTestSkipped int) prometheus.Gauge {
	testSkippedGauge := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   namespace,
		Subsystem:   subsystem,
		Name:        "e2e_test_skipped",
		Help:        "number of skipped tests on presubmit job for merged PRs observed over the report period",
		ConstLabels: labels,
	})
	testSkippedGauge.Set(float64(timesTestSkipped))
	return testSkippedGauge
}

func convertToIntOrDie(failed string) int {
	timesTestFailed, err := strconv.Atoi(failed)
	if err != nil {
		log.Fatalf("cannot convert %s to int", failed)
	}
	return timesTestFailed
}
