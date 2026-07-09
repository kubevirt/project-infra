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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/google/go-github/v32/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func newTestServer(mux *http.ServeMux) (*github.Client, *httptest.Server) {
	server := httptest.NewServer(mux)
	DeferCleanup(server.Close)

	client := github.NewClient(nil)
	baseURL, _ := url.Parse(server.URL + "/")
	client.BaseURL = baseURL
	return client, server
}

func newTestClient(mux *http.ServeMux) *github.Client {
	client, _ := newTestServer(mux)
	return client
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(v)
	Expect(err).NotTo(HaveOccurred())
	return data
}

var _ = Describe("verifyRequiredCheck", func() {

	It("succeeds when lane is a required check", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/branches/main/protection", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"required_status_checks":{"strict":true,"contexts":["pull-kubevirt-e2e-k8s-1.36-sig-compute","other-check"]}}`)
		})
		client := newTestClient(mux)

		err := verifyRequiredCheck(context.Background(), client, "owner", "repo", "pull-kubevirt-e2e-k8s-1.36-sig-compute")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error when lane is not a required check", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/branches/main/protection", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"required_status_checks":{"strict":true,"contexts":["other-check"]}}`)
		})
		client := newTestClient(mux)

		err := verifyRequiredCheck(context.Background(), client, "owner", "repo", "nonexistent-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not a required status check"))
	})

	It("returns error when no required status checks are configured", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/branches/main/protection", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{}`)
		})
		client := newTestClient(mux)

		err := verifyRequiredCheck(context.Background(), client, "owner", "repo", "some-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no required status checks"))
	})

	It("returns error when API fails", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/branches/main/protection", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		client := newTestClient(mux)

		err := verifyRequiredCheck(context.Background(), client, "owner", "repo", "some-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("fetching branch protection"))
	})
})

var _ = Describe("listOpenPRs", func() {

	It("fetches all open PRs across pages", func() {
		mux := http.NewServeMux()
		client, server := newTestServer(mux)
		mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
			page := r.URL.Query().Get("page")
			w.Header().Set("Content-Type", "application/json")
			if page == "" || page == "1" {
				w.Header().Set("Link", fmt.Sprintf(`<%s/repos/owner/repo/pulls?page=2>; rel="next"`, server.URL))
				w.Write(mustJSON([]*github.PullRequest{
					{Number: intPtr(1), Title: strPtr("PR 1")},
					{Number: intPtr(2), Title: strPtr("PR 2")},
				}))
			} else {
				w.Write(mustJSON([]*github.PullRequest{
					{Number: intPtr(3), Title: strPtr("PR 3")},
				}))
			}
		})

		prs, err := listOpenPRs(context.Background(), client, "owner", "repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(prs).To(HaveLen(3))
		Expect(prs[0].GetNumber()).To(Equal(1))
		Expect(prs[2].GetNumber()).To(Equal(3))
	})

	It("returns error on API failure", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		client := newTestClient(mux)

		_, err := listOpenPRs(context.Background(), client, "owner", "repo")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("listing PRs"))
	})

	It("returns empty slice for repo with no open PRs", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.PullRequest{}))
		})
		client := newTestClient(mux)

		prs, err := listOpenPRs(context.Background(), client, "owner", "repo")
		Expect(err).NotTo(HaveOccurred())
		Expect(prs).To(BeEmpty())
	})
})

