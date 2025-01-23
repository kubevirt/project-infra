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

package cmd

import (
	"encoding/json"
	"kubevirt.io/project-infra/robots/pkg/cannier"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("host", func() {

	When("request data serialization", func() {

		It("deserialize works", func() {
			var requestData *RequestData
			open, err := os.Open("testdata/cannier-featureset.json")
			Expect(err).ToNot(HaveOccurred())
			err = json.NewDecoder(open).Decode(&requestData)
			Expect(err).ToNot(HaveOccurred())
		})

		It("serialize works", func() {
			requestData := &RequestData{
				Features: cannier.FeatureSet{
					ASTDepth:             0,
					Assertions:           0,
					CyclomaticComplexity: 0,
					TestLinesOfCode:      0,
					ExternalModules:      0,
					HalsteadVolume:       0,
					Maintainability:      0,
					ReadCount:            0,
					WriteCount:           0,
					MaxMemory:            0,
					ContextSwitches:      0,
					MaxThreads:           0,
					MaxChildren:          0,
					RunTime:              0,
					WaitTime:             0,
					CoveredLines:         0,
					SourceCoveredLines:   0,
					CoveredChanges:       0,
				},
			}
			temp, err := os.CreateTemp("", "host-model_test*.json")
			err = json.NewEncoder(temp).Encode(&requestData)
			Expect(err).ToNot(HaveOccurred())
		})

	})

})
