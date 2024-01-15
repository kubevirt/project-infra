package main_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"

	"kubevirt.io/project-infra/external-plugins/checklist/plugin/handler"
)

const (
	org      = "kubevirt"
	repo     = "kubevirt"
	baseRef  = "main"
	orgRepo  = org + "/" + repo
	prNumber = 17
)

var _ = Describe("Checklist", func() {
	Context("A valid pull request event", func() {
		Context("PR is opened with empty checklist", func() {
			It("Should label the PR with incomplete checklist label", func() {
				gh := fakegithub.NewFakeClient()

				var event github.PullRequestEvent
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: "testuser",
						},
						PullRequest: github.PullRequest{
							Number: prNumber,
							State:  "open",
							Body:   "- [ ] Design: abc",
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						prNumber: &event.PullRequest,
					}
				})

				By("Sending the event to the checklist plugin server", func() {
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						gh)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					Expect(len(gh.IssueLabelsAdded)).To(Equal(1), "Expected github labels to be added")
					Expect(gh.IssueLabelsAdded[0]).To(Equal(fmt.Sprintf("%s#%d:do-not-merge/incomplete-checklist", orgRepo, prNumber)))
				})

			})

		})

		Context("PR is edited, label exists, checklist is fully done", func() {
			It("Should remove the label from the PR", func() {
				gh := fakegithub.NewFakeClient()
				gh.IssueLabelsExisting = append(gh.IssueLabelsExisting, issueLabels(handler.IncompleteChecklistLabel)...)

				var event github.PullRequestEvent
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionEdited,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: "testuser",
						},
						PullRequest: github.PullRequest{
							Number: prNumber,
							State:  "open",
							Body:   "- [x] Design: abc",
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						prNumber: &event.PullRequest,
					}
				})

				By("Sending the event to the checklist plugin server", func() {
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						gh)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					Expect(len(gh.IssueLabelsRemoved)).To(Equal(1), "Expected github labels to be removed")
					Expect(gh.IssueLabelsRemoved[0]).To(Equal(fmt.Sprintf("%s#%d:do-not-merge/incomplete-checklist", orgRepo, prNumber)))
				})

			})

		})
	})

})

func makeHandlerPullRequestEvent(event *github.PullRequestEvent) (*handler.GitHubEvent, error) {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	handlerEvent := &handler.GitHubEvent{
		Type:    "pull_request",
		GUID:    event.GUID,
		Payload: eventBytes,
	}
	return handlerEvent, nil
}

func issueLabels(labels ...string) []string {
	var ls []string
	for _, label := range labels {
		ls = append(ls, fmt.Sprintf("%s#%d:%s", orgRepo, prNumber, label))
	}
	return ls
}
