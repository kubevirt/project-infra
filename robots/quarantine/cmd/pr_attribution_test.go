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
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/pkg/searchci"
)

func presubmitBuildURL(pr int, buildID string) string {
	return fmt.Sprintf("https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/%d/pull-kubevirt-e2e-k8s-1.37-sig-compute-migrations/%s", pr, buildID)
}

func tideBatchBuildURL(id string) string {
	return "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/batch/pull-kubevirt-e2e-k8s-1.37-sig-compute-migrations/" + id
}

func attributed(resolved bool, prs ...int) buildAttribution {
	set := make(map[int]struct{}, len(prs))
	for _, pr := range prs {
		set[pr] = struct{}{}
	}
	return buildAttribution{prs: set, resolved: resolved}
}

var _ = Describe("pull request attribution", func() {
	DescribeTable("decides whether one pull request explains the failures",
		func(builds []buildAttribution, minIndependent int, wantDominated bool, wantPull int) {
			dominated, pull, _, _ := singlePRDominated(builds, minIndependent)
			Expect(dominated).To(Equal(wantDominated))
			if wantDominated {
				Expect(pull).To(Equal(wantPull))
			}
		},
		Entry("repeated runs of one pull request",
			[]buildAttribution{attributed(true, 18459), attributed(true, 18459), attributed(true, 18459)},
			2, true, 18459,
		),
		Entry("one pull request and Tide batches that contain it",
			[]buildAttribution{attributed(true, 18459), attributed(true, 18459, 19700), attributed(true, 18459, 20000)},
			2, true, 18459,
		),
		Entry("one outside failure does not corroborate a repeated pull request",
			[]buildAttribution{attributed(true, 18459), attributed(true, 18459), attributed(true, 18459), attributed(true, 20000)},
			2, true, 18459,
		),
		Entry("two pull requests failing once each",
			[]buildAttribution{attributed(true, 101), attributed(true, 202)},
			2, false, 0,
		),
		Entry("a repeated pull request plus two independent failures",
			[]buildAttribution{attributed(true, 18459), attributed(true, 18459), attributed(true, 18459), attributed(true, 300), attributed(true, 400)},
			2, false, 0,
		),
		Entry("Tide batches that share one pull request",
			[]buildAttribution{attributed(true, 18459, 11), attributed(true, 18459, 12)},
			2, true, 18459,
		),
		Entry("Tide batches with no pull request in common",
			[]buildAttribution{attributed(true, 11, 12), attributed(true, 13, 14)},
			2, false, 0,
		),
		Entry("an unresolved batch is not independent corroboration",
			[]buildAttribution{attributed(true, 18459), attributed(false), attributed(false)},
			2, true, 18459,
		),
		Entry("only unresolved builds",
			[]buildAttribution{attributed(false), attributed(false)},
			2, true, 0,
		),
		Entry("two distinct pull requests stay independent when a batch cannot be resolved",
			[]buildAttribution{attributed(true, 101), attributed(true, 202), attributed(false)},
			2, false, 0,
		),
		Entry("equal coverage reports the smaller pull number",
			[]buildAttribution{attributed(true, 8, 1), attributed(true, 8, 1)},
			2, true, 1,
		),
		Entry("values below 1 still reject a total overlap",
			[]buildAttribution{attributed(true, 18459), attributed(true, 18459)},
			0, true, 18459,
		),
		Entry("values below 1 treat a single outside failure as corroboration",
			[]buildAttribution{attributed(true, 18459), attributed(true, 18459), attributed(true, 6)},
			0, false, 0,
		),
	)

	It("attributes a direct presubmit without reading prowjob.json", func() {
		original := fetchProwJob
		fetchProwJob = func(string) ([]byte, error) {
			Fail("direct presubmit should not fetch prowjob.json")
			return nil, nil
		}
		DeferCleanup(func() { fetchProwJob = original })

		prs, resolved := attributeBuild(presubmitBuildURL(18459, "123"))
		Expect(resolved).To(BeTrue())
		Expect(prs).To(Equal(map[int]struct{}{18459: {}}))
	})

	It("reads the pull number from an org_repo presubmit URL", func() {
		prs, resolved := attributeBuild("https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/pr-logs/pull/nmstate_kubernetes-nmstate/1354/pull-kubernetes-nmstate-e2e-operator-k8s/1947282574778830848")
		Expect(resolved).To(BeTrue())
		Expect(prs).To(Equal(map[int]struct{}{1354: {}}))
	})

	It("attributes a Tide batch from its pull requests", func() {
		original := pullsForBatchBuild
		pullsForBatchBuild = func(string) ([]int, error) {
			return []int{18459, 19001}, nil
		}
		DeferCleanup(func() { pullsForBatchBuild = original })

		prs, resolved := attributeBuild(tideBatchBuildURL("99"))
		Expect(resolved).To(BeTrue())
		Expect(prs).To(Equal(map[int]struct{}{18459: {}, 19001: {}}))
	})

	It("leaves a Tide batch unresolved when it lists no pull requests", func() {
		original := pullsForBatchBuild
		pullsForBatchBuild = func(string) ([]int, error) {
			return nil, nil
		}
		DeferCleanup(func() { pullsForBatchBuild = original })

		_, resolved := attributeBuild(tideBatchBuildURL("empty"))
		Expect(resolved).To(BeFalse())
	})

	It("leaves a Tide batch unresolved when prowjob.json cannot be read", func() {
		original := pullsForBatchBuild
		pullsForBatchBuild = func(string) ([]int, error) {
			return nil, fmt.Errorf("prowjob unavailable")
		}
		DeferCleanup(func() { pullsForBatchBuild = original })

		_, resolved := attributeBuild(tideBatchBuildURL("missing"))
		Expect(resolved).To(BeFalse())
	})

	It("leaves an unrecognized build URL unresolved", func() {
		_, resolved := attributeBuild("https://example.com/not-a-prow-build")
		Expect(resolved).To(BeFalse())
	})

	DescribeTable("builds a prowjob.json URL",
		func(buildURL, expected string) {
			Expect(prowJobURL(buildURL)).To(Equal(expected))
		},
		Entry("deck gs view",
			tideBatchBuildURL("99"),
			"https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/batch/pull-kubevirt-e2e-k8s-1.37-sig-compute-migrations/99/prowjob.json",
		),
		Entry("deck gcs view",
			"https://prow.ci.kubevirt.io/view/gcs/kubevirt-prow/pr-logs/pull/batch/job/7",
			"https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/batch/job/7/prowjob.json",
		),
		Entry("storage object",
			"https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/batch/job/7",
			"https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/batch/job/7/prowjob.json",
		),
		Entry("storage object that is already prowjob.json",
			"https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/batch/job/7/prowjob.json",
			"https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/batch/job/7/prowjob.json",
		),
	)

	It("rejects a build URL that is not stored in GCS", func() {
		_, err := prowJobURL("https://example.com/view/not-gcs/job/1")
		Expect(err).To(HaveOccurred())
	})

	It("reads spec.refs.pulls and ignores auxiliary repos", func() {
		body := []byte(`{
			"spec": {
				"refs": {"org": "kubevirt", "repo": "kubevirt", "pulls": [{"number": 18459}, {"number": 0}, {"number": 19001}]},
				"extra_refs": [{"org": "kubevirt", "repo": "kubevirtci", "pulls": [{"number": 999}]}]
			}
		}`)
		Expect(pullNumbersFromProwJob(body)).To(Equal([]int{18459, 19001}))
	})

	It("returns an error for malformed prowjob.json", func() {
		_, err := pullNumbersFromProwJob([]byte(`{`))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("could not decode prowjob"))
	})

	It("loads batch pulls through prowjob.json", func() {
		const prowJob = `{
			"spec": {
				"refs": {"pulls": [{"number": 18459}, {"number": 19700}]},
				"extra_refs": [{"pulls": [{"number": 1}]}]
			}
		}`
		original := fetchProwJob
		fetchProwJob = func(rawURL string) ([]byte, error) {
			Expect(rawURL).To(Equal("https://storage.googleapis.com/kubevirt-prow/pr-logs/pull/batch/pull-kubevirt-e2e-k8s-1.37-sig-compute-migrations/99/prowjob.json"))
			return []byte(prowJob), nil
		}
		DeferCleanup(func() { fetchProwJob = original })

		numbers, err := defaultPullsForBatchBuild(tideBatchBuildURL("99"))
		Expect(err).NotTo(HaveOccurred())
		Expect(numbers).To(Equal([]int{18459, 19700}))
	})

	It("surfaces a missing prowjob.json", func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		}))
		DeferCleanup(server.Close)

		_, err := fetchProwJob(server.URL)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("404"))
	})

	It("attributes only failures inside the recent window", func() {
		impact := searchci.Impact{
			BuildURLs: []searchci.JobBuildURL{
				{URL: presubmitBuildURL(18459, "recent-a"), Interval: 4 * time.Hour},
				{URL: presubmitBuildURL(18459, "recent-b"), Interval: 30 * time.Hour},
				{URL: presubmitBuildURL(300, "old-a"), Interval: 200 * time.Hour},
				{URL: presubmitBuildURL(400, "old-b"), Interval: 240 * time.Hour},
			},
		}
		dominated, pull, covered, resolved := impactDominatedBySinglePR(impact, 72*time.Hour, 2)
		Expect(dominated).To(BeTrue())
		Expect(pull).To(Equal(18459))
		Expect(covered).To(Equal(2))
		Expect(resolved).To(Equal(2))
	})

	It("keeps cross-PR failures inside the recent window when older runs are one pull request", func() {
		impact := searchci.Impact{
			BuildURLs: []searchci.JobBuildURL{
				{URL: presubmitBuildURL(101, "recent-a"), Interval: 4 * time.Hour},
				{URL: presubmitBuildURL(202, "recent-b"), Interval: 36 * time.Hour},
				{URL: presubmitBuildURL(18459, "old-a"), Interval: 200 * time.Hour},
				{URL: presubmitBuildURL(18459, "old-b"), Interval: 220 * time.Hour},
				{URL: presubmitBuildURL(18459, "old-c"), Interval: 240 * time.Hour},
			},
		}
		dominated, _, _, resolved := impactDominatedBySinglePR(impact, 72*time.Hour, 2)
		Expect(dominated).To(BeFalse())
		Expect(resolved).To(Equal(2))
	})
})
