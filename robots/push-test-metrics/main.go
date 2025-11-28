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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/joshdk/go-junit"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"

	"github.com/prometheus/client_golang/prometheus/push"
)

type jobNames []string

func (j jobNames) String() string {
	return strings.Join(j, ",")
}
func (j jobNames) Set(string) error {
	return nil
}

func main() {
	var pushgatewayURL string
	var jobNamesToScan jobNames
	flag.StringVar(&pushgatewayURL, "pushgateway-url", "http://localhost:8080/", "pushgateway url to push values to")
	flag.Var(jobNamesToScan, "job-name", "periodic job names to scan for values")
	flag.Parse()
	if len(jobNamesToScan) == 0 {
		jobNamesToScan = jobNames{
			"periodic-kubevirt-e2e-k8s-1.29-sig-compute",
			"periodic-kubevirt-e2e-k8s-1.29-sig-storage",
			"periodic-kubevirt-e2e-k8s-1.29-sig-network",
			"periodic-kubevirt-e2e-k8s-1.29-sig-operator",
		}
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("error creating gcs client: %v", err)
	}
	bucket := client.Bucket("kubevirt-prow")
	pusher := push.New(pushgatewayURL, "ci_test_results")
	for _, jobName := range jobNamesToScan {
		jobRunIterator := bucket.Objects(ctx, &storage.Query{
			Delimiter: "/",
			Prefix:    fmt.Sprintf("logs/%s/", jobName),
		})
		var jobRuns []string
		for {
			attrs, err := jobRunIterator.Next()
			if errors.Is(err, iterator.Done) {
				break
			}
			if err != nil {
				log.Fatalf("err: %v", err)

			}
			if attrs.Prefix != "" {
				jobRuns = append(jobRuns, path.Base(attrs.Prefix))
			}
		}
		sort.Sort(sort.Reverse(sort.StringSlice(jobRuns)))
		var junitBytes []byte
		for jobRunIndex := 0; jobRunIndex < len(jobRuns); jobRunIndex++ {
			junitGCSPath := fmt.Sprintf("logs/%s/%s/artifacts/junit.functest.xml", jobName, jobRuns[jobRunIndex])
			latestJunitXML := client.Bucket("kubevirt-prow").Object(junitGCSPath)
			reader, err := latestJunitXML.NewReader(ctx)
			if errors.Is(err, storage.ErrObjectNotExist) {
				continue
			}
			if err != nil {
				log.Fatalf("failed to fetch %s: %v", junitGCSPath, err)
			}
			junitBytes, err = io.ReadAll(reader)
			if err != nil {
				log.Fatalf("failed to fetch %s: %v", junitGCSPath, err)
			}
			log.Infof("jobName: %s, read %s", jobName, junitGCSPath)
			break
		}
		suites, err := junit.Ingest(junitBytes)
		if err != nil {
			log.Fatalf("failed to ingest JUnit xml: %v", err)
		}
		buckets := []float64{
			1.0,
			10.0,
			60.0,
			120.0,
			300.0,
			600.0,
		}
		ciTestsExecutionSummary := prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: "ci",
				Subsystem: "test",
				Name:      "runtime_seconds_total",
				Help:      "time in seconds all tests took on ci",
				ConstLabels: prometheus.Labels{
					"job_name": jobName,
				},
				Buckets: buckets,
			},
		)
		for _, suite := range suites {
			for _, test := range suite.Tests {
				if test.Status == junit.StatusSkipped {
					continue
				}
				ciTestsExecutionSummary.Observe(test.Duration.Seconds())
				labels := prometheus.Labels{
					"job_name":    jobName,
					"test_status": string(test.Status),
					"test_name":   test.Name,
				}
				ciTestExecutionSummary := prometheus.NewSummary(
					prometheus.SummaryOpts{
						Namespace:   "ci",
						Subsystem:   "test",
						Name:        "runtime_seconds",
						Help:        "time in seconds the test took on ci",
						ConstLabels: labels,
					})
				ciTestExecutionSummary.Observe(test.Duration.Seconds())
				pusher.Collector(ciTestExecutionSummary)
				log.Debugf("jobName: %s, test: %s, duration: %s", jobName, test.Name, test.Duration)
			}
		}
		pusher.Collector(ciTestsExecutionSummary)
	}
	err = pusher.Push()
	if err != nil {
		log.Fatalf("push to %s failed: %v", pushgatewayURL, err)
	}
}
