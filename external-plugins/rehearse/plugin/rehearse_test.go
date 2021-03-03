package main_test

import (
	"encoding/json"

	"kubevirt.io/project-infra/external-plugins/rehearse/plugin/handler"

	"k8s.io/client-go/testing"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/test-infra/prow/config"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/client/clientset/versioned/typed/prowjobs/v1/fake"
	"k8s.io/test-infra/prow/git/localgit"
	git2 "k8s.io/test-infra/prow/git/v2"
)

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

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
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
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
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

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with an unrelated modified file", func() {
					err := gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"some-file": []byte(""),
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
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
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
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

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit that removes a job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
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
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
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

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.PullRequestEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.PullRequestEvent{
						Action: github.PullRequestActionOpened,
						GUID:   "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
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
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"",
						false,
						gitClientFactory)

					handlerEvent, err := makeHandlerPullRequestEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())
					go eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

		})

		Context("ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changes configs with ok-to-test label", func() {

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
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
							FullName: "foo/bar",
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
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
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
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
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

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
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
							FullName: "foo/bar",
						},
						Sender: github.User{
							Login: testuser,
						},
						PullRequest: github.PullRequest{
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
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
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
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
				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.IssueCommentEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {

					pr := &github.PullRequest{
						Number: 17,
						Base: github.PullRequestBranch{
							Repo: github.Repo{
								Name:     "bar",
								FullName: "foo/bar",
							},
							Ref: baseref,
							SHA: baseref,
						},
						Head: github.PullRequestBranch{
							Repo: github.Repo{
								Name:     "bar",
								FullName: "foo/bar",
							},
							Ref: headref,
							SHA: headref,
						},
					}

					event = github.IssueCommentEvent{
						Action: github.IssueCommentActionCreated,
						Comment: github.IssueComment{
							Body: "/rehearse",
							User: github.User{
								Login: testuser,
							},
						},
						GUID: "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
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

					gh.PullRequests = map[int]*github.PullRequest{
						17: pr,
					}
				})

				By("Sending the event to the rehearsal server", func() {

					prowc := &fake.FakeProwV1{
						Fake: &testing.Fake{},
					}
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"",
						true,
						gitClientFactory)
					handlerEvent, err := makeHandlerIssueCommentEvent(&event)
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

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with an unrelated modified file", func() {
					err := gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"some-file": []byte(""),
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.IssueCommentEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {
							testuser,
						},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.IssueCommentEvent{
						Action: github.IssueCommentActionCreated,
						Comment: github.IssueComment{
							Body: "/rehearse",
							User: github.User{
								Login: testuser,
							},
						},
						GUID: "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Issue: github.Issue{
							Number: 17,
							User: github.User{
								Login: testuser,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: {
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}
				})

				By("Sending the event to the rehearsal server", func() {

					prowc := &fake.FakeProwV1{
						Fake: &testing.Fake{},
					}
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"",
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerIssueCommentEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

			It("Should not generate Prow jobs if a job was deleted", func() {

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.IssueCommentEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {testuser},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.IssueCommentEvent{
						Action: github.IssueCommentActionCreated,
						Comment: github.IssueComment{
							Body: "/rehearse",
							User: github.User{
								Login: testuser,
							},
						},
						GUID: "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Issue: github.Issue{
							Number: 17,
							User: github.User{
								Login: testuser,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: {
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}
				})

				By("Sending the event to the rehearsal server", func() {

					prowc := &fake.FakeProwV1{
						Fake: &testing.Fake{},
					}
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"",
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerIssueCommentEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

			It("Should not generate Prow jobs if a job is not permitted", func() {

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
									{
										JobBase: config.JobBase{
											Name: "modified-job",
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.IssueCommentEvent

				testuser := "testuser"
				By("Registering a user to the fake github client", func() {
					gh.OrgMembers = map[string][]string{
						"foo": {testuser},
					}
				})
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.IssueCommentEvent{
						Action: github.IssueCommentActionCreated,
						Comment: github.IssueComment{
							Body: "/rehearse",
							User: github.User{
								Login: testuser,
							},
						},
						GUID: "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Issue: github.Issue{
							Number: 17,
							User: github.User{
								Login: testuser,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: {
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}
				})

				By("Sending the event to the rehearsal server", func() {

					prowc := &fake.FakeProwV1{
						Fake: &testing.Fake{},
					}
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"",
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerIssueCommentEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

		})

		Context("ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changes configs", func() {

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.IssueCommentEvent

				testuser := "testuser"
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.IssueCommentEvent{
						Action: github.IssueCommentActionCreated,
						Comment: github.IssueComment{
							Body: "/rehearse",
							User: github.User{
								Login: testuser,
							},
						},
						GUID: "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
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

					gh.PullRequests = map[int]*github.PullRequest{
						17: {
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Labels: []github.Label{
								{
									Name: "ok-to-test",
								},
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}
				})

				By("Sending the event to the rehearsal server", func() {

					prowc := &fake.FakeProwV1{
						Fake: &testing.Fake{},
					}
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"",
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerIssueCommentEvent(&event)
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

				By("Creating a fake git repo", func() {
					makeRepoWithEmptyProwConfig(gitrepo, "foo", "bar")
				})

				var baseref string
				By("Generating a base commit with a job", func() {
					baseConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": baseConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				var headref string
				By("Generating a head commit with a modified job", func() {
					headConfig, err := json.Marshal(&config.Config{
						JobConfig: config.JobConfig{
							PresubmitsStatic: map[string][]config.Presubmit{
								"foo/bar": {
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
					err = gitrepo.AddCommit("foo", "bar", map[string][]byte{
						"jobs-config.yaml": headConfig,
					})
					Expect(err).ShouldNot(HaveOccurred())
					headref, err = gitrepo.RevParse("foo", "bar", "HEAD")
					Expect(err).ShouldNot(HaveOccurred())
				})

				gh := &fakegithub.FakeClient{}
				var event github.IssueCommentEvent

				testuser := "testuser"
				By("Generating a fake pull request event and registering it to the github client", func() {
					event = github.IssueCommentEvent{
						Action: github.IssueCommentActionCreated,
						Comment: github.IssueComment{
							Body: "/rehearse",
							User: github.User{
								Login: testuser,
							},
						},
						GUID: "guid",
						Repo: github.Repo{
							FullName: "foo/bar",
						},
						Issue: github.Issue{
							Number: 17,
							User: github.User{
								Login: testuser,
							},
						},
					}

					gh.PullRequests = map[int]*github.PullRequest{
						17: {
							Number: 17,
							Base: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: baseref,
								SHA: baseref,
							},
							Head: github.PullRequestBranch{
								Repo: github.Repo{
									Name:     "bar",
									FullName: "foo/bar",
								},
								Ref: headref,
								SHA: headref,
							},
						},
					}
				})

				By("Sending the event to the rehearsal server", func() {

					prowc := &fake.FakeProwV1{
						Fake: &testing.Fake{},
					}
					fakelog := logrus.New()
					eventsChan := make(chan *handler.GitHubEvent)
					eventsHandler := handler.NewGitHubEventsHandler(
						eventsChan,
						fakelog,
						prowc.ProwJobs("test-ns"),
						gh,
						"prowconfig.yaml",
						"",
						true,
						gitClientFactory)

					handlerEvent, err := makeHandlerIssueCommentEvent(&event)
					Expect(err).ShouldNot(HaveOccurred())

					eventsHandler.Handle(handlerEvent)

					By("Inspecting the response and the actions on the client", func() {
						Expect(prowc.Actions()).Should(HaveLen(0))
					})
				})

			})

		})

	})

})

func makeRepoWithEmptyProwConfig(lg *localgit.LocalGit, repo, org string) error {
	err := lg.MakeFakeRepo(repo, org)
	if err != nil {
		return err
	}
	prowConfig, err := json.Marshal(&config.ProwConfig{})
	if err != nil {
		return err
	}
	return lg.AddCommit("foo", "bar", map[string][]byte{
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
