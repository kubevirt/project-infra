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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/api/option"

	"github.com/joshdk/go-junit"

	"kubevirt.io/project-infra/pkg/flakefinder"
)

func TestTestExecutionReport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "test-execution-report Main Suite")
}

var _ = Describe("fetchPresubmitResults", func() {
	const repoPrefix = "pr-logs/pull/kubevirt_kubevirt/"
	const jobName = "pull-kubevirt-e2e-test"
	const junitXML = `<testsuite name="suite"><testcase name="test-1"/></testsuite>`
	var (
		client       *storage.Client
		cfg          *Config
		objects      map[string]string
		created      map[string]time.Time
		listingError bool
		start, end   time.Time
	)

	BeforeEach(func() {
		savedOpts := opts
		DeferCleanup(func() { opts = savedOpts })
		opts.org, opts.repo, opts.bucket = "kubevirt", "kubevirt", "test-bucket"
		cfg = &Config{jobPattern: regexp.MustCompile(`^pull-kubevirt-e2e-.*$`)}
		start = time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
		end = start.Add(24 * time.Hour)
		objects = map[string]string{}
		created = map[string]time.Time{}
		listingError = false

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer GinkgoRecover()
			Expect(r.Method).To(Equal(http.MethodGet))
			apiPath := "/storage/v1/b/test-bucket/o"
			if r.URL.Path == apiPath {
				if listingError {
					http.Error(w, "listing denied", http.StatusForbidden)
					return
				}
				Expect(r.URL.Query().Get("delimiter")).To(Equal("/"))
				prefix := r.URL.Query().Get("prefix")
				Expect(prefix).To(HavePrefix(repoPrefix))
				dirs := map[string]struct{}{}
				for object := range objects {
					if !strings.HasPrefix(object, prefix) {
						continue
					}
					if i := strings.Index(strings.TrimPrefix(object, prefix), "/"); i >= 0 {
						dirs[object[:len(prefix)+i+1]] = struct{}{}
					}
				}
				prefixes := sortedKeys(dirs)
				pageStart := 0
				if token := r.URL.Query().Get("pageToken"); token != "" {
					var err error
					pageStart, err = strconv.Atoi(token)
					Expect(err).NotTo(HaveOccurred())
				}
				pageEnd := min(pageStart+2, len(prefixes))
				response := map[string]interface{}{"prefixes": prefixes[pageStart:pageEnd]}
				if pageEnd < len(prefixes) {
					response["nextPageToken"] = strconv.Itoa(pageEnd)
				}
				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(response)).To(Succeed())
				return
			}
			Expect(r.URL.Path).To(HavePrefix(apiPath + "/"))
			object := strings.TrimPrefix(r.URL.Path, apiPath+"/")
			body, ok := objects[object]
			if !ok {
				http.NotFound(w, r)
				return
			}
			if r.URL.Query().Get("alt") == "media" {
				fmt.Fprint(w, body)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			Expect(json.NewEncoder(w).Encode(map[string]string{
				"name": object, "timeCreated": created[object].Format(time.RFC3339Nano),
			})).To(Succeed())
		}))
		DeferCleanup(server.Close)
		var err error
		client, err = storage.NewClient(context.Background(), option.WithEndpoint(server.URL+"/storage/v1/"),
			option.WithoutAuthentication(), storage.WithJSONReads())
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(client.Close)
	})

	It("discovers every stored PR across pages without latest-build aliases or a gap cutoff", func() {
		for _, pr := range []string{"50", "100", "300"} {
			buildDir := repoPrefix + pr + "/" + jobName + "/1/"
			objects[buildDir+"finished.json"] = "{}"
			created[buildDir+"finished.json"] = start.Add(time.Hour)
			objects[buildDir+"artifacts/junit.functest.xml"] = junitXML
		}
		for pr := 101; pr <= 160; pr++ {
			objects[fmt.Sprintf("%s%d/other-job/1/finished.json", repoPrefix, pr)] = "{}"
		}
		for _, pr := range []string{"0", "-1", "not-a-pr"} {
			objects[repoPrefix+pr+"/"+jobName+"/1/finished.json"] = "{}"
		}

		results, err := fetchPresubmitResults(context.Background(), client, cfg, start, end)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(3))
		prs := []int{}
		for _, result := range results {
			prs = append(prs, result.PR)
			Expect(result.Job).To(Equal(jobName))
		}
		sort.Ints(prs)
		Expect(prs).To(Equal([]int{50, 100, 300}))
	})

	It("ignores unrelated jobs and release-branch lanes", func() {
		for _, job := range []string{"other-job", jobName + "-1.9"} {
			buildDir := repoPrefix + "100/" + job + "/1/"
			objects[buildDir+"finished.json"] = "{}"
			created[buildDir+"finished.json"] = start.Add(time.Hour)
			objects[buildDir+"artifacts/junit.functest.xml"] = junitXML
		}
		results, err := fetchPresubmitResults(context.Background(), client, cfg, start, end)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(BeEmpty())
	})

	It("returns no results for an empty repository", func() {
		results, err := fetchPresubmitResults(context.Background(), client, cfg, start, end)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(BeEmpty())
	})

	It("reports a failure to discover PR directories", func() {
		listingError = true
		_, err := fetchPresubmitResults(context.Background(), client, cfg, start, end)
		Expect(err).To(MatchError(ContainSubstring("listing presubmit PR directories")))
	})

	DescribeTable("keeps earlier usable results when the newest build is unusable",
		func(newestState string) {
			olderDir := repoPrefix + "100/" + jobName + "/1/"
			objects[olderDir+"finished.json"] = "{}"
			created[olderDir+"finished.json"] = start.Add(time.Hour)
			objects[olderDir+"artifacts/junit.functest.xml"] = junitXML
			newerDir := repoPrefix + "100/" + jobName + "/2/"
			objects[newerDir+"started.json"] = "{}"
			if newestState != "running" {
				objects[newerDir+"finished.json"] = "{}"
				created[newerDir+"finished.json"] = start.Add(2 * time.Hour)
			}
			if newestState == "malformed" {
				objects[newerDir+"artifacts/junit.functest.xml"] = "<testsuite"
			}

			results, err := fetchPresubmitResults(context.Background(), client, cfg, start, end)
			Expect(err).NotTo(HaveOccurred())
			Expect(results).To(HaveLen(1))
			Expect(results[0].BuildNumber).To(Equal(1))
			Expect(results[0].PR).To(Equal(100))
		},
		Entry("still running", "running"),
		Entry("finished without JUnit", "missing-junit"),
		Entry("malformed JUnit", "malformed"),
	)

	It("aggregates execution evidence from every usable build in the window", func() {
		for build := 1; build <= 2; build++ {
			buildDir := fmt.Sprintf("%s100/%s/%d/", repoPrefix, jobName, build)
			objects[buildDir+"finished.json"] = "{}"
			created[buildDir+"finished.json"] = start.Add(time.Duration(build) * time.Hour)
			if build == 1 {
				objects[buildDir+"artifacts/junit.functest.xml"] = junitXML
			} else {
				objects[buildDir+"artifacts/junit.functest.xml"] = `<testsuite name="suite"><testcase name="test-1"><skipped/></testcase><testcase name="test-2"><failure/></testcase></testsuite>`
			}
		}

		results, err := fetchPresubmitResults(context.Background(), client, cfg, start, end)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(2))
		Expect(results[0].BuildNumber).To(Equal(2))
		Expect(results[1].BuildNumber).To(Equal(1))
		matrix := buildMatrix(results, regexp.MustCompile(`.*`))
		Expect(matrix["test-1"][jobName]).To(Equal(ExecRan))
		Expect(matrix["test-2"][jobName]).To(Equal(ExecRan))
	})

	It("filters by completion time without assuming it follows build order", func() {
		completionTimes := []time.Time{start, end, end.Add(time.Second), start.Add(-time.Second)}
		for i, completed := range completionTimes {
			buildDir := fmt.Sprintf("%s100/%s/%d/", repoPrefix, jobName, i+1)
			objects[buildDir+"finished.json"] = "{}"
			created[buildDir+"finished.json"] = completed
			objects[buildDir+"artifacts/junit.functest.xml"] = junitXML
		}

		results, err := fetchPresubmitResults(context.Background(), client, cfg, start, end)
		Expect(err).NotTo(HaveOccurred())
		Expect(results).To(HaveLen(2))
		Expect(results[0].BuildNumber).To(Equal(2))
		Expect(results[1].BuildNumber).To(Equal(1))
	})
})

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
		Entry("preserves execution when a later result is quarantined",
			[]*flakefinder.JobResult{
				newResult("job-a", passedTest("[QUARANTINE] test-q")),
				newResult("job-a", skippedTest("[QUARANTINE] test-q")),
			},
			"[QUARANTINE] test-q", "job-a", ExecRan),
		Entry("promotes quarantine to execution when a later result ran",
			[]*flakefinder.JobResult{
				newResult("job-a", skippedTest("[QUARANTINE] test-q")),
				newResult("job-a", passedTest("[QUARANTINE] test-q")),
			},
			"[QUARANTINE] test-q", "job-a", ExecRan),
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

	DescribeTable("retains quarantine annotations without classifying executed tests as not run",
		func(statuses []junit.Status) {
			const testName = "[QUARANTINE] test-q"
			results := []*flakefinder.JobResult{}
			for _, status := range statuses {
				results = append(results, newResult("job-a", junit.Test{Name: testName, Status: status}))
			}
			matrix := buildMatrix(results, matchAll)
			_, skipped, quarantined := classifyTests(matrix)
			Expect(matrix[testName]["job-a"]).To(Equal(ExecRan))
			Expect(skipped).NotTo(HaveKey(testName))
			Expect(quarantined).To(HaveKeyWithValue(testName, true))
		},
		Entry("passed then skipped", []junit.Status{junit.StatusPassed, junit.StatusSkipped}),
		Entry("skipped then passed", []junit.Status{junit.StatusSkipped, junit.StatusPassed}),
		Entry("failed then skipped", []junit.Status{junit.StatusFailed, junit.StatusSkipped}),
		Entry("skipped then failed", []junit.Status{junit.StatusSkipped, junit.StatusFailed}),
		Entry("repeated skipped observations around an execution", []junit.Status{junit.StatusSkipped, junit.StatusPassed, junit.StatusSkipped}),
	)
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
