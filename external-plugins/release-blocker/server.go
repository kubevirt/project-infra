package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/test-infra/prow/repoowners"
)

const pluginName = "release-blocker"
const baseLabel = "release-blocker"

var releaseBlockRe = regexp.MustCompile(`(?m)^(?:/releaseblock|/release-block|/release-blocker|/releaseblocker)\s+(.+)$`)
var releaseBlockCancelRe = regexp.MustCompile(`(?m)^(?:/releaseblock\s+cancel|/release-block\s+cancel|/release-blocker\s+cancel|releaseblocker\s+cancel)\s+(.+)$`)

type issueEvent struct {
	github.IssueEvent `json:",inline"`
	Sender            github.User `json:"sender"`
}

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	CreateComment(org, repo string, number int, comment string) error
	IsMember(org, user string) (bool, error)
}

type prowOwnersClient interface {
	LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error)
}

// HelpProvider construct the pluginhelp.PluginHelp for this plugin.
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The release-blocker plugin is used to signal an issue or PR must be resolved before the next release is made.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/release-blocker [branch]",
		Description: "Mark a PR or issue as a release blocker.",
		Featured:    true,
		WhoCanUse:   "Project members",
		Examples:    []string{"/release-blocker release-3.9", "/release-blocker release-1.15"},
	})
	return pluginHelp, nil
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	tokenGenerator func() []byte
	botName        string

	// Used for unit testing
	push         func(newBranch string) error
	ghc          githubClient
	log          *logrus.Entry
	ownersClient prowOwnersClient

	branchExists func(org string, repo string, targetBranch string) (bool, error)
}

// ServeHTTP validates an incoming webhook and puts it into the event channel.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}
	fmt.Fprint(w, "Event received. Have a nice day.")

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

func hasLabel(ghc githubClient, label string, org string, repo string, num int) (bool, error) {
	labels, err := ghc.GetIssueLabels(org, repo, num)
	if err != nil {
		return false, fmt.Errorf("failed to get the labels on %s/%s#%d: %v", org, repo, num, err)
	}

	hasLabel := github.HasLabel(label, labels)

	return hasLabel, nil
}

func (s *Server) canLabel(org string, repo string, base string, commentAuthor string) (bool, error) {
	// only members can add blocking label.
	// Leave this in as a safety precaution in the event
	// that there's a vulnerability in the owners logic.
	ok, err := s.ghc.IsMember(org, commentAuthor)
	if err != nil {
		return false, err
	} else if !ok {
		return false, nil
	}

	owners, err := s.ownersClient.LoadRepoOwners(org, repo, base)
	if err != nil {
		return false, err
	}

	ok = owners.TopLevelApprovers().Has(commentAuthor)

	return ok, nil
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := logrus.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	switch eventType {
	case "issue_comment":
		var ic github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ic); err != nil {
			return err
		}
		go func() {
			if err := s.handleIssueComment(l, ic); err != nil {
				s.log.WithError(err).WithFields(l.Data).Info("release-blocker issue comment failed.")
			}
		}()
	// https://developer.github.com/webhooks/event-payloads/#issues
	case "issues":
		var issue issueEvent
		if err := json.Unmarshal(payload, &issue); err != nil {
			return err
		}

		go func() {
			if err := s.handleIssue(l, issue); err != nil {
				s.log.WithError(err).WithFields(l.Data).Info("release-blocker issue comment failed.")
			}
		}()
	//https://developer.github.com/webhooks/event-payloads/#pull_request
	case "pull_request":
		var pr github.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		go func() {
			if err := s.handlePR(l, pr); err != nil {
				s.log.WithError(err).WithFields(l.Data).Info("release-block failed.")
			}
		}()
	default:
		logrus.WithFields(l.Data).Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func branchExistsFunc(org string, repo string, targetBranch string) (bool, error) {

	resp, err := http.Head(fmt.Sprintf("https://github.com/%s/%s/tree/%s", org, repo, targetBranch))
	if err != nil {
		return false, err
	}

	if resp.StatusCode == 200 {
		return true, nil
	} else if resp.StatusCode == 404 {
		return false, nil
	}

	return false, fmt.Errorf("Failed to detect branch %s, got http response code %d", targetBranch, resp.StatusCode)
}

func (s *Server) handleLabel(targetBranch string, org string, repo string, num int, add bool) (string, error) {

	label := fmt.Sprintf("%s/%s", baseLabel, targetBranch)

	hasLabel, err := hasLabel(s.ghc, label, org, repo, num)
	if err != nil {
		return "", err
	}

	if !hasLabel && add {
		exists, err := s.branchExists(org, repo, targetBranch)
		if err != nil {
			return "", err
		}

		if !exists {
			return fmt.Sprintf("Unable to place blocker label for release branch [%s] because branch does not exist.", targetBranch), nil
		}

		err = s.ghc.AddLabel(org, repo, num, label)
		if err != nil {
			return "", err
		}
	} else if hasLabel && !add {
		err := s.ghc.RemoveLabel(org, repo, num, label)
		if err != nil {
			return "", err
		}
	}

	return "", nil
}

func (s *Server) handleManualLabelling(l *logrus.Entry, action string, org string, repo string, labelName string, num int, user string) error {
	if action != "labeled" && action != "unlabeled" {
		// not a label action
		return nil
	} else if !strings.Contains(labelName, baseLabel) {
		// not a blocker label
		return nil
	}

	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   num,
	})

	if user == s.botName {
		// it's just us, ignore
		return nil
	}

	s.log.WithFields(l.Data).
		WithField("requestor", user).
		Debug("release-blocker label manually modified.")

	resp := ""

	// else, reject anyone trying to set/unset these labels manually
	if action == "unlabeled" {
		resp = fmt.Sprintf("Re-adding label %s, Release blocker labels are not permitted to be removed manually. Please use `/release-blocker cancel <branch name>` comment", labelName)
		err := s.ghc.AddLabel(org, repo, num, labelName)
		if err != nil {
			return err
		}
	} else if action == "labeled" {
		resp = fmt.Sprintf("Removing label %s, Release blocker labels are not permitted to be set manually. Please use `/release-blocker <branch name>` comment", labelName)
		err := s.ghc.RemoveLabel(org, repo, num, labelName)
		if err != nil {
			return err
		}
	}

	s.log.WithFields(l.Data).Info(resp)
	return s.ghc.CreateComment(org, repo, num, resp)

}

