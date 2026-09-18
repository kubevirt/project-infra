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
	"regexp"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/joshdk/go-junit"

	"kubevirt.io/project-infra/pkg/flakefinder"
)

func TestTestExecutionReport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "test-execution-report Main Suite")
}

var _ = Describe("buildMatrix", func() {

	matchAll := regexp.MustCompile(`.*`)

	newResult := func(job string, tests ...junit.Test) *flakefinder.JobResult {
		return &flakefinder.JobResult{
			Job: job,
			JUnit: []junit.Suite{
				{Tests: tests},
			},
		}
	}

	passedTest := func(name string) junit.Test {
		return junit.Test{Name: name, Status: junit.StatusPassed}
	}

	skippedTest := func(name string) junit.Test {
		return junit.Test{Name: name, Status: junit.StatusSkipped}
	}

	failedTest := func(name string) junit.Test {
		return junit.Test{Name: name, Status: junit.StatusFailed}
	}

	DescribeTable("returns empty matrix",
		func(results []*flakefinder.JobResult) {
			Expect(buildMatrix(results, matchAll)).To(BeEmpty())
		},
		Entry("for nil input", nil),
		Entry("for empty results", []*flakefinder.JobResult{}),
		Entry("for results with no JUnit suites",
			[]*flakefinder.JobResult{{Job: "job-a", JUnit: nil}}),
		Entry("for suites with no tests",
			[]*flakefinder.JobResult{{Job: "job-a", JUnit: []junit.Suite{{Tests: nil}}}}),
	)

	DescribeTable("maps execution status",
		func(results []*flakefinder.JobResult, testName, jobName string, expectedStatus int) {
			matrix := buildMatrix(results, matchAll)
			Expect(matrix[testName][jobName]).To(Equal(expectedStatus))
		},
		Entry("passed test as ExecRan",
			[]*flakefinder.JobResult{newResult("job-a", passedTest("test-1"))},
			"test-1", "job-a", ExecRan),
		Entry("skipped test as ExecSkipped",
			[]*flakefinder.JobResult{newResult("job-a", skippedTest("test-1"))},
			"test-1", "job-a", ExecSkipped),
		Entry("quarantined test as ExecQuarantined",
			[]*flakefinder.JobResult{newResult("job-a", skippedTest("[QUARANTINE] test-1"))},
			"[QUARANTINE] test-1", "job-a", ExecQuarantined),
		Entry("failed test as ExecRan",
			[]*flakefinder.JobResult{newResult("job-a", failedTest("test-1"))},
			"test-1", "job-a", ExecRan),
		Entry("promotes skipped to ran across multiple results",
			[]*flakefinder.JobResult{
				newResult("job-a", skippedTest("test-1")),
				newResult("job-a", passedTest("test-1")),
			},
			"test-1", "job-a", ExecRan),
		Entry("does not demote a higher status",
			[]*flakefinder.JobResult{
				newResult("job-a", passedTest("test-1")),
				newResult("job-a", skippedTest("test-1")),
			},
			"test-1", "job-a", ExecRan),
		Entry("quarantine beats skipped for the same test on the same job",
			[]*flakefinder.JobResult{
				newResult("job-a", skippedTest("[QUARANTINE] test-q")),
				newResult("job-a", skippedTest("[QUARANTINE] test-q")),
			},
			"[QUARANTINE] test-q", "job-a", ExecQuarantined),
		Entry("ExecQuarantined beats ExecRan because it has a higher iota value",
			[]*flakefinder.JobResult{
				newResult("job-a", passedTest("[QUARANTINE] test-q")),
				newResult("job-a", skippedTest("[QUARANTINE] test-q")),
			},
			"[QUARANTINE] test-q", "job-a", ExecQuarantined),
		Entry("returns ExecNoData for jobs a test was not seen on",
			[]*flakefinder.JobResult{
				newResult("job-a", passedTest("test-1")),
				newResult("job-b", passedTest("test-2")),
			},
			"test-1", "job-b", ExecNoData),
	)

	DescribeTable("builds correct matrix",
		func(results []*flakefinder.JobResult, expected map[string]map[string]int) {
			Expect(buildMatrix(results, matchAll)).To(Equal(expected))
		},
		Entry("tracks tests across multiple jobs independently",
			[]*flakefinder.JobResult{
				newResult("job-a", passedTest("test-1")),
				newResult("job-b", skippedTest("test-1")),
			},
			map[string]map[string]int{
				"test-1": {"job-a": ExecRan, "job-b": ExecSkipped},
			}),
		Entry("tracks multiple tests across multiple jobs",
			[]*flakefinder.JobResult{
				newResult("job-a", passedTest("test-1"), skippedTest("test-2")),
				newResult("job-b", skippedTest("test-1"), passedTest("test-2")),
			},
			map[string]map[string]int{
				"test-1": {"job-a": ExecRan, "job-b": ExecSkipped},
				"test-2": {"job-a": ExecSkipped, "job-b": ExecRan},
			}),
		Entry("handles multiple suites within a single JobResult",
			[]*flakefinder.JobResult{
				{
					Job: "job-a",
					JUnit: []junit.Suite{
						{Tests: []junit.Test{passedTest("test-1")}},
						{Tests: []junit.Test{passedTest("test-2")}},
					},
				},
			},
			map[string]map[string]int{
				"test-1": {"job-a": ExecRan},
				"test-2": {"job-a": ExecRan},
			}),
	)

	It("filters tests by the provided pattern", func() {
		pattern := regexp.MustCompile(`^include-`)
		results := []*flakefinder.JobResult{
			newResult("job-a", passedTest("include-this"), passedTest("exclude-that")),
		}
		matrix := buildMatrix(results, pattern)
		Expect(matrix).To(HaveKey("include-this"))
		Expect(matrix).NotTo(HaveKey("exclude-that"))
	})
})

