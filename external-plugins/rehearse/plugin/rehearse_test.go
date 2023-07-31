package main_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/testing"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/client/clientset/versioned/typed/prowjobs/v1/fake"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/localgit"
	git2 "k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"

	"kubevirt.io/project-infra/external-plugins/rehearse/plugin/handler"
)

const org, repo = "foo", "bar"
const orgRepo = org + "/" + repo

var _ = Describe("Rehearse", func() {

	Context("A valid pull request event", func() {

		var gitrepo *localgit.LocalGit
		var gitClientFactory git2.ClientFactory

		sendPREventToRehearsalServer := func(gh *fakegithub.FakeClient, event *github.PullRequestEvent) *fake.FakeProwV1 {
			By("Sending the event to the rehearsal server")

			prowc := &fake.FakeProwV1{
				Fake: &testing.Fake{},
			}
			fakelog := logrus.New()
			eventsHandler := handler.NewGitHubEventsHandler(
				fakelog,
				prowc.ProwJobs("test-ns"),
				gh,
				"prowconfig.yaml",
				"",
				gitClientFactory)

			handlerEvent, err := makeHandlerPullRequestEvent(event)
			Expect(err).ShouldNot(HaveOccurred())
			eventsHandler.Handle(handlerEvent)
			return prowc

		}

		expectNoJobsCreated := func(gh *fakegithub.FakeClient, event *github.PullRequestEvent) {
			prowc := sendPREventToRehearsalServer(gh, event)

			By("Inspecting the response and the actions on the client")
			Expect(prowc.Actions()).Should(HaveLen(0))
		}

		expectJobsCreated := func(gh *fakegithub.FakeClient, event *github.PullRequestEvent) {
			prowc := sendPREventToRehearsalServer(gh, event)

			By("Inspecting the response and the actions on the client")
			Expect(prowc.Actions()).Should(HaveLen(1))
			pjAction := prowc.Actions()[0].GetResource()
			Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
		}

		BeforeEach(func() {
			var err error

			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				gitClientFactory.Clean()
			}
		})

		Context("with Periodics", func() {
			It("Should generate Prow jobs for changed config", func() {
				makeRepoWithEmptyProwConfig(gitrepo)

				By("Generating a base commit with a jobs")
				baseRef := GenerateConfigCommit(gitrepo,
					NewConfigWithPeriodics(BaseExistingPeriodicJob(), BaseModifiedJPeriodicob()),
				)

				By("Generating a head commit that modifies a job")
				headRef := GenerateConfigCommit(gitrepo,
					NewConfigWithPeriodics(BaseExistingPeriodicJob(), ModifiedJPeriodicob()),
				)

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHPullRequestEvent(gh, baseRef, headRef)
				expectJobsCreated(gh, event)
			})

			It("Should not generate Prow jobs for not related change", func() {
				makeRepoWithEmptyProwConfig(gitrepo)

				By("Generating a base commit with a jobs")
				baseRef := GenerateConfigCommit(gitrepo,
					NewConfigWithPeriodics(BaseExistingPeriodicJob(), BaseModifiedJPeriodicob()),
				)

				var headRef string
				By("Generating a head commit with an unrelated modified file", func() {
					err := gitrepo.AddCommit(org, repo, map[string][]byte{
						"some-file": []byte(""),
					})
					Expect(err).ShouldNot(HaveOccurred())
					headRef, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHPullRequestEvent(gh, baseRef, headRef)
				expectNoJobsCreated(gh, event)
			})

			It("Should not generate Prow jobs for removed job", func() {
				makeRepoWithEmptyProwConfig(gitrepo)

				By("Generating a base commit with a jobs")
				baseRef := GenerateConfigCommit(gitrepo,
					NewConfigWithPeriodics(BaseExistingPeriodicJob(), BaseModifiedJPeriodicob()),
				)

				By("Generating a head commit that removes job")
				headRef := GenerateConfigCommit(gitrepo,
					NewConfigWithPeriodics(BaseExistingPeriodicJob()),
				)

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHPullRequestEvent(gh, baseRef, headRef)
				expectNoJobsCreated(gh, event)
			})
		})

		Context("User in org", func() {

			It("Should generate Prow jobs for the changed configs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(ModifiedJob(), BaseExistingJob()),
				)

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHPullRequestEvent(gh, baseref, headref)
				expectJobsCreated(gh, event)

			})

			It("Should not generate Prow jobs if there are no changes", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with an unrelated modified file", func() {
					err := gitrepo.AddCommit(org, repo, map[string][]byte{
						"some-file": []byte(""),
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHPullRequestEvent(gh, baseref, headref)
				expectNoJobsCreated(gh, event)

			})

			It("Should not generate Prow jobs if a job was deleted", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit that removes a job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(BaseExistingJob()),
				)

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHPullRequestEvent(gh, baseref, headref)

				expectNoJobsCreated(gh, event)

			})

		})

		Context("ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changed configs with ok-to-test label", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(BaseExistingJob(), ModifiedJob()),
				)

				gh := &fakegithub.FakeClient{}

				event := NewGHPullRequestEvent(gh, baseref, headref, func(pr *github.PullRequest) {
					pr.Labels = append(pr.Labels, github.Label{Name: "ok-to-test"})
				})

				expectJobsCreated(gh, event)

			})

		})

		Context("Unauthorized user", func() {

			It("Should not generate Prow jobs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(BaseExistingJob(), ModifiedJob()),
				)

				gh := &fakegithub.FakeClient{}

				event := NewGHPullRequestEvent(gh, baseref, headref)

				expectNoJobsCreated(gh, event)

			})

		})

	})

	Context("A valid comment event", func() {

		var gitrepo *localgit.LocalGit
		var gitClientFactory git2.ClientFactory

		sendIssueCommentEventToRehearsalServer := func(gh *fakegithub.FakeClient, event *github.IssueCommentEvent) *fake.FakeProwV1 {
			prowc := &fake.FakeProwV1{
				Fake: &testing.Fake{},
			}
			fakelog := logrus.New()
			eventsHandler := handler.NewGitHubEventsHandler(
				fakelog,
				prowc.ProwJobs("test-ns"),
				gh,
				"prowconfig.yaml",
				"",
				gitClientFactory)

			handlerEvent, err := makeHandlerIssueCommentEvent(event)
			Expect(err).ShouldNot(HaveOccurred())

			eventsHandler.Handle(handlerEvent)

			return prowc
		}

		expectNoJobsCreated := func(gh *fakegithub.FakeClient, event *github.IssueCommentEvent) {
			prowc := sendIssueCommentEventToRehearsalServer(gh, event)

			By("Inspecting the response and the actions on the client")
			Expect(prowc.Actions()).Should(HaveLen(0))

		}

		expectJobToBeCreated := func(gh *fakegithub.FakeClient, event *github.IssueCommentEvent) {
			prowc := sendIssueCommentEventToRehearsalServer(gh, event)

			By("Inspecting the response and the actions on the client", func() {
				Expect(prowc.Actions()).Should(HaveLen(1))
				pjAction := prowc.Actions()[0].GetResource()
				Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
			})
		}

		BeforeEach(func() {
			var err error

			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())

		})

		AfterEach(func() {
			if gitClientFactory != nil {
				gitClientFactory.Clean()
			}
		})

		Context("User in org", func() {
			//
			It("Should generate Prow jobs for the changed configs", func() {
				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(BaseExistingJob(), ModifiedJob()),
				)

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHIssueCommentEvent(gh, baseref, headref)

				expectJobToBeCreated(gh, event)

			})

			It("Should not generate Prow jobs if there are no changes", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with an unrelated modified file", func() {
					err := gitrepo.AddCommit(org, repo, map[string][]byte{
						"some-file": []byte(""),
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				event := NewGHIssueCommentEvent(gh, baseref, headref)

				expectNoJobsCreated(gh, event)

			})

			It("Should not generate Prow jobs if a job was deleted", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(BaseExistingJob()),
				)
				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {testuser},
					}
				})
				event := NewGHIssueCommentEvent(gh, baseref, headref)
				expectNoJobsCreated(gh, event)

			})

			It("Should not generate Prow jobs if a job is not permitted", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")

				headref := GenerateConfigCommit(gitrepo,
					NewConfig(ModifiedJob(func(jb *config.JobBase) {
						if jb.Annotations == nil {
							jb.Annotations = make(map[string]string)
						}
						jb.Annotations["rehearsal.restricted"] = "true"
					}),
						BaseExistingJob(),
					),
				)

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {testuser},
					}
				})
				event := NewGHIssueCommentEvent(gh, baseref, headref)
				expectNoJobsCreated(gh, event)

			})

		})

		Context("ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changed configs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(BaseExistingJob(), ModifiedJob()),
				)

				gh := &fakegithub.FakeClient{}
				event := NewGHIssueCommentEvent(gh, baseref, headref,
					func(pr *github.PullRequest) {
						if pr.Labels == nil {
							pr.Labels = []github.Label{}
						}
						pr.Labels = append(pr.Labels, github.Label{
							Name: "ok-to-test",
						})
					})

				expectJobToBeCreated(gh, event)
			})

		})

		Context("Unauthorized user", func() {

			It("Should not generate Prow jobs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				By("Generating a head commit with a modified job")
				headref := GenerateConfigCommit(gitrepo,
					NewConfig(BaseExistingJob(), ModifiedJob()),
				)

				gh := &fakegithub.FakeClient{}
				event := NewGHIssueCommentEvent(gh, baseref, headref)
				expectNoJobsCreated(gh, event)

			})

		})

	})

})

