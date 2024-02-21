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

package ghgraphql_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
)

var _ = Describe("Labels", func() {
	DescribeTable("NewLabels", func(labels []ghgraphql.Label, expected ghgraphql.PRLabels) {
		Expect(ghgraphql.NewPRLabels(labels)).To(BeEquivalentTo(expected))
	},
		Entry("no labels don't set IsHoldPresent", nil, ghgraphql.PRLabels{
			IsHoldPresent: false,
		}),
		Entry("some labels don't set IsHoldPresent", []ghgraphql.Label{
			{Name: "test"},
		}, ghgraphql.PRLabels{
			IsHoldPresent: false,
			Labels: []ghgraphql.Label{
				{Name: "test"},
			},
		}),
		Entry("exactly one label sets IsHoldPresent", []ghgraphql.Label{
			{Name: "ok-to-test"},
			{Name: "do-not-merge/hold"},
			{Name: "needs-rebase"},
		}, ghgraphql.PRLabels{
			IsHoldPresent: true,
			Labels: []ghgraphql.Label{
				{Name: "ok-to-test"},
				{Name: "do-not-merge/hold"},
				{Name: "needs-rebase"},
			},
		}),
	)
})