var _ = Describe("classifyTests", func() {

	DescribeTable("classifies tests",
		func(matrix map[string]map[string]int, expectedNames []string, expectedSkipped, expectedQuarantined map[string]bool) {
			names, skipped, quarantined := classifyTests(matrix)
			Expect(names).To(Equal(expectedNames))
			Expect(skipped).To(Equal(expectedSkipped))
			Expect(quarantined).To(Equal(expectedQuarantined))
		},
		Entry("empty matrix",
			map[string]map[string]int{},
			([]string)(nil),
			map[string]bool{},
			map[string]bool{}),
		Entry("single test that ran",
			map[string]map[string]int{
				"test-1": {"job-a": ExecRan},
			},
			[]string{"test-1"},
			map[string]bool{},
			map[string]bool{}),
		Entry("single test that was only skipped",
			map[string]map[string]int{
				"test-1": {"job-a": ExecSkipped},
			},
			[]string{"test-1"},
			map[string]bool{"test-1": true},
			map[string]bool{}),
		Entry("quarantined test is in both skipped and quarantined",
			map[string]map[string]int{
				"[QUARANTINE] test-1": {"job-a": ExecQuarantined},
			},
			[]string{"[QUARANTINE] test-1"},
			map[string]bool{"[QUARANTINE] test-1": true},
			map[string]bool{"[QUARANTINE] test-1": true}),
		Entry("quarantined test that also ran is quarantined but not skipped",
			map[string]map[string]int{
				"[QUARANTINE] test-1": {"job-a": ExecRan, "job-b": ExecQuarantined},
			},
			[]string{"[QUARANTINE] test-1"},
			map[string]bool{},
			map[string]bool{"[QUARANTINE] test-1": true}),
		Entry("test ran on one job but skipped on another is not skipped",
			map[string]map[string]int{
				"test-1": {"job-a": ExecRan, "job-b": ExecSkipped},
			},
			[]string{"test-1"},
			map[string]bool{},
			map[string]bool{}),
		Entry("test with only ExecNoData is skipped",
			map[string]map[string]int{
				"test-1": {"job-a": ExecNoData},
			},
			[]string{"test-1"},
			map[string]bool{"test-1": true},
			map[string]bool{}),
		Entry("test names are sorted",
			map[string]map[string]int{
				"z-test": {"job-a": ExecRan},
				"a-test": {"job-a": ExecRan},
				"m-test": {"job-a": ExecRan},
			},
			[]string{"a-test", "m-test", "z-test"},
			map[string]bool{},
			map[string]bool{}),
		Entry("mixed: ran, skipped, and quarantined tests",
			map[string]map[string]int{
				"test-ran":                {"job-a": ExecRan},
				"test-skipped":            {"job-a": ExecSkipped},
				"[QUARANTINE] test-qskip": {"job-a": ExecQuarantined},
				"[QUARANTINE] test-qran":  {"job-a": ExecRan},
			},
			[]string{"[QUARANTINE] test-qran", "[QUARANTINE] test-qskip", "test-ran", "test-skipped"},
			map[string]bool{"test-skipped": true, "[QUARANTINE] test-qskip": true},
			map[string]bool{"[QUARANTINE] test-qskip": true, "[QUARANTINE] test-qran": true}),
	)
})
