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

			It("Should generate Prow jobs for the changed configs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
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
														Image: "modified-image",
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: &event.PullRequest,
					}
				})

				By("Sending the event to the rehearsal server", func() {

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
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())
					eventsHandler.Handle(handlerEvent)
					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(1))
						pjAction := prowc.Actions()[0].GetResource()
						Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
					})
				})

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
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: &event.PullRequest,
					}
				})

				By("Sending the event to the rehearsal server", func() {

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
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					eventsHandler.Handle(handlerEvent)
					Expect(err).ShouldNot(HaveOccurred())
					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

			It("Should not generate Prow jobs if a job was deleted", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit that removes a job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								orgRepo: {
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
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: &event.PullRequest,
					}
				})

				By("Sending the event to the rehearsal server", func() {

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
						true,
						gitClientFactory)
					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())
					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

			It("Should not act on pull request event if always run is set to false", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
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
														Image: "modified-image",
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: &event.PullRequest,
					}
				})

				By("Sending the event to the rehearsal server", func() {

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
						false,
						gitClientFactory)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())
					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

		})

		Context("ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changed configs with ok-to-test label", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
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
														Image: "modified-image",
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Labels: []github.Label{
								{
									Name: "ok-to-test",
								},
							},
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: &event.PullRequest,
					}
				})

				By("Sending the event to the rehearsal server", func() {

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
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {

						Expect(prowc.Actions()).Should(HaveLen(1))
						pjAction := prowc.Actions()[0].GetResource()
						Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
					})
				})

			})

		})

		Context("Unauthorized user", func() {

			It("Should not generate Prow jobs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
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
														Image: "modified-image",
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: orgRepo,
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     repo,
									FullName: orgRepo,
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: &event.PullRequest,
					}
				})

				By("Sending the event to the rehearsal server", func() {

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
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {

						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

		})

	})

	Context("A valid comment event", func() {

		var gitrepo *localgit.LocalGit
		var gitClientFactory git2.ClientFactory
		var sendIssueCommentEventToRehearsalServer func(gh *fakegithub.FakeClient, event *github.IssueCommentEvent) *fake.FakeProwV1

		BeforeEach(func() {
			var err error

			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())

			sendIssueCommentEventToRehearsalServer = func(gh *fakegithub.FakeClient, event *github.IssueCommentEvent) *fake.FakeProwV1 {
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
					true,
					gitClientFactory)

				handlerEvent, err := makeHandlerIssueCommentEvent(event)
				Expect(err).ShouldNot(HaveOccurred())

				eventsHandler.Handle(handlerEvent)

				return prowc
			}

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

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
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
														Image: "modified-image",
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
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

				By("Sending the event to the rehearsal server", func() {

					prowc := sendIssueCommentEventToRehearsalServer(gh, event)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(1))
						pjAction := prowc.Actions()[0].GetResource()
						Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
					})
				})

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

				By("Sending the event to the rehearsal server", func() {

					prowc := sendIssueCommentEventToRehearsalServer(gh, event)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

			It("Should not generate Prow jobs if a job was deleted", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								orgRepo: {
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {testuser},
					}
				})
				event := NewGHIssueCommentEvent(gh, baseref, headref)
				By("Sending the event to the rehearsal server", func() {

					prowc := sendIssueCommentEventToRehearsalServer(gh, event)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

			//TODO - FIX me
			PIt("Should not generate Prow jobs if a job is not permitted", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								orgRepo: {
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						org: {testuser},
					}
				})
				event := NewGHIssueCommentEvent(gh, baseref, headref)
				By("Sending the event to the rehearsal server", func() {

					prowc := sendIssueCommentEventToRehearsalServer(gh, event)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

		})

		Context("ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changed configs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
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
														Image: "modified-image",
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

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

				By("Sending the event to the rehearsal server", func() {

					prowc := sendIssueCommentEventToRehearsalServer(gh, event)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(1))

						pjAction := prowc.Actions()[0].GetResource()
						Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
					})
				})

			})

		})

		Context("Unauthorized user", func() {

			It("Should not generate Prow jobs", func() {

				makeRepoWithEmptyProwConfig(gitrepo)

				baseref := GenerateBaseCommit(gitrepo)

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
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
														Image: "modified-image",
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
					err = gitrepo.AddCommit(org, repo, map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse(org, repo, "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				event := NewGHIssueCommentEvent(gh, baseref, headref)
				By("Sending the event to the rehearsal server", func() {

					prowc := sendIssueCommentEventToRehearsalServer(gh, event)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

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
