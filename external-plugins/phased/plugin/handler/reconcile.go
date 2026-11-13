package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/github"
)

const requestGracePeriod = 10 * time.Minute

// ReconcileSummary counts PRs, not individual jobs. Requested PRs may still be
// waiting for Prow to publish their first statuses.
type ReconcileSummary struct {
	Scanned, Skipped, Blocked, Candidates, Requested, AlreadyRequested, Errors int
}

// RunReconciler recovers missed transitions immediately and after each interval.
// Zero disables periodic scans, but startup failures still retry with backoff.
func (h *GitHubEventsHandler) RunReconciler(ctx context.Context, interval time.Duration) {
	backoff := time.Second
	for ctx.Err() == nil {
		summary, err := h.ReconcileOpenPRs(ctx)
		if ctx.Err() != nil {
			return
		}
		h.logger.WithField("summary", fmt.Sprintf("%+v", summary)).Info("Phase reconciliation finished")
		delay := interval
		if err != nil {
			h.logger.WithError(err).Error("Phase reconciliation failed")
			delay = backoff
			backoff = min(2*backoff, time.Minute)
		} else {
			backoff = time.Second
			if interval == 0 {
				return
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// ReconcileOpenPRs loads one fresh configuration snapshot and scans all open
// PRs before making paced requests. GetPullRequests handles GitHub pagination.
func (h *GitHubEventsHandler) ReconcileOpenPRs(ctx context.Context) (ReconcileSummary, error) {
	h.reconcileMu.Lock()
	defer h.reconcileMu.Unlock()
	var summary ReconcileSummary
	if err := ctx.Err(); err != nil {
		return summary, err
	}
	presubmits, err := h.loadPresubmits(github.PullRequest{Base: github.PullRequestBranch{
		Ref: "main", Repo: github.Repo{FullName: "kubevirt/kubevirt"},
	}})
	if err != nil {
		return summary, fmt.Errorf("refreshing presubmits: %w", err)
	}
	h.cache.set(presubmits)
	_, phases := groupJobsByPhase(presubmits)
	if len(phases) < 2 {
		return summary, nil
	}
	prs, err := h.ghClient.GetPullRequests("kubevirt", "kubevirt")
	if err != nil {
		return summary, fmt.Errorf("listing open PRs: %w", err)
	}
	results := h.scanPRs(ctx, prs, presubmits)
	var candidates []github.PullRequest
	for _, result := range results {
		summary.Scanned++
		switch {
		case result.err != nil:
			summary.Errors++
			h.logger.WithError(result.err).WithField("pr", result.pr.Number).Error("Could not inspect phase state")
		case result.state == "skipped":
			summary.Skipped++
		case result.state == "blocked":
			summary.Blocked++
		default:
			candidates = append(candidates, result.pr)
		}
	}
	summary.Candidates = len(candidates)
	h.logger.WithField("summary", fmt.Sprintf("%+v", summary)).Info("Phase reconciliation scan finished; requesting missing phases")
	for _, pr := range candidates {
		requested, requestErr := h.reconcilePR(ctx, pr, presubmits, true)
		switch {
		case requestErr != nil:
			summary.Errors++
			h.logger.WithError(requestErr).WithField("pr", pr.Number).Error("Could not recover phase transition")
		case requested:
			summary.Requested++
		default:
			summary.AlreadyRequested++
		}
		if ctx.Err() != nil {
			return summary, ctx.Err()
		}
	}
	if summary.Errors > 0 {
		return summary, fmt.Errorf("%d PR reconciliations failed", summary.Errors)
	}
	return summary, ctx.Err()
}

type scanResult struct {
	pr    github.PullRequest
	state string
	err   error
}

func (h *GitHubEventsHandler) scanPRs(ctx context.Context, prs []github.PullRequest, presubmits []config.Presubmit) []scanResult {
	results := make([]scanResult, len(prs))
	work := make(chan int)
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			for i := range work {
				pr := prs[i]
				results[i] = scanResult{pr: pr, state: "skipped", err: ctx.Err()}
				if results[i].err != nil || !phasePREligible(pr) {
					continue
				}
				statuses, err := h.phaseStatuses(pr.Head.SHA)
				results[i].err = err
				_, next := nextPhase(presubmits, statuses)
				results[i].state = "candidate"
				if next < 0 {
					results[i].state = "blocked"
				}
			}
		})
	}
	for i := range prs {
		work <- i
	}
	close(work)
	workers.Wait()
	return results
}

func phasePREligible(pr github.PullRequest) bool {
	return !isPRNotMergeable(pr) && pr.Head.SHA != "" &&
		(pr.Base.Ref == "main" || pr.Base.Ref == "master")
}

func (h *GitHubEventsHandler) phaseStatuses(sha string) (map[string]string, error) {
	combined, err := h.ghClient.GetCombinedStatus("kubevirt", "kubevirt", sha)
	if err != nil {
		return nil, fmt.Errorf("reading HEAD statuses: %w", err)
	}
	statuses := make(map[string]string)
	if combined != nil {
		for _, status := range combined.Statuses {
			statuses[status.Context] = status.State
		}
	}
	return statuses, nil
}

func nextPhase(presubmits []config.Presubmit, statuses map[string]string) (int, int) {
	jobs, phases := groupJobsByPhase(presubmits)
	return findNextPhaseToTrigger(jobs, phases, statuses)
}

// reconcilePR shares its striped locks and recent-request ledger with status
// webhooks. Paced scans wait before acquiring a PR lock, then re-read live state.
func (h *GitHubEventsHandler) reconcilePR(ctx context.Context, observed github.PullRequest, presubmits []config.Presubmit, paced bool) (bool, error) {
	if paced {
		if err := h.reconcileLimiter.Wait(ctx); err != nil {
			return false, err
		}
	}
	lock := &h.phaseLocks[uint(observed.Number)%uint(len(h.phaseLocks))]
	lock.Lock()
	defer lock.Unlock()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	pr, err := h.ghClient.GetPullRequest("kubevirt", "kubevirt", observed.Number)
	if err != nil {
		return false, fmt.Errorf("refreshing PR: %w", err)
	}
	if pr == nil || !phasePREligible(*pr) || pr.Head.SHA != observed.Head.SHA {
		return false, nil
	}
	statuses, err := h.phaseStatuses(pr.Head.SHA)
	if err != nil {
		return false, err
	}
	completed, next := nextPhase(presubmits, statuses)
	if next < 0 {
		return false, nil
	}
	jobs, _ := groupJobsByPhase(presubmits)
	missing, err := h.unrequestedJobs(ctx, *pr, jobs[next], next, statuses)
	if err != nil || len(missing) == 0 {
		return false, err
	}
	// Comment lookups can take time; never apply an old decision to a new HEAD.
	latest, err := h.ghClient.GetPullRequest("kubevirt", "kubevirt", pr.Number)
	if err != nil {
		return false, fmt.Errorf("rechecking PR: %w", err)
	}
	if latest == nil || !phasePREligible(*latest) || latest.Head.SHA != pr.Head.SHA || latest.Base.Ref != pr.Base.Ref {
		return false, nil
	}
	statuses, err = h.phaseStatuses(pr.Head.SHA)
	if err != nil {
		return false, err
	}
	if _, currentNext := nextPhase(presubmits, statuses); currentNext != next {
		return false, nil
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	comment := fmt.Sprintf(PhaseIntroFmt, completed, next)
	var requested []config.Presubmit
	for _, job := range missing {
		if _, exists := statuses[jobContext(job)]; exists {
			continue
		}
		requested = append(requested, job)
		comment += "/test " + job.Name + "\n"
	}
	if len(requested) == 0 {
		return false, nil
	}
	for _, job := range requested {
		comment += phaseRequestMarker(pr.Head.SHA, next, job.Name) + "\n"
	}
	if err := h.ghClient.CreateCommentWithContext(ctx, "kubevirt", "kubevirt", pr.Number, comment); err != nil {
		return false, fmt.Errorf("requesting phase %d: %w", next, err)
	}
	h.requestsMu.Lock()
	defer h.requestsMu.Unlock()
	if h.requests == nil {
		h.requests = make(map[string]time.Time)
	}
	for _, job := range requested {
		h.requests[fmt.Sprint(pr.Number)+phaseRequestMarker(pr.Head.SHA, next, job.Name)] = time.Now()
	}
	h.logger.WithFields(map[string]interface{}{"pr": pr.Number, "sha": pr.Head.SHA, "phase": next, "jobs": len(requested)}).Info("Requested missing phase jobs")
	return true, nil
}

func phaseRequestMarker(sha string, phase int, job string) string {
	return fmt.Sprintf("<!-- phased: sha=%s phase=%d job=%s -->", sha, phase, job)
}

func (h *GitHubEventsHandler) unrequestedJobs(ctx context.Context, pr github.PullRequest, jobs []config.Presubmit, phase int, statuses map[string]string) ([]config.Presubmit, error) {
	comments, err := h.ghClient.ListIssueCommentsWithContext(ctx, "kubevirt", "kubevirt", pr.Number)
	if err != nil {
		return nil, fmt.Errorf("reading recent phase requests: %w", err)
	}
	isBot, err := h.ghClient.BotUserCheckerWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("identifying phase requester: %w", err)
	}
	cutoff := time.Now().Add(-requestGracePeriod)
	h.requestsMu.Lock()
	defer h.requestsMu.Unlock()
	for key, posted := range h.requests {
		if posted.Before(cutoff) {
			delete(h.requests, key)
		}
	}
	var missing []config.Presubmit
	for _, job := range jobs {
		if _, exists := statuses[jobContext(job)]; exists {
			continue
		}
		marker := phaseRequestMarker(pr.Head.SHA, phase, job.Name)
		if _, exists := h.requests[fmt.Sprint(pr.Number)+marker]; exists {
			continue
		}
		recent := false
		for _, comment := range comments {
			if !comment.CreatedAt.Before(cutoff) && isBot(comment.User.Login) && strings.Contains(comment.Body, marker) {
				recent = true
				break
			}
		}
		if !recent {
			missing = append(missing, job)
		}
	}
	return missing, nil
}
