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

package cannier

import (
	"time"
)

type featureExtractor func(*FeatureSet) error

// ExtractFeatures extracts the feature set from a Go test file.
func ExtractFeatures(filename string) (FeatureSet, error) {
	featureSet := FeatureSet{}

	startTime := time.Now()

	extractors, err := getStaticAnalysisExtractors(filename)
	if err != nil {
		return featureSet, err
	}

	extractors = append(extractors, getMonitoringExtractors(startTime)...)
	extractors = append(extractors, getCoverageExtractors()...)

	// TODO: initialize process monitoring

	// TODO: run the test

	// TODO: stop process monitoring

	for _, extract := range extractors {
		err = extract(&featureSet)
		if err != nil {
			return featureSet, err
		}
	}

	return featureSet, nil
}
