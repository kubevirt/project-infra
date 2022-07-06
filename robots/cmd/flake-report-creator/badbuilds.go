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
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/bndr/gotabulate"
	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/pkg/flakefinder/build"
	flakejenkins "kubevirt.io/project-infra/robots/pkg/jenkins"
	"net/http"
	"sync"
	"time"
)

const (
	defaultJenkinsBaseUrl = "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/"
)

type options struct {
	endpoint  string
	startFrom time.Duration
	jobName   string
}

func flagOptions() options {
	o := options{}
	flag.StringVar(&o.endpoint, "endpoint", defaultJenkinsBaseUrl, "jenkins base url")
	flag.DurationVar(&o.startFrom, "start-from", 14*24*time.Hour, "jenkins job name")
	flag.StringVar(&o.jobName, "job-name", "", "jenkins job name")
	flag.Parse()
	return o
}

func main() {
	opts := flagOptions()

	jLog := log.StandardLogger().WithField("robot", "badbuilds")

	ctx := context.Background()

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: 5,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	jLog.Printf("Creating client for %s", opts.endpoint)
	jenkins := gojenkins.CreateJenkins(client, opts.endpoint)
	_, err := jenkins.Init(ctx)
	if err != nil {
		jLog.Fatalf("failed to contact jenkins %s: %v", opts.endpoint, err)
	}

	job, err := jenkins.GetJob(ctx, opts.jobName)
	if err != nil {
		jLog.Fatalf("failed to get job %s: %v", opts.jobName, err)
	}

	startOfReport := time.Now().Add(-1 * opts.startFrom)
	buildNumbersToFailures := getBuildNumbersToFailuresForJob(startOfReport, job, ctx, jLog)

	result := build.NewRating(opts.jobName, opts.endpoint, opts.startFrom, buildNumbersToFailures)

	data := [][]interface{}{}
	for _, buildNo := range result.BuildNumbers {
		failures, deviating := result.GetBuildData(buildNo)
		row := []interface{}{result.Name, buildNo, failures, deviating}
		data = append(data, row)
	}

	t := gotabulate.Create(data)
	t.SetHeaders([]string{"job_name", "build_number", "failures", "is bad build"})
	t.SetEmptyString("-")
	t.SetAlign("left")
	fmt.Println(t.Render("simple"))
}

func getBuildNumbersToFailuresForJob(startOfReport time.Time, job *gojenkins.Job, ctx context.Context, jLog *log.Entry) map[int64]int64 {
	completedBuilds := flakejenkins.FetchCompletedBuildsForJob(startOfReport, job.Raw.LastBuild.Number, job, ctx, jLog)

	buildNumbersToFailuresChan := make(chan []int64)

	go fetchFailuresForBuilds(ctx, completedBuilds, buildNumbersToFailuresChan, jLog)

	buildNumbersToFailures := map[int64]int64{}
	for buildNumberToFailure := range buildNumbersToFailuresChan {
		jLog.Debugf("adding %v results", buildNumberToFailure)
		buildNumbersToFailures[buildNumberToFailure[0]] = buildNumberToFailure[1]
	}
	jLog.Debugf("total result: %v", buildNumbersToFailures)
	return buildNumbersToFailures
}

func fetchFailuresForBuilds(ctx context.Context, completedBuilds []*gojenkins.Build, buildNumbersToFailuresChan chan []int64, jLog *log.Entry) {

	var wg sync.WaitGroup
	wg.Add(len(completedBuilds))

	defer close(buildNumbersToFailuresChan)
	for _, completedBuild := range completedBuilds {
		go fetchFailureForBuild(ctx, completedBuild, jLog, &wg, buildNumbersToFailuresChan)
	}

	jLog.Debugf("waiting for %d results", len(completedBuilds))
	wg.Wait()
	jLog.Debugf("got %d results", len(completedBuilds))
}

func fetchFailureForBuild(ctx context.Context, completedBuild *gojenkins.Build, jLog *log.Entry, wg *sync.WaitGroup, buildNumbersToFailuresChan chan []int64) {
	defer wg.Done()

	buildNumber := completedBuild.GetBuildNumber()
	jLog.Debugf("fetching failures for build %d", buildNumber)
	testResult, err := completedBuild.GetResultSet(ctx)
	if err != nil {
		jLog.Fatalf("failed to get resultset for %v: %v", completedBuild, err)
	}
	failures := testResult.FailCount
	jLog.Debugf("build %d has %d failures", buildNumber, failures)

	buildNumbersToFailuresChan <- []int64{buildNumber, failures}
}
