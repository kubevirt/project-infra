package main_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/testing"

	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1/fake"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/git/localgit"
	git2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"

	"kubevirt.io/project-infra/external-plugins/test-subset/plugin/handler"
)

const (
	org      = "kubevirt"
	repo     = "kubevirt"
	baseRef  = "main"
	orgRepo  = org + "/" + repo
	prNumber = 17
)

var _ = Describe("Test-subset", func() {
	Context("A valid pull request comment event", func() {
		var gitrepo *localgit.LocalGit
		var gitClientFactory git2.ClientFactory

		BeforeEach(func() {
			var err error
			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				if err := gitClientFactory.Clean(); err != nil {
					logrus.WithError(err).Error("Failed to clean up git client factory")
				}
			}
		})

		Context("a member comments a test-subset command", func() {
			It("Should run the specified prow jobs", func() {
				By("Creating a fake git repo", func() {
					if err := makeRepoWithEmptyProwConfig(gitrepo, org, repo); err != nil {
						logrus.WithError(err).Fatal("Failed to create fake git repo")
					}
				})

				var baseref string
				By("Generating a base commit with jobs", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								orgRepo: {
									{
										JobBase: config.JobBase{
											Name: "job1",
											Spec: &v1.PodSpec{
												Containers: []v1.Container{
													{
														Image: "image1",
													},
												},
											},
										},
									},
									{
										JobBase: config.JobBase{
											Name: "job2",
											Spec: &v1.PodSpec{
												Containers: []v1.Container{
													{
														Image: "image2",
													},
												},
											},
										},
									},
								},
							},
						},
					})
					Expect(err).ShouldNot(HaveOccurred())
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := fakegithub.NewFakeClient()
				testuser := "testuser"

				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						repo: {testuser},
					}
				})

				var event *github.IssueCommentEvent
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = &github.IssueCommentEvent{
						Action: github.IssueCommentActionCreated,
						Comment: github.IssueComment{
							Body: `/test-subset job1 "(label1)"`,
							User: github.User{
								Login: testuser,
							},
						},
						GUID: "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Issue: github.Issue{
							Number: prNumber,
							State:  "open",
							User: github.User{
								Login: testuser,
							},
							PullRequest: &struct{}{},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						prNumber: {
							Number: prNumber,
							State:  "open",
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseRef,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseRef,
								SHA: baseref,
							},
						},
					}
				})

				By("Sending the event to the test-subset plugin server", func() {
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					prowc := &fake.FakeProwV1{
						Fake: &testing.Fake{},
					}
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"jobs-config.yaml",
						"",
						gitClientFactory)

					handlerEvent, err := makeHandlerIssueCommentEvent(event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.SetLocalConfLoad()
					eventsHandler.Handle(handlerEvent)

					Expect(prowc.Actions()).Should(HaveLen(1))
					pjAction := prowc.Actions()[0].GetResource()
					Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
				})
			})
		})
	})
})

func makeRepoWithEmptyProwConfig(lg *localgit.LocalGit, org, repo string) error {
	err := lg.MakeFakeRepo(org, repo)
	if err != nil {
		return err
	}
	prowConfig, err := json.Marshal(&config.ProwConfig{})
	if err != nil {
		return err
	}
	return lg.AddCommit(org, repo, map[string][]byte{
		"prowconfig.yaml": prowConfig,
	})
}

func makeHandlerIssueCommentEvent(event *github.IssueCommentEvent) (*handler.GitHubEvent, error) {
	eventBytes, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	handlerEvent := &handler.GitHubEvent{
		Type:    "issue_comment",
		GUID:    event.GUID,
		Payload: eventBytes,
	}
	return handlerEvent, nil
}
