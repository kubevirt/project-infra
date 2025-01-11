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
	"fmt"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"time"
)

type featureExtractor func(*FeatureSet) error

// ExtractFeatures extracts the feature set for a test.
func ExtractFeatures(test *ginkgo.TestDescriptor) (*FeatureSet, error) {
	featureSet := &FeatureSet{}
	if test == nil {
		return nil, fmt.Errorf("no test descriptor given")
	}

	startTime := time.Now()

	extractors, err := getStaticAnalysisExtractors(test)
	if err != nil {
		return featureSet, err
	}

	extractors = append(extractors, getMonitoringExtractors(startTime)...)
	extractors = append(extractors, getCoverageExtractors(test)...)

	// TODO: initialize process monitoring

	// TODO: run the test

	// TODO: stop process monitoring

	for _, extract := range extractors {
		err = extract(featureSet)
		if err != nil {
			return featureSet, err
		}
	}

	return featureSet, nil
}
