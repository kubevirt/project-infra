package main_test

import (
	"encoding/json"
	"fmt"

	kubeVirtLabels "kubevirt.io/project-infra/pkg/github/labels"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/git/localgit"
	git2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
	"sigs.k8s.io/prow/pkg/labels"

	"kubevirt.io/project-infra/external-plugins/phased/plugin/handler"
)

const (
	org      = "kubevirt"
	repo     = "kubevirt"
	baseRef  = "main"
	orgRepo  = org + "/" + repo
	prNumber = 17
)

type TestCase struct {
	AddedLabel         string
	ApproveLabelExists bool
	LGTMLabelExists    bool
	ExpectComment      bool
}

var _ = Describe("Phased", func() {
	Context("A valid pull request event", func() {
		var gitrepo *localgit.LocalGit
		var gitClientFactory git2.ClientFactory
		var baseref string

		BeforeEach(func() {
			var err error
			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())
		})

		BeforeEach(func() {
			baseConfig, err := json.Marshal(&config.Config{
				JobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						orgRepo: {
							{
								AlwaysRun: true,
								JobBase: config.JobBase{
									Name: "job_always_run",
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
									Name: "job_always_run_false",
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

			makeRepoWithEmptyProwConfig(gitrepo, org, repo)

			Expect(err).ShouldNot(HaveOccurred())
			err = gitrepo.AddCommit(org, repo, map[string][]byte{
				"jobs-config.yaml": baseConfig,
			})
			Expect(err).ShouldNot(HaveOccurred())
			baseref, err = gitrepo.RevParse(org, repo, "HEAD")
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				gitClientFactory.Clean()
			}
		})

		DescribeTable("Prow Job Commenting",
			func(tc TestCase) {
				gh := fakegithub.NewFakeClient()
				if tc.ApproveLabelExists {
					gh.IssueLabelsExisting = append(gh.IssueLabelsExisting, issueLabels(labels.Approved)...)
				}
				if tc.LGTMLabelExists {
					gh.IssueLabelsExisting = append(gh.IssueLabelsExisting, issueLabels(labels.LGTM)...)
				}
				var event github.PullRequestEvent
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionLabeled,
						Label:  github.Label{Name: tc.AddedLabel},
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

					gh.PullRequests = map[int]*github.PullRequest{
						prNumber: &event.PullRequest,
					}
				})

				By("Sending the event to the phased plugin server", func() {
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						gh,
						"prowconfig.yaml",
						"jobs-config.yaml",
						"",
						gitClientFactory)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.SetLocalConfLoad()
					eventsHandler.Handle(handlerEvent)

					if tc.ExpectComment {
						Expect(len(gh.IssueCommentsAdded)).To(Equal(1), "Expected github comment to be added")
						Expect(gh.IssueCommentsAdded[0]).To(Equal(
							fmt.Sprintf("%s#%d:%s/test job_always_run_false\n", orgRepo, prNumber,
								handler.Intro)))
					} else {
						Expect(len(gh.IssueCommentsAdded)).To(Equal(0), "Expect no github comment to be added")
					}
				})
			},
			Entry("LGTM is added, Approve exists",
				TestCase{
					AddedLabel:         labels.LGTM,
					ApproveLabelExists: true,
					ExpectComment:      true}),
			Entry("LGTM is added, Approve doesnt exist",
				TestCase{
					AddedLabel:         labels.LGTM,
					ApproveLabelExists: false,
					ExpectComment:      false}),
			Entry("Approve is added, LGTM exists",
				TestCase{
					AddedLabel:      labels.Approved,
					LGTMLabelExists: true,
					ExpectComment:   true}),
			Entry("Approve is added, LGTM doesnt exist",
				TestCase{
					AddedLabel:      labels.Approved,
					LGTMLabelExists: false,
					ExpectComment:   false}),
			Entry("Skip Review is added, LGTM and Approve dont exist",
				TestCase{
					AddedLabel:         kubeVirtLabels.SkipReview,
					ApproveLabelExists: false,
					LGTMLabelExists:    false,
					ExpectComment:      true}),
		)

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
