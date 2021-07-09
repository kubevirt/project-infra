package main_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
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

const testUserName = "testuser"

var _ = Describe("Rehearse", func() {

	var gitrepo *localgit.LocalGit
	var gitClientFactory git2.ClientFactory
	var baseref string

	var gh *fakegithub.FakeClient

	var prowc *fake.FakeProwV1
	var fakelog *logrus.Logger

	var eventsChan chan *handler.GitHubEvent
	var eventsHandler *handler.GitHubEventsHandler

	BeforeEach(func() {
		var err error

		gitrepo, gitClientFactory, err = localgit.NewV2()
		Expect(err).ShouldNot(HaveOccurred())

		Expect(makeFakeGitRepository(gitrepo, "foo", "bar")).Should(Succeed())

		gh = &fakegithub.FakeClient{}

		prowc = &fake.FakeProwV1{
			Fake: &testing.Fake{},
		}
		fakelog = logrus.New()
		eventsChan = make(chan *handler.GitHubEvent)
		eventsHandler = handler.NewGitHubEventsHandler(
			eventsChan,
			fakelog,
			prowc.ProwJobs("test-ns"),
			gh,
			"prowconfig.yaml",
			"",
			true,
			gitClientFactory)

		By("Generating a base commit with two presubmits", func() {
			expectAddCommitForJobsConfigToSucceed(gitrepo,
				expectGenerateMarshalledConfigWithJobsToSucceed([]config.Presubmit{
					makePresubmit("modified-job", "some-image"),
					makePresubmit("existing-job", "other-image"),
				}))
			baseref, err = gitrepo.RevParse("foo", "bar", "HEAD")
			Expect(err).ShouldNot(HaveOccurred())
		})

	})

	AfterEach(func() {
		if gitClientFactory != nil {
			Expect(gitClientFactory.Clean()).To(Succeed())
		}
	})

	Context("A valid pull request event", func() {

		When("User is an org member", func() {

			BeforeEach(func() {
				registerTestUserAsOrgMember(gh)
			})

			It("Should generate Prow jobs for the changed configs", func() {
				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makePullRequestEvent(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &event.PullRequest,
				}

				letEventsHandlerHandleMadePullRequestEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(1))
				pjAction := prowc.Actions()[0].GetResource()
				Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
			})

			It("Should not generate Prow jobs if there are unrelated changes", func() {

				err := gitrepo.AddCommit("foo", "bar", map[string][]byte{
					"some-file": []byte(""),
				})
				Expect(err).ShouldNot(HaveOccurred())
				headref, err := gitrepo.RevParse("foo", "bar", "HEAD")
				Expect(err).ShouldNot(HaveOccurred())

				var event github.PullRequestEvent

				event = makePullRequestEvent(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &event.PullRequest,
				}

				letEventsHandlerHandleMadePullRequestEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

			It("Should not generate Prow jobs if a job was deleted", func() {
				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("existing-job", "other-image"),
				})

				event := makePullRequestEvent(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &event.PullRequest,
				}

				letEventsHandlerHandleMadePullRequestEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

			It("Should act on pull request event if always run is set to false", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmitWithAlwaysRun("modified-job", "modified-image", false),
					makePresubmit("existing-job", "other-image"),
				})

				event := makePullRequestEvent(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &event.PullRequest,
				}

				letEventsHandlerHandleMadePullRequestEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(1))
				pjAction := prowc.Actions()[0].GetResource()
				Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
			})

			It("Should not generate Prow jobs if job is rehearsal restricted", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmitWithRestrictedAnnotation("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makePullRequestEvent(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &event.PullRequest,
				}

				letEventsHandlerHandleMadePullRequestEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

		})

		When("user is not an org member but ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changed configs with ok-to-test label", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makePullRequestEvent(baseref, headref, []github.Label{
					{
						Name: "ok-to-test",
					},
				})
				gh.PullRequests = map[int]*github.PullRequest{
					17: &event.PullRequest,
				}

				letEventsHandlerHandleMadePullRequestEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(1))
				pjAction := prowc.Actions()[0].GetResource()
				Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
			})

		})

		When("Unauthorized user", func() {

			It("Should not generate Prow jobs", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makePullRequestEvent(baseref, headref, nil)

				gh.PullRequests = map[int]*github.PullRequest{
					17: &event.PullRequest,
				}

				letEventsHandlerHandleMadePullRequestEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

		})

	})

	Context("A valid comment event", func() {

		openIssueWithPullRequest := github.Issue{
			Number: 17,
			State:  "open",
			User: github.User{
				Login: testUserName,
			},
			PullRequest: &struct{}{},
		}

		When("User is an org member", func() {

			BeforeEach(func() {
				registerTestUserAsOrgMember(gh)
			})

			It("Should generate Prow jobs for the changed configs", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makeRehearseCommentEvent(openIssueWithPullRequest)
				pr := makePullRequest(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &pr,
				}

				letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(1))
				pjAction := prowc.Actions()[0].GetResource()
				Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
			})

			It("Should not generate Prow jobs if there are unrelated changes", func() {

				err := gitrepo.AddCommit("foo", "bar", map[string][]byte{
					"some-file": []byte(""),
				})
				Expect(err).ShouldNot(HaveOccurred())
				headref, err := gitrepo.RevParse("foo", "bar", "HEAD")
				Expect(err).ShouldNot(HaveOccurred())

				var event github.IssueCommentEvent

				event = makeRehearseCommentEvent(openIssueWithPullRequest)
				pr := makePullRequest(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &pr,
				}

				letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

			It("Should not generate Prow jobs if a job was deleted", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("existing-job", "other-image"),
				})

				registerTestUserAsOrgMember(gh)

				event := makeRehearseCommentEvent(openIssueWithPullRequest)
				pr := makePullRequest(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &pr,
				}

				letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

			It("Should not generate Prow jobs if PR is not open", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				notOpenIssue := github.Issue{
					Number: 17,
					User: github.User{
						Login: testUserName,
					},
					PullRequest: &struct{}{},
				}
				event := makeRehearseCommentEvent(notOpenIssue)
				pr := makePullRequest(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &pr,
				}

				letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

			It("Should not generate Prow jobs if a job is rehearsal restricted", func() {

				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmitWithRestrictedAnnotation("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makeRehearseCommentEvent(openIssueWithPullRequest)
				pr := makePullRequest(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &pr,
				}

				letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

		})

		When("user is not an org member but ok-to-test label is set", func() {

			It("Should generate Prow jobs for the changed configs", func() {

				By("Generating a head commit with a modified job")
				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makeRehearseCommentEvent(openIssueWithPullRequest)
				pr := makePullRequest(baseref, headref, []github.Label{
					{
						Name: "ok-to-test",
					},
				})
				gh.PullRequests = map[int]*github.PullRequest{
					17: &pr,
				}

				letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(1))

				pjAction := prowc.Actions()[0].GetResource()
				Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
			})

		})

		When("Unauthorized user", func() {

			It("Should not generate Prow jobs", func() {

				By("Generating a head commit with a modified job")
				headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
					makePresubmit("modified-job", "modified-image"),
					makePresubmit("existing-job", "other-image"),
				})

				event := makeRehearseCommentEvent(openIssueWithPullRequest)
				pr := makePullRequest(baseref, headref, nil)
				gh.PullRequests = map[int]*github.PullRequest{
					17: &pr,
				}

				letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler, &event)

				Expect(prowc.Actions()).Should(HaveLen(0))
			})

		})

	})

})

