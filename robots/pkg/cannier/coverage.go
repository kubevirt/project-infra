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

import "kubevirt.io/project-infra/robots/pkg/ginkgo"

// TODO: evaluate whether that is feasible with a reasonable amount of work, would need remote instrumentation

var (
	coverageExtractors = []featureExtractor{
		func(featureSet *FeatureSet) error {
			featureSet.CoveredChanges = 0
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.CoveredLines = 0
			return nil
		},
		func(featureSet *FeatureSet) error {
			featureSet.SourceCoveredLines = 0
			return nil
		},
	}
)

func getCoverageExtractors(test *ginkgo.TestDescriptor) []featureExtractor {
	return coverageExtractors
}