var _ = Describe("getCombinedStatusInfo", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	It("finds the lane status and detects e2e statuses", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("pending")},
					{Context: strPtr("pull-kubevirt-e2e-k8s-1.35-sig-network"), State: strPtr("success")},
				},
			}))
		})
		client := newTestClient(mux)

		status, hasE2E, err := getCombinedStatusInfo(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).NotTo(BeNil())
		Expect(status.GetState()).To(Equal("pending"))
		Expect(hasE2E).To(BeTrue())
	})

	It("returns nil status when lane is not present", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr("other-check"), State: strPtr("success")},
				},
			}))
		})
		client := newTestClient(mux)

		status, hasE2E, err := getCombinedStatusInfo(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(BeNil())
		Expect(hasE2E).To(BeFalse())
	})

	It("detects e2e statuses even when lane is missing", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr("pull-kubevirt-e2e-k8s-1.35-sig-storage"), State: strPtr("success")},
				},
			}))
		})
		client := newTestClient(mux)

		status, hasE2E, err := getCombinedStatusInfo(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(err).NotTo(HaveOccurred())
		Expect(status).To(BeNil())
		Expect(hasE2E).To(BeTrue())
	})

	It("returns error on API failure", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/status", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		client := newTestClient(mux)

		_, _, err := getCombinedStatusInfo(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("checkRawStatuses", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	It("returns hasURL=true when latest status has a target URL", func() {
		ts := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/statuses", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.RepoStatus{
				{Context: strPtr(lane), TargetURL: strPtr("https://prow.example.com/job/123"), UpdatedAt: &ts},
			}))
		})
		client := newTestClient(mux)

		hasURL, updatedAt := checkRawStatuses(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeTrue())
		Expect(updatedAt).To(Equal("2026-06-10T12:00:00Z"))
	})

	It("returns hasURL=false when latest status has no target URL (stuck)", func() {
		ts := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/statuses", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.RepoStatus{
				{Context: strPtr(lane), TargetURL: strPtr(""), UpdatedAt: &ts},
			}))
		})
		client := newTestClient(mux)

		hasURL, _ := checkRawStatuses(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeFalse())
	})

	It("picks the latest status when multiple exist for the lane", func() {
		older := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
		newer := time.Date(2026, 6, 10, 14, 0, 0, 0, time.UTC)
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/statuses", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.RepoStatus{
				{Context: strPtr(lane), TargetURL: strPtr(""), UpdatedAt: &older},
				{Context: strPtr(lane), TargetURL: strPtr("https://prow.example.com/job/456"), UpdatedAt: &newer},
			}))
		})
		client := newTestClient(mux)

		hasURL, updatedAt := checkRawStatuses(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeTrue())
		Expect(updatedAt).To(Equal("2026-06-10T14:00:00Z"))
	})

	It("returns empty when lane has no statuses", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha123/statuses", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.RepoStatus{
				{Context: strPtr("other-check"), TargetURL: strPtr("https://ci.example.com")},
			}))
		})
		client := newTestClient(mux)

		hasURL, updatedAt := checkRawStatuses(context.Background(), client, "owner", "repo", "sha123", lane)
		Expect(hasURL).To(BeFalse())
		Expect(updatedAt).To(BeEmpty())
	})
})

var _ = Describe("classifyPR", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	makePR := func(number int, sha string) *github.PullRequest {
		updated := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		return &github.PullRequest{
			Number:    intPtr(number),
			Title:     strPtr(fmt.Sprintf("PR %d", number)),
			User:      &github.User{Login: strPtr("user")},
			Head:      &github.PullRequestBranch{SHA: strPtr(sha)},
			Draft:     boolPtr(false),
			UpdatedAt: &updated,
		}
	}

	It("classifies a PR with successful lane status as success", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-ok/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("success")},
				},
			}))
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(1, "sha-ok"))
		Expect(result.category).To(Equal("success"))
		Expect(result.pr).To(BeNil())
	})

	It("classifies a PR with failed lane status as failed", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-fail/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("failure")},
				},
			}))
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(2, "sha-fail"))
		Expect(result.category).To(Equal("failed"))
		Expect(result.pr).To(BeNil())
	})

	It("classifies a PR with error lane status as failed", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-err/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("error")},
				},
			}))
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(3, "sha-err"))
		Expect(result.category).To(Equal("failed"))
	})

	It("classifies a pending lane with target URL as running", func() {
		ts := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-run/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("pending")},
				},
			}))
		})
		mux.HandleFunc("/repos/owner/repo/commits/sha-run/statuses", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.RepoStatus{
				{Context: strPtr(lane), TargetURL: strPtr("https://prow.example.com/job/789"), UpdatedAt: &ts},
			}))
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(4, "sha-run"))
		Expect(result.category).To(Equal("running"))
		Expect(result.pr).To(BeNil())
	})

	It("classifies a pending lane without target URL as stuck", func() {
		ts := time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC)
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-stuck/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("pending")},
				},
			}))
		})
		mux.HandleFunc("/repos/owner/repo/commits/sha-stuck/statuses", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.RepoStatus{
				{Context: strPtr(lane), TargetURL: strPtr(""), UpdatedAt: &ts},
			}))
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(5, "sha-stuck"))
		Expect(result.category).To(Equal("stuck"))
		Expect(result.pr).NotTo(BeNil())
		Expect(result.pr.Number).To(Equal(5))
		Expect(result.pr.StatusState).To(Equal("pending"))
	})

	It("classifies a PR with no lane status and no e2e statuses as success", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-clean/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr("ci/lint"), State: strPtr("success")},
				},
			}))
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(6, "sha-clean"))
		Expect(result.category).To(Equal("success"))
	})

	It("classifies a PR with no lane status but e2e statuses as missing", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-miss/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr("pull-kubevirt-e2e-k8s-1.35-sig-network"), State: strPtr("success")},
				},
			}))
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(7, "sha-miss"))
		Expect(result.category).To(Equal("missing"))
		Expect(result.pr).NotTo(BeNil())
		Expect(result.pr.StatusState).To(Equal("missing"))
	})

	It("classifies as failed when combined status API errors", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/commits/sha-api-err/status", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		client := newTestClient(mux)

		result := classifyPR(context.Background(), client, "owner", "repo", lane, makePR(8, "sha-api-err"))
		Expect(result.category).To(Equal("failed"))
	})
})

