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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	github "sigs.k8s.io/prow/pkg/github"
)

type fakeClient struct {
	branchProtection *github.BranchProtection
	branchProtErr    error

	pullRequests    []github.PullRequest
	pullRequestsErr error

	combinedStatuses  map[string]*github.CombinedStatus
	combinedStatusErr error

	statuses    map[string][]github.Status
	statusesErr error

	comments   map[int][]string
	commentErr error
}

func (f *fakeClient) GetBranchProtection(org, repo, branch string) (*github.BranchProtection, error) {
	return f.branchProtection, f.branchProtErr
}

func (f *fakeClient) GetPullRequests(org, repo string) ([]github.PullRequest, error) {
	return f.pullRequests, f.pullRequestsErr
}

func (f *fakeClient) GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error) {
	if f.combinedStatusErr != nil {
		return nil, f.combinedStatusErr
	}
	if cs, ok := f.combinedStatuses[ref]; ok {
		return cs, nil
	}
	return &github.CombinedStatus{}, nil
}

func (f *fakeClient) ListStatuses(org, repo, ref string) ([]github.Status, error) {
	if f.statusesErr != nil {
		return nil, f.statusesErr
	}
	return f.statuses[ref], nil
}

func (f *fakeClient) CreateComment(org, repo string, number int, comment string) error {
	if f.commentErr != nil {
		return f.commentErr
	}
	if f.comments == nil {
		f.comments = make(map[int][]string)
	}
	f.comments[number] = append(f.comments[number], comment)
	return nil
}

var _ = Describe("verifyRequiredCheck", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	It("succeeds when lane is a required check", func() {
		fc := &fakeClient{
			branchProtection: &github.BranchProtection{
				RequiredStatusChecks: &github.RequiredStatusChecks{
					Contexts: []string{lane, "other-check"},
				},
			},
		}
		Expect(verifyRequiredCheck(fc, "owner", "repo", lane)).To(Succeed())
	})

	It("returns error when lane is not a required check", func() {
		fc := &fakeClient{
			branchProtection: &github.BranchProtection{
				RequiredStatusChecks: &github.RequiredStatusChecks{
					Contexts: []string{"other-check"},
				},
			},
		}
		err := verifyRequiredCheck(fc, "owner", "repo", "nonexistent-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not a required status check"))
	})

	It("returns error when no required status checks are configured", func() {
		fc := &fakeClient{
			branchProtection: &github.BranchProtection{},
		}
		err := verifyRequiredCheck(fc, "owner", "repo", "some-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no required status checks"))
	})

	It("returns error when branch protection is nil", func() {
		fc := &fakeClient{}
		err := verifyRequiredCheck(fc, "owner", "repo", "some-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no required status checks"))
	})

	It("returns error when API fails", func() {
		fc := &fakeClient{
			branchProtErr: fmt.Errorf("api error"),
		}
		err := verifyRequiredCheck(fc, "owner", "repo", "some-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fetching branch protection"))
	})
})