func makePullRequestEvent(baseref, headref string, labels []github.Label) github.PullRequestEvent {
	return github.PullRequestEvent{
		Action: github.PullRequestActionOpened,
		GUID:   "guid",
		Repo: github.Repo{
			FullName: "foo/bar",
		},
		Sender: github.User{
			Login: testUserName,
		},
		PullRequest: makePullRequest(baseref, headref, labels),
	}
}

func makePullRequest(baseref, headref string, labels []github.Label) github.PullRequest {
	return github.PullRequest{
		Number: 17,
		Labels: labels,
		Base:   makePullRequestBranch(baseref),
		Head:   makePullRequestBranch(headref),
	}
}

func makePullRequestBranch(ref string) github.PullRequestBranch {
	return github.PullRequestBranch{
		Repo: github.Repo{
			Name:     "bar",
			FullName: "foo/bar",
		},
		Ref: ref,
		SHA: ref,
	}
}

func makeRehearseCommentEvent(issue github.Issue) github.IssueCommentEvent {
	return github.IssueCommentEvent{
		Action: github.IssueCommentActionCreated,
		Comment: github.IssueComment{
			Body: "/rehearse",
			User: github.User{
				Login: testUserName,
			},
		},
		GUID: "guid",
		Repo: github.Repo{
			FullName: "foo/bar",
		},
		Issue: issue,
	}
}