func (s *Server) handlePR(l *logrus.Entry, pr github.PullRequestEvent) error {

	action := string(pr.Action)
	org := pr.Repo.Owner.Login
	repo := pr.Repo.Name
	num := pr.Number
	user := pr.Sender.Login
	labelName := pr.Label.Name

	return s.handleManualLabelling(l, action, org, repo, labelName, num, user)

}

func (s *Server) handleIssue(l *logrus.Entry, issue issueEvent) error {

	action := string(issue.Action)
	org := issue.Repo.Owner.Login
	repo := issue.Repo.Name
	num := issue.Issue.Number
	user := issue.Sender.Login
	labelName := issue.Label.Name

	return s.handleManualLabelling(l, action, org, repo, labelName, num, user)
}

func (s *Server) handleIssueComment(l *logrus.Entry, ic github.IssueCommentEvent) error {
	// we're only interested in new comments
	if ic.Action != github.IssueCommentActionCreated {
		return nil
	}

	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	num := ic.Issue.Number
	commentAuthor := ic.Comment.User.Login

	needsLabel := true
	targetBranch := ""

	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   num,
	})

	cancelMatches := releaseBlockCancelRe.FindAllStringSubmatch(ic.Comment.Body, -1)
	matches := releaseBlockRe.FindAllStringSubmatch(ic.Comment.Body, -1)

	if len(cancelMatches) == 1 && len(cancelMatches[0]) == 2 {
		needsLabel = false
		targetBranch = strings.TrimSpace(cancelMatches[0][1])
	} else if len(matches) == 1 && len(matches[0]) == 2 {
		needsLabel = true
		targetBranch = strings.TrimSpace(matches[0][1])
	} else {
		// no matches
		return nil
	}

	// validate the user is allowed to block or unblock
	// Since this needs to work with issues and PRs, we default to the
	// owners in the main branch of the repo
	ok, err := s.canLabel(org, repo, "main", commentAuthor)
	if err != nil {
		return err
	}

	// not authorized.
	if !ok {
		resp := fmt.Sprintf("only [%s](https://github.com/orgs/%s/people) org members may request release block label", org, org)
		s.log.WithFields(l.Data).Info(resp)
		return s.ghc.CreateComment(org, repo, num, plugins.FormatICResponse(ic.Comment, resp))
	}

	s.log.WithFields(l.Data).
		WithField("requestor", ic.Comment.User.Login).
		WithField("target_branch", targetBranch).
		Debug("release-blocker request.")

	resp, err := s.handleLabel(targetBranch, org, repo, num, needsLabel)
	if err != nil {
		s.log.WithFields(l.Data).WithError(err)
		return err
	} else if resp != "" {
		s.log.WithFields(l.Data).Info(resp)
		return s.ghc.CreateComment(org, repo, num, plugins.FormatICResponse(ic.Comment, resp))
	}

	return nil
}