var _ = Describe("listOpenPRs", func() {

	It("returns all PRs from the client", func() {
		fc := &fakeClient{
			pullRequests: []github.PullRequest{
				{Number: 1, Title: "PR 1"},
				{Number: 2, Title: "PR 2"},
				{Number: 3, Title: "PR 3"},
			},
		}
		prs, err := listOpenPRs(fc, "owner", "repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(prs).To(HaveLen(3))
		Expect(prs[0].Number).To(Equal(1))
		Expect(prs[2].Number).To(Equal(3))
	})

	It("returns error on API failure", func() {
		fc := &fakeClient{
			pullRequestsErr: fmt.Errorf("api error"),
		}
		_, err := listOpenPRs(fc, "owner", "repo")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listing PRs"))
	})

	It("returns empty slice for repo with no open PRs", func() {
		fc := &fakeClient{}
		prs, err := listOpenPRs(fc, "owner", "repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(prs).To(BeEmpty())
	})
})

var _ = Describe("getLaneStatus", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	It("finds the lane status and detects e2e statuses", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha123": {
					Statuses: []github.Status{
						{Context: lane, State: "pending"},
						{Context: "pull-kubevirt-e2e-k8s-1.35-sig-network", State: "success"},
					},
				},
			},
		}
		status, hasE2E, err := getLaneStatus(fc, "owner", "repo", "sha123", lane)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).NotTo(BeNil())
		Expect(status.State).To(Equal("pending"))
		Expect(hasE2E).To(BeTrue())
	})

	It("returns nil status when lane is not present", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha123": {
					Statuses: []github.Status{
						{Context: "other-check", State: "success"},
					},
				},
			},
		}
		status, hasE2E, err := getLaneStatus(fc, "owner", "repo", "sha123", lane)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(BeNil())
		Expect(hasE2E).To(BeFalse())
	})

	It("detects e2e statuses even when lane is missing", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha123": {
					Statuses: []github.Status{
						{Context: "pull-kubevirt-e2e-k8s-1.35-sig-storage", State: "success"},
					},
				},
			},
		}
		status, hasE2E, err := getLaneStatus(fc, "owner", "repo", "sha123", lane)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(BeNil())
		Expect(hasE2E).To(BeTrue())
	})

	It("returns error on API failure", func() {
		fc := &fakeClient{
			combinedStatusErr: fmt.Errorf("api error"),
		}
		_, _, err := getLaneStatus(fc, "owner", "repo", "sha123", lane)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("checkRawStatuses", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	It("returns hasURL=true when latest status has a target URL", func() {
		fc := &fakeClient{
			statuses: map[string][]github.Status{
				"sha123": {
					{Context: lane, TargetURL: "https://prow.example.com/job/123"},
				},
			},
		}
		hasURL, _ := checkRawStatuses(fc, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeTrue())
	})

	It("returns hasURL=false when latest status has no target URL (stuck)", func() {
		fc := &fakeClient{
			statuses: map[string][]github.Status{
				"sha123": {
					{Context: lane, TargetURL: ""},
				},
			},
		}
		hasURL, _ := checkRawStatuses(fc, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeFalse())
	})

	It("uses the first matching status (latest in reverse-chronological order)", func() {
		fc := &fakeClient{
			statuses: map[string][]github.Status{
				"sha123": {
					{Context: lane, TargetURL: "https://prow.example.com/job/456"},
					{Context: lane, TargetURL: ""},
				},
			},
		}
		hasURL, _ := checkRawStatuses(fc, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeTrue())
	})

	It("returns empty when lane has no statuses", func() {
		fc := &fakeClient{
			statuses: map[string][]github.Status{
				"sha123": {
					{Context: "other-check", TargetURL: "https://ci.example.com"},
				},
			},
		}
		hasURL, updatedAt := checkRawStatuses(fc, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeFalse())
		Expect(updatedAt).To(BeEmpty())
	})

	It("returns false on API error", func() {
		fc := &fakeClient{
			statusesErr: fmt.Errorf("api error"),
		}
		hasURL, updatedAt := checkRawStatuses(fc, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeFalse())
		Expect(updatedAt).To(BeEmpty())
	})

	It("returns false when no statuses exist at all", func() {
		fc := &fakeClient{}
		hasURL, updatedAt := checkRawStatuses(fc, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeFalse())
		Expect(updatedAt).To(BeEmpty())
	})
})

var _ = Describe("classifyPR", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	makePR := func(number int, sha string) *github.PullRequest {
		updated := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		return &github.PullRequest{
			Number:    number,
			Title:     fmt.Sprintf("PR %d", number),
			User:      github.User{Login: "user"},
			Head:      github.PullRequestBranch{SHA: sha},
			UpdatedAt: updated,
		}
	}

	It("classifies a PR with successful lane status as success", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-ok": {Statuses: []github.Status{{Context: lane, State: "success"}}},
			},
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(1, "sha-ok"))
		Expect(result.category).To(Equal("success"))
		Expect(result.pr).To(BeNil())
	})

	It("classifies a PR with failed lane status as failed", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-fail": {Statuses: []github.Status{{Context: lane, State: "failure"}}},
			},
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(2, "sha-fail"))
		Expect(result.category).To(Equal("failed"))
	})

	It("classifies a PR with error lane status as failed", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-err": {Statuses: []github.Status{{Context: lane, State: "error"}}},
			},
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(3, "sha-err"))
		Expect(result.category).To(Equal("failed"))
	})

	It("classifies a pending lane with target URL as running", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-run": {Statuses: []github.Status{{Context: lane, State: "pending"}}},
			},
			statuses: map[string][]github.Status{
				"sha-run": {{Context: lane, TargetURL: "https://prow.example.com/job/789"}},
			},
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(4, "sha-run"))
		Expect(result.category).To(Equal("running"))
		Expect(result.pr).To(BeNil())
	})

	It("classifies a pending lane without target URL as stuck", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-stuck": {Statuses: []github.Status{{Context: lane, State: "pending"}}},
			},
			statuses: map[string][]github.Status{
				"sha-stuck": {{Context: lane, TargetURL: ""}},
			},
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(5, "sha-stuck"))
		Expect(result.category).To(Equal("stuck"))
		Expect(result.pr).NotTo(BeNil())
		Expect(result.pr.Number).To(Equal(5))
		Expect(result.pr.StatusState).To(Equal("pending"))
	})

	It("classifies a PR with no lane status and no e2e statuses as success", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-clean": {Statuses: []github.Status{{Context: "ci/lint", State: "success"}}},
			},
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(6, "sha-clean"))
		Expect(result.category).To(Equal("success"))
	})

	It("classifies a PR with no lane status but e2e statuses as missing", func() {
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-miss": {Statuses: []github.Status{{Context: "pull-kubevirt-e2e-k8s-1.35-sig-network", State: "success"}}},
			},
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(7, "sha-miss"))
		Expect(result.category).To(Equal("missing"))
		Expect(result.pr).NotTo(BeNil())
		Expect(result.pr.StatusState).To(Equal("missing"))
	})

	It("classifies as failed when combined status API errors", func() {
		fc := &fakeClient{
			combinedStatusErr: fmt.Errorf("api error"),
		}
		result := classifyPR(fc, "owner", "repo", lane, makePR(8, "sha-api-err"))
		Expect(result.category).To(Equal("failed"))
	})
})