func makeRepoWithEmptyProwConfig(lg *localgit.LocalGit) error {
	By("Creating a fake git repo")
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

type pullRequestOption func(*github.PullRequest)

func NewGHIssueCommentEvent(gh *fakegithub.FakeClient, baseRef, headRef string, pullRequestOptions ...pullRequestOption) *github.IssueCommentEvent {
	testuser := "testuser"
	event := &github.IssueCommentEvent{
		Action: github.IssueCommentActionCreated,
		Comment: github.IssueComment{
			Body: "/rehearse",
			User: github.User{
				Login: testuser,
			},
		},
		GUID: "guid",
		Repo: github.Repo{
			FullName: orgRepo,
		},
		Issue: github.Issue{
			Number: 17,
			State:  "open",
			User: github.User{
				Login: testuser,
			},
			PullRequest: &struct{}{},
		},
	}

	pr := &github.PullRequest{
		Number: 17,
		Base: github.PullRequestBranch{
			Repo: github.Repo{
				Name:     repo,
				FullName: orgRepo,
			},
			Ref: baseRef,
			SHA: baseRef,
		},
		Head: github.PullRequestBranch{
			Repo: github.Repo{
				Name:     repo,
				FullName: orgRepo,
			},
			Ref: headRef,
			SHA: headRef,
		},
	}
	for _, f := range pullRequestOptions {
		f(pr)
	}

	gh.PullRequests = map[int]*github.PullRequest{
		17: pr,
	}
	return event
}

func NewGHPullRequestEvent(gh *fakegithub.FakeClient, baseRef, headRef string, pullRequestOption ...pullRequestOption) *github.PullRequestEvent {
	By("Generating a fake pull request event and registering it to the github client")
	pr := github.PullRequest{
		Number: 17,
		Base: github.PullRequestBranch{
			Repo: github.Repo{
				Name:     repo,
				FullName: orgRepo,
			},
			Ref: baseRef,
			SHA: baseRef,
		},
		Head: github.PullRequestBranch{
			Repo: github.Repo{
				Name:     repo,
				FullName: orgRepo,
			},
			Ref: headRef,
			SHA: headRef,
		},
	}
	for _, f := range pullRequestOption {
		f(&pr)
	}

	event := github.PullRequestEvent{
		Action: github.PullRequestActionOpened,
		GUID:   "guid",
		Repo: github.Repo{
			FullName: orgRepo,
		},
		Sender: github.User{
			Login: "testuser",
		},
		PullRequest: pr,
	}

	gh.PullRequests = map[int]*github.PullRequest{
		17: &pr,
	}

	return &event
}

func GenerateBaseCommit(gitrepo *localgit.LocalGit) string {
	By("Generating a base commit with a jobs")
	baseConfig, err := json.Marshal(&config.Config{
		JobConfig: config.JobConfig{
			PresubmitsStatic: map[string][]config.Presubmit{
				orgRepo: {
					{
						JobBase: config.JobBase{
							Name: "modified-job",
							Annotations: map[string]string{
								"rehearsal.allowed": "true",
							},
							Spec: &v1.PodSpec{
								Containers: []v1.Container{
									{
										Image: "some-image",
									},
								},
							},
						},
					},
					{
						JobBase: config.JobBase{
							Name: "existing-job",
							Annotations: map[string]string{
								"rehearsal.allowed": "true",
							},
							Spec: &v1.PodSpec{
								Containers: []v1.Container{
									{
										Image: "other-image",
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
	baseref, err := gitrepo.RevParse(org, repo, "HEAD")
	Expect(err).ShouldNot(HaveOccurred())
	return baseref
}

func GenerateConfigCommit(gitrepo *localgit.LocalGit, config *config.Config) string {
	configBytes, err := json.Marshal(config)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	err = gitrepo.AddCommit(org, repo, map[string][]byte{
		"jobs-config.yaml": configBytes,
	})
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	ref, err := gitrepo.RevParse(org, repo, "HEAD")
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	return ref
}

func NewConfigWithPeriodics(periodics ...config.Periodic) *config.Config {
	return &config.Config{
		JobConfig: config.JobConfig{
			Periodics: periodics,
		},
	}
}

func NewConfig(presubmits ...config.Presubmit) *config.Config {
	config := config.Config{
		JobConfig: config.JobConfig{
			PresubmitsStatic: map[string][]config.Presubmit{
				orgRepo: presubmits,
			},
		},
	}
	return &config
}

var existingJobBase = config.JobBase{
	Name: "existing-job",
	Annotations: map[string]string{
		"rehearsal.allowed": "true",
	},
	Spec: &v1.PodSpec{
		Containers: []v1.Container{
			{
				Image: "other-image",
			},
		},
	},
}

func BaseExistingPeriodicJob() config.Periodic {
	return config.Periodic{
		Cron:    "5 * * * *",
		JobBase: existingJobBase,
	}
}

func BaseExistingJob() config.Presubmit {
	return config.Presubmit{
		JobBase: existingJobBase,
	}

}

var modifiedJobBase = config.JobBase{
	Name: "modified-job",
	Annotations: map[string]string{
		"rehearsal.allowed": "true",
	},
	Spec: &v1.PodSpec{
		Containers: []v1.Container{
			{
				Image: "some-image",
			},
		},
	},
}

func BaseModifiedJPeriodicob() config.Periodic {
	return config.Periodic{
		Cron:    "5 * * * *",
		JobBase: modifiedJobBase,
	}
}

func ModifiedJPeriodicob() config.Periodic {
	job := BaseModifiedJPeriodicob()
	job.Spec.Containers[0].Image = "modified"
	return job
}

func BaseModifiedJob() config.Presubmit {
	return config.Presubmit{
		JobBase: modifiedJobBase,
	}
}

func ModifiedJob(options ...func(*config.JobBase)) config.Presubmit {
	job := BaseModifiedJob()
	job.Spec.Containers[0].Image = "modified"
	for _, f := range options {
		f(&job.JobBase)
	}
	return job
}
