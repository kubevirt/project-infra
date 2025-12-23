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

package flakestats

import (
	"fmt"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("main", func() {

	DescribeTable("TopXTests less comparison",
		func(t TopXTests, expected bool) {
			Expect(t.Less(0, 1)).To(BeEquivalentTo(expected))
		},
		Entry("Sum i > j => i Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresSum(42)),
				NewTopXTestWithOptions("j", WithAllFailuresSum(17)),
			},
			true,
		),
		Entry("Sum i < j =>  i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresSum(17)),
				NewTopXTestWithOptions("j", WithAllFailuresSum(42)),
			},
			false,
		),
		Entry("Sum i = j => i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresSum(17)),
				NewTopXTestWithOptions("j", WithAllFailuresSum(17)),
			},
			false,
		),
		Entry("Max i > j => i Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresMax(42)),
				NewTopXTestWithOptions("j", WithAllFailuresMax(17)),
			},
			true,
		),
		Entry("Max i < j =>  i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresMax(17)),
				NewTopXTestWithOptions("j", WithAllFailuresMax(42)),
			},
			false,
		),
		Entry("Max i = j => i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresMax(17)),
				NewTopXTestWithOptions("j", WithAllFailuresMax(17)),
			},
			false,
		),
		Entry("Avg i > j => i Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresAvg(42)),
				NewTopXTestWithOptions("j", WithAllFailuresAvg(17)),
			},
			true,
		),
		Entry("Avg i < j =>  i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresAvg(17)),
				NewTopXTestWithOptions("j", WithAllFailuresAvg(42)),
			},
			false,
		),
		Entry("Avg i = j => i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresAvg(17)),
				NewTopXTestWithOptions("j", WithAllFailuresAvg(17)),
			},
			false,
		),

		Entry("recency: Sum i == j && i more recent failure => i Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(17),
					WithDatedFailuresSum(time.Now(), 17),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(17),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 17),
				),
			},
			true,
		),

		Entry("recency: Sum i == j && j more recent failure => i not Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(17),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 17),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(17),
					WithDatedFailuresSum(time.Now(), 17),
				),
			},
			false,
		),

		Entry("recency: Sum i > j && i failure with same recency => i Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(20),
					WithDatedFailuresSum(time.Now(), 20),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(17),
					WithDatedFailuresSum(time.Now(), 17),
				),
			},
			true,
		),

		Entry("recency: Sum i == j && i recent failure > j recent failure => i Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(37),
					WithDatedFailuresSum(time.Now(), 20),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 17),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(37),
					WithDatedFailuresSum(time.Now(), 20),
					WithDatedFailuresSum(time.Now().Add(-48*time.Hour), 17),
				),
			},
			true,
		),

		Entry("recency: Sum i == j && i recent failure < j recent failure => i not Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(37),
					WithDatedFailuresSum(time.Now(), 20),
					WithDatedFailuresSum(time.Now().Add(-48*time.Hour), 17),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(37),
					WithDatedFailuresSum(time.Now(), 20),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 17),
				),
			},
			false,
		),

		Entry("recency: Sum i < j but i failures more recent than j failures => i Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(17),
					WithDatedFailuresSum(time.Now(), 10),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 7),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(37),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 20),
					WithDatedFailuresSum(time.Now().Add(-48*time.Hour), 17),
				),
			},
			true,
		),

		Entry("recency: Sum i == j and i failures same recency as j failures and latest_failure(i) > latest_failure(j) => i !Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 4),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 2),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
				),
			},
			false,
		),

		Entry("recency: Sum i == j and i failures same recency as j failures and latest_failure(i) < latest_failure(j) but second_latest_failure > j => i !Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 2),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 4),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
				),
			},
			false,
		),

		Entry("recency: Sum i == j and i failures same recency as j failures and latest_failure(i) > latest_failure(j) but second_latest_failure < j => i !Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 2),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 4),
				),
			},
			false,
		),

		Entry("recency: Sum i > j and i failures same recency as j failures and sum_latest_failures(i) > sum_latest_failures(j) => i Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(7),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 4),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
				),
			},
			true,
		),

		Entry("recency: Sum i < j and i failures same recency as j failures and sum_latest_failures(i) < sum_latest_failures(j) => i !Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(6),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(7),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 4),
				),
			},
			false,
		),

		Entry("recency: Sum i > j and i failures same recency as j failures and latest_failure(i) == latest_failure(j) but sum_latest_failures(i) > sum_latest_failures(j) => i Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(10),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 4),
					WithDatedFailuresSum(time.Now().Add(-48*time.Hour), 3),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(9),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
					WithDatedFailuresSum(time.Now().Add(-48*time.Hour), 3),
				),
			},
			true,
		),

		Entry("recency: Sum i == j and i failures more recent than j failures and sum_latest_failures(i) == sum_latest_failures(j) => i Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(9),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
					WithDatedFailuresSum(time.Now().Add(-48*time.Hour), 3),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(9),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
					WithDatedFailuresSum(time.Now().Add(-72*time.Hour), 3),
				),
			},
			true,
		),

		Entry("recency: Sum i == j and i failures less recent than j failures and sum_latest_failures(i) == sum_latest_failures(j) => i !Less j",
			TopXTests{
				NewTopXTestWithOptions(
					"i",
					WithAllFailuresSum(9),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
					WithDatedFailuresSum(time.Now().Add(-72*time.Hour), 3),
				),
				NewTopXTestWithOptions(
					"j",
					WithAllFailuresSum(9),
					WithDatedFailuresSum(time.Now(), 3),
					WithDatedFailuresSum(time.Now().Add(-24*time.Hour), 3),
					WithDatedFailuresSum(time.Now().Add(-48*time.Hour), 3),
				),
			},
			false,
		),
	)

	DescribeTable("generate TestGrid URL",
		func(jobName string, expectedFragments []string) {
			urlForJob := generateTestGridURLForJob(jobName)
			_, err := url.Parse(urlForJob)
			if err != nil {
				Fail(fmt.Sprintf("failed to parse url %q: %v", urlForJob, err))
			}
			for _, fragmentExpected := range expectedFragments {
				Expect(urlForJob).To(ContainSubstring(fragmentExpected))
			}
		},
		Entry("presubmit",
			"pull-kubevirt-e2e-k8s-1.28-sig-storage",
			[]string{"testgrid", "kubevirt-presubmits"},
		),
		Entry("periodic",
			"periodic-kubevirt-e2e-k8s-1.28-sig-compute",
			[]string{"testgrid", "kubevirt-periodics"},
		),
	)

})

type TopXTestOption func(*TopXTest)

func WithAllFailuresSum(sum int) TopXTestOption {
	return func(t *TopXTest) {
		t.AllFailures.Sum = sum
	}
}

func WithAllFailuresMax(max int) TopXTestOption {
	return func(t *TopXTest) {
		t.AllFailures.Max = max
	}
}

func WithAllFailuresAvg(avg float64) TopXTestOption {
	return func(t *TopXTest) {
		t.AllFailures.Avg = avg
	}
}

func WithDatedFailuresSum(date time.Time, sum int) TopXTestOption {
	return func(t *TopXTest) {
		t.FailuresPerDay[date.Format(time.DateOnly)+"T00:00:00Z"] = &FailureCounter{
			Sum: sum,
		}
	}
}

func NewTopXTestWithOptions(name string, options ...TopXTestOption) *TopXTest {
	test := NewTopXTest(name)
	for _, o := range options {
		o(test)
	}
	test.daysInThePast = defaultDaysInThePast
	return test
}