var _ = Describe("classifyPRs", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	It("aggregates classification results across multiple PRs", func() {
		updated := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		fc := &fakeClient{
			combinedStatuses: map[string]*github.CombinedStatus{
				"sha-1": {Statuses: []github.Status{{Context: lane, State: "success"}}},
				"sha-2": {Statuses: []github.Status{{Context: lane, State: "pending"}}},
				"sha-3": {Statuses: []github.Status{{Context: lane, State: "failure"}}},
			},
			statuses: map[string][]github.Status{
				"sha-2": {{Context: lane, TargetURL: ""}},
			},
		}

		prs := []github.PullRequest{
			{Number: 1, Title: "PR 1", User: github.User{Login: "u"}, Head: github.PullRequestBranch{SHA: "sha-1"}, UpdatedAt: updated},
			{Number: 2, Title: "PR 2", User: github.User{Login: "u"}, Head: github.PullRequestBranch{SHA: "sha-2"}, UpdatedAt: updated},
			{Number: 3, Title: "PR 3", User: github.User{Login: "u"}, Head: github.PullRequestBranch{SHA: "sha-3"}, UpdatedAt: updated},
		}

		summary, stuckPRs := classifyPRs(fc, "owner", "repo", lane, prs)

		Expect(summary.Total).To(Equal(3))
		Expect(summary.Success).To(Equal(1))
		Expect(summary.Stuck).To(Equal(1))
		Expect(summary.Failed).To(Equal(1))
		Expect(summary.Running).To(Equal(0))
		Expect(summary.Missing).To(Equal(0))
		Expect(stuckPRs).To(HaveLen(1))
		Expect(stuckPRs[0].Number).To(Equal(2))
	})
})

var _ = Describe("postComment", func() {

	It("posts a comment on the given PR", func() {
		fc := &fakeClient{}
		err := postComment(fc, "owner", "repo", 42, "/test my-lane")
		Expect(err).NotTo(HaveOccurred())
		Expect(fc.comments[42]).To(ConsistOf("/test my-lane"))
	})

	It("returns error on API failure", func() {
		fc := &fakeClient{
			commentErr: fmt.Errorf("forbidden"),
		}
		err := postComment(fc, "owner", "repo", 99, "/test my-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("commenting on PR #99"))
	})
})
