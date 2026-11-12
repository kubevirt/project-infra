package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
)

type reconcileClient struct {
	*fakegithub.FakeClient
	prs      []github.PullRequest
	list     func() ([]github.PullRequest, error)
	get      func(int) (*github.PullRequest, error)
	status   func(string) (*github.CombinedStatus, error)
	post     func(context.Context, int, string) error
	botError error
}

func (c *reconcileClient) GetPullRequests(_, _ string) ([]github.PullRequest, error) {
	if c.list != nil {
		return c.list()
	}
	return c.prs, nil
}

func (c *reconcileClient) GetPullRequest(org, repo string, number int) (*github.PullRequest, error) {
	if c.get != nil {
		return c.get(number)
	}
	return c.FakeClient.GetPullRequest(org, repo, number)
}

func (c *reconcileClient) GetCombinedStatus(org, repo, sha string) (*github.CombinedStatus, error) {
	if c.status != nil {
		return c.status(sha)
	}
	return c.FakeClient.GetCombinedStatus(org, repo, sha)
}

func (c *reconcileClient) CreateCommentWithContext(ctx context.Context, org, repo string, number int, body string) error {
	if c.post != nil {
		return c.post(ctx, number, body)
	}
	return c.FakeClient.CreateCommentWithContext(ctx, org, repo, number, body)
}

func (c *reconcileClient) BotUserCheckerWithContext(ctx context.Context) (func(string) bool, error) {
	if c.botError != nil {
		return nil, c.botError
	}
	return c.FakeClient.BotUserCheckerWithContext(ctx)
}

func recoveryPR(number int, sha string) github.PullRequest {
	mergeable := true
	return github.PullRequest{
		Number: number, State: "open", Mergable: &mergeable,
		Head: github.PullRequestBranch{SHA: sha},
		Base: github.PullRequestBranch{Ref: "main", Repo: github.Repo{
			FullName: "kubevirt/kubevirt", Name: "kubevirt", Owner: github.User{Login: "kubevirt"},
		}},
	}
}

// Use the actual HTTP config loader without mutating the package-global test loader.
func serveRecoveryConfig(h *GitHubEventsHandler, jobs []config.Presubmit) (*atomic.Value, *atomic.Int32) {
	current := &atomic.Value{}
	current.Store(jobs)
	loads := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer GinkgoRecover()
		if r.URL.Path == "/config" {
			_, err := fmt.Fprint(w, "{}")
			Expect(err).NotTo(HaveOccurred())
			return
		}
		loads.Add(1)
		cfg := config.Config{JobConfig: config.JobConfig{PresubmitsStatic: map[string][]config.Presubmit{
			"kubevirt/kubevirt": current.Load().([]config.Presubmit),
		}}}
		Expect(json.NewEncoder(w).Encode(cfg)).To(Succeed())
	}))
	DeferCleanup(server.Close)
	h.prowLocation = server.URL
	h.prowConfigPath = "config"
	h.jobsConfigBase = "jobs"
	return current, loads
}

