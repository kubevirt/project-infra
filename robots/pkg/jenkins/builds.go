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
	"time"
)

func FetchCompletedBuildsForJob(startOfReport time.Time, lastBuildNumber int64, job *gojenkins.Job, ctx context.Context, fLog *log.Entry) []*gojenkins.Build {
	fLog.Printf("Fetching completed builds, starting at %d", lastBuildNumber)
	var completedBuilds []*gojenkins.Build
	for buildNumber := lastBuildNumber; buildNumber > 0; buildNumber-- {
		fLog.Printf("Fetching build no %d", buildNumber)
		build, statusCode, err := getBuildWithRetry(job, ctx, buildNumber, fLog)

		if build == nil {
			if statusCode != http.StatusNotFound {
				fLog.Fatalf("failed to fetch build data for build no %d: %v", buildNumber, err)
			}
			continue
		}

		if build.GetResult() != "SUCCESS" &&
			build.GetResult() != "UNSTABLE" {
			fLog.Printf("Skipping build with state %s", build.GetResult())
			continue
		}

		buildTime := msecsToTime(build.Info().Timestamp)
		fLog.Printf("Build %d ran at %s", build.Info().Number, buildTime.Format(time.RFC3339))
		if buildTime.Before(startOfReport) {
			fLog.Printf("Skipping remaining builds")
			break
		}

		completedBuilds = append(completedBuilds, build)
	}
	fLog.Printf("Fetched %d completed builds", len(completedBuilds))
	return completedBuilds
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
