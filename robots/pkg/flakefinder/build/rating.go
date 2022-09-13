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

package build

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

type BuildData struct {
	Number   int64   `json:"number"`
	Failures int64   `json:"failures"`
	Sigma    float64 `json:"sigma"`
}

// Rating contains the statistical evaluation regarding sigma values for failures per build over the contained set of builds.
// The goal is to determine so-called "bad builds" via the three sigma rule, that denotes that
// see https://en.wikipedia.org/wiki/68%E2%80%9395%E2%80%9399.7_rule
type Rating struct {
	Name                 string              `json:"name"`
	Source               string              `json:"source"`
	StartFrom            time.Duration       `json:"startFrom"`
	BuildNumbers         []int64             `json:"buildNumbers"`
	BuildNumbersToData   map[int64]BuildData `json:"buildNumbersToData"`
	TotalCompletedBuilds int64               `json:"totalCompletedBuilds"`
	TotalFailures        int64               `json:"totalFailures"`
	Mean                 float64             `json:"mean"`
	Variance             float64             `json:"variance"`
	StandardDeviation    float64             `json:"standardDeviation"`
}

func (r Rating) GetBuildData(buildNumber int64) BuildData {
	return r.BuildNumbersToData[buildNumber]
}

func (r Rating) ShouldFilterBuild(buildNumber int64) bool {
	buildData := r.GetBuildData(buildNumber)
	return buildData.Sigma > 3 && float64(buildData.Failures) > r.Mean
}

func (r Rating) String() string {
	bytes, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf("failed to serialize: %v", err)
	}
	return string(bytes)
}

func NewRating(name string, source string, startFrom time.Duration, buildNumbersToFailures map[int64]int64) Rating {
	totalFailures := calculateTotalNumberOfFailures(buildNumbersToFailures)
	totalCompletedBuilds := int64(len(buildNumbersToFailures))
	mean := float64(totalFailures) / float64(totalCompletedBuilds)

	sumOfSquareDeviations := calculateSumOfSquareDeviations(buildNumbersToFailures, mean)
	variance := sumOfSquareDeviations / float64(totalCompletedBuilds)
	standardDeviation := math.Sqrt(variance)

	buildNumbersToData := buildDeviationMap(buildNumbersToFailures, mean, standardDeviation)

	buildNumbers := []int64{}
	for buildNumber := range buildNumbersToData {
		buildNumbers = append(buildNumbers, buildNumber)
	}
	sort.Slice(buildNumbers, func(i, j int) bool { return buildNumbers[i] >= buildNumbers[j] })

	return Rating{
		Name:                 name,
		Source:               source,
		StartFrom:            startFrom,
		TotalCompletedBuilds: totalCompletedBuilds,
		TotalFailures:        totalFailures,
		Mean:                 mean,
		Variance:             variance,
		StandardDeviation:    standardDeviation,
		BuildNumbers:         buildNumbers,
		BuildNumbersToData:   buildNumbersToData,
	}
}

// buildDeviationMap creates a map for all builds
func buildDeviationMap(buildNumbersToFailures map[int64]int64, mean float64, standardDeviation float64) (buildNumbersToData map[int64]BuildData) {
	buildNumbersToData = map[int64]BuildData{}
	for buildNo, failures := range buildNumbersToFailures {
		var sigma float64
		if standardDeviation != float64(0) {
			sigma = math.Ceil(math.Abs(float64(failures)-mean) / standardDeviation)
		}
		buildNumbersToData[buildNo] = BuildData{
			Number:   buildNo,
			Failures: failures,
			Sigma:    sigma,
		}
	}
	return buildNumbersToData
}

// calculateSumOfSquareDeviations calculates the square of the deviation of each data point and returns the sum of all
func calculateSumOfSquareDeviations(buildNumbersToFailures map[int64]int64, mean float64) float64 {
	sumOfSquareDeviations := float64(0)
	for _, jobFailure := range buildNumbersToFailures {
		sumOfSquareDeviations += (float64(jobFailure) - mean) * (float64(jobFailure) - mean)
	}
	return sumOfSquareDeviations
}

func calculateTotalNumberOfFailures(buildNumbersToFailures map[int64]int64) int64 {
	totalFailures := int64(0)
	for _, jobFailure := range buildNumbersToFailures {
		totalFailures += jobFailure
	}
	return totalFailures
}
