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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPerTestExecution(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "per-test-execution Main Suite")
}

var _ = Describe("per-test-execution", func() {
	Context("ByFailuresDescending", func() {
		DescribeTable("sorts", func(i, j *TestExecutions, less bool) {
			Expect(ByFailuresDescending([]*TestExecutions{i, j}).Less(0, 1)).To(Equal(less))
		},
			Entry(
				"nothing",
				&TestExecutions{},
				&TestExecutions{},
				false,
			),
			Entry(
				"2 failed and 80 total is not less than 3 failed and 44 total",
				&TestExecutions{FailedExecutions: 2, TotalExecutions: 80, Name: "a"},
				&TestExecutions{FailedExecutions: 3, TotalExecutions: 44, Name: "a"},
				false,
			),
			Entry(
				"3 failed and 44 total is less than 2 failed and 80 total",
				&TestExecutions{FailedExecutions: 3, TotalExecutions: 44, Name: "a"},
				&TestExecutions{FailedExecutions: 2, TotalExecutions: 80, Name: "a"},
				true,
			),
			Entry(
				"3 failed and 44 total is less than 3 failed and 44 total if test name is lexically smaller",
				&TestExecutions{FailedExecutions: 3, TotalExecutions: 44, Name: "a"},
				&TestExecutions{FailedExecutions: 3, TotalExecutions: 44, Name: "b"},
				true,
			),
		)
	})
})
