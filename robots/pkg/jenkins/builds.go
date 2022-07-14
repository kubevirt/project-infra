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

package jenkins

import (
	"context"
	"github.com/avast/retry-go"
	"github.com/bndr/gojenkins"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type BuildStop struct {
	buildNumber int64
	build       *gojenkins.Build
	stop        bool
}

type buildTestResult struct {
	jobName     string
	buildNumber int64
	testResult  *gojenkins.TestResult
}

func GetBuildNumbersToFailuresForJob(startOfReport time.Time, job *gojenkins.Job, ctx context.Context, jLog *log.Entry) map[int64]int64 {
	testResultsForJob := GetBuildNumbersToTestResultsForJob(startOfReport, job, ctx, jLog)

	buildNumbersToFailures := map[int64]int64{}
	for buildNo, buildNumberToFailure := range testResultsForJob {
		buildNumbersToFailures[buildNo] = buildNumberToFailure.FailCount
	}
	return buildNumbersToFailures
}

func GetBuildNumbersToTestResultsForJob(startOfReport time.Time, job *gojenkins.Job, ctx context.Context, jLog *log.Entry) map[int64]*gojenkins.TestResult {
	bLog := jLog.WithField("job", job.GetName())
	completedBuilds := FetchCompletedBuildsForJob(startOfReport, job.Raw.LastBuild.Number, job, ctx, bLog, 4)

	buildNumbersToTestResultsChan := make(chan buildTestResult)

	go fetchTestResultsForBuilds(ctx, completedBuilds, buildNumbersToTestResultsChan, bLog)

	buildNumbersToTestResults := map[int64]*gojenkins.TestResult{}
	for buildNumberToTestResult := range buildNumbersToTestResultsChan {
		bLog.Debugf("adding %v results", buildNumberToTestResult)
		buildNumbersToTestResults[buildNumberToTestResult.buildNumber] = buildNumberToTestResult.testResult
	}
	bLog.Debugf("total result: %v", buildNumbersToTestResults)
	return buildNumbersToTestResults
}

func fetchTestResultsForBuilds(ctx context.Context, completedBuilds []*gojenkins.Build, buildNumbersToFailuresChan chan buildTestResult, jLog *log.Entry) {

	var wg sync.WaitGroup
	wg.Add(len(completedBuilds))

	defer close(buildNumbersToFailuresChan)
	for _, completedBuild := range completedBuilds {
		go fetchTestResultForBuild(ctx, completedBuild, jLog, &wg, buildNumbersToFailuresChan)
	}

	jLog.Debugf("waiting for %d results", len(completedBuilds))
	wg.Wait()
	jLog.Debugf("got %d results", len(completedBuilds))
}

func fetchTestResultForBuild(ctx context.Context, completedBuild *gojenkins.Build, jLog *log.Entry, wg *sync.WaitGroup, buildNumbersToFailuresChan chan buildTestResult) {
	defer wg.Done()

	buildNumber := completedBuild.GetBuildNumber()
	jLog.Debugf("fetching testresult for build %d", buildNumber)
	testResult, err := completedBuild.GetResultSet(ctx)
	if err != nil {
		jLog.Fatalf("failed to get resultset for %v: %v", completedBuild, err)
	}
	jLog.Debugf("build %d has %d failures", buildNumber, testResult.FailCount)

	buildNumbersToFailuresChan <- buildTestResult{completedBuild.Job.GetName(), buildNumber, testResult}
}

func FetchCompletedBuildsForJob(startOfReport time.Time, lastBuildNumber int64, job *gojenkins.Job, ctx context.Context, fLog *log.Entry, paginationSize int) []*gojenkins.Build {
	fLog.Printf("Fetching completed builds, starting at %d", lastBuildNumber)
	var completedBuilds []*gojenkins.Build
	for buildNumber := lastBuildNumber; buildNumber > 0; buildNumber = buildNumber - int64(paginationSize) {

		buildStopChan := make(chan BuildStop)

		go getBuildsPaged(startOfReport, paginationSize, buildStopChan, buildNumber, job, ctx, fLog)

		stop := false
		for buildStop := range buildStopChan {
			fLog.Debugf("Fetched buildStop %v", buildStop)
			if buildStop.build != nil {
				completedBuilds = append(completedBuilds, buildStop.build)
			}
			if buildStop.stop {
				stop = true
			}
		}
		if stop {
			break
		}
	}
	fLog.Printf("Fetched %d completed builds", len(completedBuilds))
	return completedBuilds
}

func getBuildsPaged(startOfReport time.Time, paginationSize int, buildStopChan chan BuildStop, buildNumber int64, job *gojenkins.Job, ctx context.Context, fLog *log.Entry) {
	var wg sync.WaitGroup
	wg.Add(paginationSize)

	defer close(buildStopChan)
	for i := 0; i < paginationSize; i++ {
		pageBuildNumber := buildNumber - int64(i)
		go getFilteredBuildOrStop(buildStopChan, startOfReport, pageBuildNumber, job, ctx, fLog.WithField("build", pageBuildNumber), &wg)
	}

	wg.Wait()
}

func getFilteredBuildOrStop(buildStopChan chan BuildStop, startOfReport time.Time, buildNumber int64, job *gojenkins.Job, ctx context.Context, fLog *log.Entry, wg *sync.WaitGroup) {
	defer wg.Done()
	build, stop := getFilteredBuild(startOfReport, job, ctx, buildNumber, fLog)
	buildStopChan <- BuildStop{
		buildNumber: buildNumber,
		build:       build,
		stop:        stop,
	}
}

func getFilteredBuild(startOfReport time.Time, job *gojenkins.Job, ctx context.Context, buildNumber int64, fLog *log.Entry) (build *gojenkins.Build, stop bool) {
	fLog.Printf("Fetching build no %d", buildNumber)
	build, statusCode, err := getBuildWithRetry(job, ctx, buildNumber, fLog)

	if build == nil {
		if statusCode != http.StatusNotFound {
			fLog.Fatalf("failed to fetch build data for build no %d: %v", buildNumber, err)
		}
		return nil, false
	}

	if build.GetResult() != "SUCCESS" &&
		build.GetResult() != "UNSTABLE" {
		fLog.Printf("Skipping build no %d with state %s", buildNumber, build.GetResult())
		return nil, false
	}

	buildTime := msecsToTime(build.Info().Timestamp)
	fLog.Printf("Build %d ran at %s", build.Info().Number, buildTime.Format(time.RFC3339))
	if buildTime.Before(startOfReport) {
		fLog.Printf("Skipping build no %d as too early", buildNumber)
		return nil, true
	}

	return build, false
}

func getBuildWithRetry(job *gojenkins.Job, ctx context.Context, buildNumber int64, fLog *log.Entry) (build *gojenkins.Build, statusCode int, err error) {
	retry.Do(
		func() error {
			build, err = job.GetBuild(ctx, buildNumber)
			if err != nil {
				return err
			}
			return nil
		},
		retry.RetryIf(func(err error) bool {
			fLog.Warningf("failed to fetch build data for build no %d: %v", buildNumber, err)
			statusCode = httpStatusOrDie(err, fLog)
			if statusCode == http.StatusNotFound {
				return false
			}
			if statusCode == http.StatusGatewayTimeout {
				return true
			}
			return false
		}),
		retry.Delay(5*time.Second),
		retry.MaxJitter(3*time.Second),
	)
	return build, statusCode, err
}

// httpStatusOrDie fetches [stringly typed](https://wiki.c2.com/?StringlyTyped) error code produced by jenkins client
// or logs a fatal error if conversion to int is not possible
func httpStatusOrDie(err error, fLog *log.Entry) int {
	statusCode, conversionError := strconv.Atoi(err.Error())
	if conversionError != nil {
		fLog.Fatalf("Failed to get status code from error %v: %v", err, conversionError)
	}
	return statusCode
}

func msecsToTime(msecs int64) time.Time {
	return time.Unix(msecs/1000, msecs%1000)
}