var _ = Describe("phase reconciliation", func() {
	var h *GitHubEventsHandler
	var client *reconcileClient
	var pr github.PullRequest
	var jobs []config.Presubmit
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		client = &reconcileClient{FakeClient: fakegithub.NewFakeClient()}
		pr = recoveryPR(42, "head-sha")
		client.prs = []github.PullRequest{pr}
		client.PullRequests[42] = &pr
		client.CombinedStatuses[pr.Head.SHA] = &github.CombinedStatus{Statuses: []github.Status{
			{Context: "build", State: github.StatusSuccess},
		}}
		jobs = []config.Presubmit{presubmit("build", "0"), presubmitWithContext("network", "ci/network", "1"), presubmit("merge-only", "")}
		for i := range jobs {
			jobs[i].Spec = &v1.PodSpec{Containers: []v1.Container{{Image: "test-image"}}}
		}
		h = NewGitHubEventsHandler(nil, logrus.New(), client, "", "", "", nil)
		h.reconcileLimiter = rate.NewLimiter(rate.Inf, 1)
	})

	It("recovers a missed phase directly without rerunning phase zero", func() {
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeTrue())
		Expect(client.IssueCommentsAdded).To(HaveLen(1))
		Expect(client.IssueCommentsAdded[0]).To(ContainSubstring("Phase 0 jobs completed, triggering phase 1 presubmits:\n/test network\n"))
		Expect(client.IssueCommentsAdded[0]).To(ContainSubstring(phaseRequestMarker(pr.Head.SHA, 1, "network")))
		Expect(client.IssueCommentsAdded[0]).NotTo(ContainSubstring("/test build"))
		Expect(client.IssueCommentsAdded[0]).NotTo(ContainSubstring("/test merge-only"))
	})

	DescribeTable("does not request jobs whose status already exists", func(state string) {
		client.CombinedStatuses[pr.Head.SHA].Statuses = append(client.CombinedStatuses[pr.Head.SHA].Statuses,
			github.Status{Context: "ci/network", State: state})
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeFalse())
		Expect(client.IssueCommentsAdded).To(BeEmpty())
	}, Entry("pending", github.StatusPending), Entry("failed", github.StatusFailure), Entry("error", github.StatusError), Entry("succeeded", github.StatusSuccess))

	DescribeTable("blocks on incomplete phase zero", func(state string) {
		client.CombinedStatuses[pr.Head.SHA].Statuses = nil
		if state != "missing" {
			client.CombinedStatuses[pr.Head.SHA].Statuses = []github.Status{{Context: "build", State: state}}
		}
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeFalse())
	}, Entry("missing", "missing"), Entry("pending", github.StatusPending), Entry("failed", github.StatusFailure), Entry("error", github.StatusError))

	DescribeTable("eligibility", func(change func(*github.PullRequest), expected bool) {
		change(&pr)
		Expect(phasePREligible(pr)).To(Equal(expected))
	},
		Entry("open main", func(*github.PullRequest) {}, true),
		Entry("master", func(p *github.PullRequest) { p.Base.Ref = "master" }, true),
		Entry("draft", func(p *github.PullRequest) { p.Draft = true }, false),
		Entry("closed", func(p *github.PullRequest) { p.State = "closed" }, false),
		Entry("merged", func(p *github.PullRequest) { p.Merged = true }, false),
		Entry("conflict", func(p *github.PullRequest) { p.Mergable = new(bool) }, false),
		Entry("release branch", func(p *github.PullRequest) { p.Base.Ref = "release-1.0" }, false),
		Entry("missing HEAD", func(p *github.PullRequest) { p.Head.SHA = "" }, false),
	)

	It("requests only the missing subset of the next phase", func() {
		jobs = append(jobs, presubmit("already-started", "1"))
		client.CombinedStatuses[pr.Head.SHA].Statuses = append(client.CombinedStatuses[pr.Head.SHA].Statuses,
			github.Status{Context: "already-started", State: github.StatusPending})
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeTrue())
		Expect(client.IssueCommentsAdded[0]).NotTo(ContainSubstring("/test already-started"))
	})

	It("advances one next configured phase, including noncontiguous phases", func() {
		jobs = append(jobs, presubmit("later", "3"))
		client.CombinedStatuses[pr.Head.SHA].Statuses = append(client.CombinedStatuses[pr.Head.SHA].Statuses,
			github.Status{Context: "ci/network", State: github.StatusSuccess})
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeTrue())
		Expect(client.IssueCommentsAdded[0]).To(ContainSubstring("Phase 1 jobs completed, triggering phase 3 presubmits:\n/test later"))
	})

	It("rejects a stale HEAD before reading statuses", func() {
		observed := pr
		observed.Head.SHA = "old-head"
		requested, err := h.reconcilePR(ctx, observed, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeFalse())
	})

	It("rechecks HEAD immediately before posting", func() {
		calls := 0
		client.get = func(int) (*github.PullRequest, error) {
			calls++
			copy := pr
			if calls == 2 {
				copy.Head.SHA = "new-head"
			}
			return &copy, nil
		}
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeFalse())
		Expect(client.IssueCommentsAdded).To(BeEmpty())
	})

	It("rechecks statuses immediately before posting", func() {
		calls := 0
		client.status = func(string) (*github.CombinedStatus, error) {
			calls++
			result := github.CombinedStatus{Statuses: []github.Status{{Context: "build", State: github.StatusSuccess}}}
			if calls == 2 {
				result.Statuses = append(result.Statuses, github.Status{Context: "ci/network", State: github.StatusPending})
			}
			return &result, nil
		}
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeFalse())
		Expect(client.IssueCommentsAdded).To(BeEmpty())
	})

	It("drops newly started jobs from a partially missing phase before posting", func() {
		jobs = append(jobs, presubmit("storage", "1"))
		calls := 0
		client.status = func(string) (*github.CombinedStatus, error) {
			calls++
			result := github.CombinedStatus{Statuses: []github.Status{{Context: "build", State: github.StatusSuccess}}}
			if calls == 2 {
				result.Statuses = append(result.Statuses, github.Status{Context: "ci/network", State: github.StatusPending})
			}
			return &result, nil
		}
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeTrue())
		Expect(client.IssueCommentsAdded[0]).To(ContainSubstring("/test storage"))
		Expect(client.IssueCommentsAdded[0]).NotTo(ContainSubstring("/test network"))
	})

	DescribeTable("propagates failures during final rechecks", func(failAt int) {
		calls := 0
		client.get = func(int) (*github.PullRequest, error) {
			calls++
			if calls == failAt {
				return nil, errors.New("PR unavailable")
			}
			return &pr, nil
		}
		_, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).To(MatchError(ContainSubstring("PR unavailable")))
		Expect(client.IssueCommentsAdded).To(BeEmpty())
	}, Entry("initial PR refresh", 1), Entry("last PR refresh", 2))

	It("rejects a PR that becomes a draft during comment lookup", func() {
		calls := 0
		client.get = func(int) (*github.PullRequest, error) {
			calls++
			copy := pr
			copy.Draft = calls == 2
			return &copy, nil
		}
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeFalse())
	})

	It("deduplicates concurrent requests before statuses appear", func() {
		var callers sync.WaitGroup
		for range 8 {
			callers.Go(func() {
				defer GinkgoRecover()
				_, err := h.reconcilePR(ctx, pr, jobs, false)
				Expect(err).NotTo(HaveOccurred())
			})
		}
		callers.Wait()
		Expect(client.IssueCommentsAdded).To(HaveLen(1))
	})

	It("shares request protection with status webhooks", func() {
		h.cache.set(jobs)
		_, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		h.handleStatusEvent(logrus.NewEntry(logrus.New()), &github.StatusEvent{
			SHA: pr.Head.SHA, State: github.StatusSuccess, Context: "build",
			Repo: github.Repo{Name: "kubevirt", Owner: github.User{Login: "kubevirt"}},
		})
		h.handlePhaseTransition(logrus.NewEntry(logrus.New()), pr, map[string]string{"build": github.StatusSuccess})
		Expect(client.IssueCommentsAdded).To(HaveLen(1))
	})

	It("suppresses a recent bot request after a restart", func() {
		bot, err := client.BotUser()
		Expect(err).NotTo(HaveOccurred())
		client.IssueComments[42] = []github.IssueComment{{
			Body: phaseRequestMarker(pr.Head.SHA, 1, "network"), CreatedAt: time.Now(), User: github.User{Login: bot.Login},
		}}
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeFalse())
		Expect(h.requests).To(BeEmpty())
	})

	DescribeTable("does not permanently suppress missing jobs", func(age time.Duration, author, sha string) {
		bot, err := client.BotUser()
		Expect(err).NotTo(HaveOccurred())
		if author == "bot" {
			author = bot.Login
		}
		client.IssueComments[42] = []github.IssueComment{{
			Body: phaseRequestMarker(sha, 1, "network"), CreatedAt: time.Now().Add(-age), User: github.User{Login: author},
		}}
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeTrue())
	}, Entry("expired request", 11*time.Minute, "bot", "head-sha"), Entry("different head", time.Minute, "bot", "old-head"), Entry("forged marker", time.Minute, "someone-else", "head-sha"))

	It("prunes expired in-memory requests", func() {
		h.requests = map[string]time.Time{"expired": time.Now().Add(-time.Hour)}
		_, err := h.unrequestedJobs(ctx, pr, jobs[1:2], 1, map[string]string{})
		Expect(err).NotTo(HaveOccurred())
		Expect(h.requests).To(BeEmpty())
	})

	It("does not suppress a new job added to an already requested phase", func() {
		_, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		jobs = append(jobs, presubmit("new-e2e", "1"))
		requested, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeTrue())
		Expect(client.IssueCommentsAdded[1]).To(ContainSubstring("/test new-e2e"))
		Expect(client.IssueCommentsAdded[1]).NotTo(ContainSubstring("/test network"))
	})

	It("allows another PR with the same HEAD to request its own jobs", func() {
		_, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		other := pr
		other.Number = 43
		client.PullRequests[43] = &other
		requested, err := h.reconcilePR(ctx, other, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		Expect(requested).To(BeTrue())
		Expect(client.IssueCommentsAdded).To(HaveLen(2))
	})

	It("propagates comment and bot identity lookup errors", func() {
		client.ListIssueCommentsWithContextError = errors.New("comments unavailable")
		_, err := h.unrequestedJobs(ctx, pr, jobs[1:2], 1, nil)
		Expect(err).To(MatchError(ContainSubstring("comments unavailable")))
		client.ListIssueCommentsWithContextError = nil
		client.botError = errors.New("identity unavailable")
		_, err = h.unrequestedJobs(ctx, pr, jobs[1:2], 1, nil)
		Expect(err).To(MatchError(ContainSubstring("identity unavailable")))
	})

	It("does not record an unsuccessful comment as requested", func() {
		client.post = func(context.Context, int, string) error { return errors.New("post failed") }
		_, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).To(MatchError(ContainSubstring("post failed")))
		Expect(h.requests).To(BeEmpty())
	})

	It("honors cancellation while waiting for a paced request", func() {
		h.reconcileLimiter = rate.NewLimiter(rate.Every(time.Hour), 1)
		Expect(h.reconcileLimiter.Allow()).To(BeTrue())
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := h.reconcilePR(canceled, pr, jobs, true)
		Expect(err).To(HaveOccurred())
		Expect(client.IssueCommentsAdded).To(BeEmpty())
	})

	It("paces recovery requests without delaying scans", func() {
		h.reconcileLimiter = rate.NewLimiter(rate.Every(40*time.Millisecond), 1)
		_, err := h.reconcilePR(ctx, pr, jobs, true)
		Expect(err).NotTo(HaveOccurred())
		other := recoveryPR(43, pr.Head.SHA)
		client.PullRequests[43] = &other
		started := time.Now()
		_, err = h.reconcilePR(ctx, other, jobs, true)
		Expect(err).NotTo(HaveOccurred())
		Expect(time.Since(started)).To(BeNumerically(">=", 30*time.Millisecond))
	})

	It("reads nil statuses safely and reports status errors", func() {
		client.status = func(string) (*github.CombinedStatus, error) { return nil, nil }
		statuses, err := h.phaseStatuses(pr.Head.SHA)
		Expect(err).NotTo(HaveOccurred())
		Expect(statuses).To(BeEmpty())
		client.status = func(string) (*github.CombinedStatus, error) { return nil, errors.New("status unavailable") }
		_, err = h.phaseStatuses(pr.Head.SHA)
		Expect(err).To(MatchError(ContainSubstring("status unavailable")))
	})

	It("uses the shared phase selector", func() {
		completed, next := nextPhase(jobs, map[string]string{"build": github.StatusSuccess})
		Expect(completed).To(Equal(0))
		Expect(next).To(Equal(1))
	})

	It("scans an entire PR list with bounded concurrency", func() {
		var active, maximum atomic.Int32
		client.status = func(string) (*github.CombinedStatus, error) {
			count := active.Add(1)
			defer active.Add(-1)
			for old := maximum.Load(); count > old && !maximum.CompareAndSwap(old, count); old = maximum.Load() {
			}
			time.Sleep(time.Millisecond)
			return &github.CombinedStatus{Statuses: []github.Status{{Context: "build", State: github.StatusSuccess}}}, nil
		}
		var prs []github.PullRequest
		for i := range 205 {
			prs = append(prs, recoveryPR(i+1, fmt.Sprint(i)))
		}
		results := h.scanPRs(ctx, prs, jobs)
		Expect(results).To(HaveLen(205))
		Expect(maximum.Load()).To(BeNumerically("<=", 4))
		for _, result := range results {
			Expect(result.err).NotTo(HaveOccurred())
			Expect(result.state).To(Equal("candidate"))
		}
	})

	It("refreshes configuration once per pass and ignores the webhook cache", func() {
		current, loads := serveRecoveryConfig(h, jobs)
		h.cache.set([]config.Presubmit{presubmit("old", "0")})
		summary, err := h.ReconcileOpenPRs(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(summary.Requested).To(Equal(1))
		Expect(loads.Load()).To(Equal(int32(1)))
		newJobs := append(append([]config.Presubmit{}, jobs...), presubmit("additional", "1"))
		newJobs[len(newJobs)-1].Spec = jobs[0].Spec
		current.Store(newJobs)
		summary, err = h.ReconcileOpenPRs(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(summary.Requested).To(Equal(1))
		Expect(loads.Load()).To(Equal(int32(2)))
		Expect(client.IssueCommentsAdded[1]).To(ContainSubstring("/test additional"))
	})

	It("continues past a failed PR and reports the pass as retryable", func() {
		serveRecoveryConfig(h, jobs)
		other := recoveryPR(43, "bad-head")
		client.prs = append(client.prs, other)
		client.status = func(sha string) (*github.CombinedStatus, error) {
			if sha == "bad-head" {
				return nil, errors.New("unavailable")
			}
			return client.FakeClient.GetCombinedStatus("kubevirt", "kubevirt", sha)
		}
		summary, err := h.ReconcileOpenPRs(ctx)
		Expect(err).To(HaveOccurred())
		Expect(summary.Scanned).To(Equal(2))
		Expect(summary.Errors).To(Equal(1))
		Expect(summary.Requested).To(Equal(1))
	})

	It("records request errors without aborting other candidates", func() {
		serveRecoveryConfig(h, jobs)
		other := recoveryPR(43, pr.Head.SHA)
		client.prs = append(client.prs, other)
		client.PullRequests[43] = &other
		client.post = func(ctx context.Context, number int, body string) error {
			if number == 42 {
				return errors.New("post failed")
			}
			return client.FakeClient.CreateCommentWithContext(ctx, "kubevirt", "kubevirt", number, body)
		}
		summary, err := h.ReconcileOpenPRs(ctx)
		Expect(err).To(HaveOccurred())
		Expect(summary.Errors).To(Equal(1))
		Expect(summary.Requested).To(Equal(1))
	})

	It("counts skipped, blocked and recently requested PRs", func() {
		serveRecoveryConfig(h, jobs)
		draft := recoveryPR(43, "draft-head")
		draft.Draft = true
		blocked := recoveryPR(44, "blocked-head")
		client.CombinedStatuses[blocked.Head.SHA] = &github.CombinedStatus{}
		client.prs = append(client.prs, draft, blocked)
		_, err := h.reconcilePR(ctx, pr, jobs, false)
		Expect(err).NotTo(HaveOccurred())
		summary, err := h.ReconcileOpenPRs(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(summary.Skipped).To(Equal(1))
		Expect(summary.Blocked).To(Equal(1))
		Expect(summary.AlreadyRequested).To(Equal(1))
	})

	It("reports configuration and list failures and supports no phased jobs", func() {
		_, err := h.ReconcileOpenPRs(ctx)
		Expect(err).To(HaveOccurred())
		current, _ := serveRecoveryConfig(h, jobs)
		client.list = func() ([]github.PullRequest, error) { return nil, errors.New("list failed") }
		_, err = h.ReconcileOpenPRs(ctx)
		Expect(err).To(MatchError(ContainSubstring("list failed")))
		current.Store([]config.Presubmit{})
		summary, err := h.ReconcileOpenPRs(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(summary.Scanned).To(BeZero())
	})

	It("does no work after cancellation", func() {
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := h.ReconcileOpenPRs(canceled)
		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
		_, err = h.reconcilePR(canceled, pr, jobs, false)
		Expect(errors.Is(err, context.Canceled)).To(BeTrue())
	})

	It("runs a startup-only scan without waiting for a webhook", func() {
		serveRecoveryConfig(h, jobs)
		h.RunReconciler(ctx, 0)
		Expect(client.IssueCommentsAdded).To(HaveLen(1))
	})

	It("runs periodic scans and stops during the interval on cancellation", func() {
		_, loads := serveRecoveryConfig(h, jobs)
		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		var calls atomic.Int32
		client.list = func() ([]github.PullRequest, error) {
			if calls.Add(1) == 2 {
				cancel()
			}
			return nil, nil
		}
		h.RunReconciler(runCtx, time.Millisecond)
		Expect(loads.Load()).To(Equal(int32(2)))
	})

	It("retries a failed startup-only scan and cancels during backoff", func() {
		serveRecoveryConfig(h, jobs)
		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		calls := 0
		client.list = func() ([]github.PullRequest, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("temporary list failure")
			}
			return nil, nil
		}
		h.RunReconciler(runCtx, 0)
		Expect(calls).To(Equal(2))
		client.list = func() ([]github.PullRequest, error) {
			cancel()
			return nil, errors.New("unavailable")
		}
		h.RunReconciler(runCtx, 0)
	})
})