func commitChangedJobConfigurations(gitrepo *localgit.LocalGit, presubmits []config.Presubmit) (headref string) {
	headConfig := expectGenerateMarshalledConfigWithJobsToSucceed(presubmits)
	expectAddCommitForJobsConfigToSucceed(gitrepo, headConfig)
	headref, err := gitrepo.RevParse("foo", "bar", "HEAD")
	Expect(err).ShouldNot(HaveOccurred())
	return headref
}

func letEventsHandlerHandleMadeIssueCommentEvent(eventsHandler *handler.GitHubEventsHandler, event *github.IssueCommentEvent) {
	handlerEvent, err := makeHandlerIssueCommentEvent(event)
	Expect(err).ShouldNot(HaveOccurred())
	eventsHandler.Handle(handlerEvent)
}

func letEventsHandlerHandleMadePullRequestEvent(eventsHandler *handler.GitHubEventsHandler, event *github.PullRequestEvent) {
	handlerEvent, err := makeHandlerPullRequestEvent(event)
	Expect(err).ShouldNot(HaveOccurred())
	eventsHandler.Handle(handlerEvent)
}

func expectAddCommitForJobsConfigToSucceed(gitrepo *localgit.LocalGit, bytes []byte) bool {
	return Expect(gitrepo.AddCommit("foo", "bar", map[string][]byte{
		"jobs-config.yaml": bytes,
	})).Should(Succeed())
}

func registerTestUserAsOrgMember(gh *fakegithub.FakeClient) {
	gh.OrgMembers = map[string][]string{
		"foo": {
			testUserName,
		},
	}
}

func expectGenerateMarshalledConfigWithJobsToSucceed(presubmits []config.Presubmit) []byte {
	bytes, err := json.Marshal(&config.Config{
		JobConfig: config.JobConfig{
			PresubmitsStatic: map[string][]config.Presubmit{
				"foo/bar": presubmits,
			},
		},
	})
	Expect(err).ToNot(HaveOccurred())
	return bytes
}

func makePresubmit(jobName string, imageName string) config.Presubmit {
	return config.Presubmit{
		JobBase: config.JobBase{
			Name: jobName,
			Spec: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: imageName,
					},
				},
			},
		},
	}
}

func makePresubmitWithAlwaysRun(jobName string, imageName string, alwaysRun bool) config.Presubmit {
	return config.Presubmit{
		JobBase: config.JobBase{
			Name: jobName,
			Spec: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: imageName,
					},
				},
			},
		},
		AlwaysRun: alwaysRun,
	}
}

func makePresubmitWithRestrictedAnnotation(jobName string, imageName string) config.Presubmit {
	return config.Presubmit{
		JobBase: config.JobBase{
			Name: jobName,
			Annotations: map[string]string{
				handler.RehearsalRestrictedAnnotation: "true",
			},
			Spec: &v1.PodSpec{
				Containers: []v1.Container{
					{
						Image: imageName,
					},
				},
			},
		},
	}
}

func makeFakeGitRepository(lg *localgit.LocalGit, repo, org string) error {
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