var _ = Describe("classifyPRs", func() {

	const lane = "pull-kubevirt-e2e-k8s-1.36-sig-compute"

	It("aggregates classification results across multiple PRs", func() {
		updated := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		ts := time.Date(2026, 6, 10, 11, 0, 0, 0, time.UTC)
		mux := http.NewServeMux()

		// PR 1: success
		mux.HandleFunc("/repos/owner/repo/commits/sha-1/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("success")},
				},
			}))
		})

		// PR 2: stuck (pending, no target URL)
		mux.HandleFunc("/repos/owner/repo/commits/sha-2/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("pending")},
				},
			}))
		})
		mux.HandleFunc("/repos/owner/repo/commits/sha-2/statuses", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON([]*github.RepoStatus{
				{Context: strPtr(lane), TargetURL: strPtr(""), UpdatedAt: &ts},
			}))
		})

		// PR 3: failed
		mux.HandleFunc("/repos/owner/repo/commits/sha-3/status", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.CombinedStatus{
				Statuses: []*github.RepoStatus{
					{Context: strPtr(lane), State: strPtr("failure")},
				},
			}))
		})

		client := newTestClient(mux)

		prs := []*github.PullRequest{
			{Number: intPtr(1), Title: strPtr("PR 1"), User: &github.User{Login: strPtr("u")}, Head: &github.PullRequestBranch{SHA: strPtr("sha-1")}, Draft: boolPtr(false), UpdatedAt: &updated},
			{Number: intPtr(2), Title: strPtr("PR 2"), User: &github.User{Login: strPtr("u")}, Head: &github.PullRequestBranch{SHA: strPtr("sha-2")}, Draft: boolPtr(false), UpdatedAt: &updated},
			{Number: intPtr(3), Title: strPtr("PR 3"), User: &github.User{Login: strPtr("u")}, Head: &github.PullRequestBranch{SHA: strPtr("sha-3")}, Draft: boolPtr(false), UpdatedAt: &updated},
		}

		summary, stuckPRs := classifyPRs(context.Background(), client, "owner", "repo", lane, prs)

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
		var capturedBody string
		var capturedPR int
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/issues/42/comments", func(w http.ResponseWriter, r *http.Request) {
			Expect(r.Method).To(Equal("POST"))
			var body github.IssueComment
			Expect(json.NewDecoder(r.Body).Decode(&body)).To(Succeed())
			capturedBody = body.GetBody()
			capturedPR = 42
			w.Header().Set("Content-Type", "application/json")
			w.Write(mustJSON(github.IssueComment{
				ID:      intPtr64(1),
				Body:    body.Body,
				HTMLURL: strPtr("https://github.com/owner/repo/pull/42#issuecomment-1"),
			}))
		})
		client := newTestClient(mux)

		err := postComment(context.Background(), client, "owner", "repo", 42, "/test my-lane")
		Expect(err).NotTo(HaveOccurred())
		Expect(capturedPR).To(Equal(42))
		Expect(capturedBody).To(Equal("/test my-lane"))
	})

	It("returns error on API failure", func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/repos/owner/repo/issues/99/comments", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		})
		client := newTestClient(mux)

		err := postComment(context.Background(), client, "owner", "repo", 99, "/test my-lane")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("commenting on PR #99"))
	})
})

func intPtr64(i int64) *int64 { return &i }
