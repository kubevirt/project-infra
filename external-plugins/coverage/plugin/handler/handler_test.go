package handler

import (
	"encoding/json"
	"fmt"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/testing"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1/fake"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
)

// Helper function to create a payload for a pull request event with default values
func createPREventPayload(action github.PullRequestEventAction, prNumber int, org, repo string) []byte {
	prEvent := github.PullRequestEvent{
		Action: action,
		PullRequest: github.PullRequest{
			Number: prNumber,
			Base: github.PullRequestBranch{
				Ref: "main",
				SHA: "sha-1",
				Repo: github.Repo{
					Name:     repo,
					Owner:    github.User{Login: org},
					FullName: fmt.Sprintf("%s/%s", org, repo),
				},
			},
			Head: github.PullRequestBranch{
				SHA: "sha-2",
			},
			User: github.User{Login: "testuser"},
		},
	}
	payload, _ := json.Marshal(prEvent)
	return payload
}

var _ = Describe("detectGoFileChanges", func() {
	DescribeTable("Should return true when a file is in a watched path",
		func(files []string) {
			Expect(detectGoFileChanges(files)).To(BeTrue())
		},
		Entry("external-plugins path", []string{"external-plugins/coverage/plugin/handler/handler.go"}),
		Entry("releng path", []string{"releng/release-tool/release-tool.go"}),
		Entry("robots path", []string{"robots/flakefinder/flakefinder.go"}),
		Entry("pkg path", []string{"pkg/git/blame_test.go"}),
	)

	DescribeTable("Should return false when no relevant file is in the list",
		func(files []string) {
			Expect(detectGoFileChanges(files)).To(BeFalse())
		},
		Entry("github path", []string{"github/ci/prow-deploy/config.yaml"}),
		Entry("root level file", []string{"README.md"}),
		Entry("empty list", []string{}),
	)
})

var _ = Describe("actOnPrEvent", func() {
	DescribeTable("Should return true when the action should trigger the coverage plugin",
		func(action github.PullRequestEventAction) {
			event := &github.PullRequestEvent{Action: action}
			Expect(actOnPrEvent(event)).To(BeTrue())
		},
		Entry("opened", github.PullRequestActionOpened),
		Entry("synchronize", github.PullRequestActionSynchronize),
	)

	DescribeTable("Should return false when the action should not trigger the coverage plugin",
		func(action github.PullRequestEventAction) {
			event := &github.PullRequestEvent{Action: action}
			Expect(actOnPrEvent(event)).To(BeFalse())
		},
		Entry("closed", github.PullRequestActionClosed),
		Entry("labeled", github.PullRequestActionLabeled),
		Entry("edited", github.PullRequestActionEdited),
	)
})

var _ = Describe("generateCoverageJob", func() {
	var (
		handler *GitHubEventsHandler
		pr      *github.PullRequest
		job     prowapi.ProwJob
	)

	BeforeEach(func() {
		handler = &GitHubEventsHandler{
			jobsNamespace: "kubevirt-prow-jobs",
		}

		pr = &github.PullRequest{
			Number: 123,
			Base: github.PullRequestBranch{
				Ref: "main",
				SHA: "sha-1",
				Repo: github.Repo{
					Name:     "project-infra",
					Owner:    github.User{Login: "kubevirt"},
					FullName: "kubevirt/project-infra",
				},
			},
			Head: github.PullRequestBranch{
				SHA: "sha-2",
			},
			User: github.User{Login: "testuser"},
		}

		job = handler.generateCoverageJob(pr, "test-event-123")
	})

	DescribeTable("Should set the correct configuration",
		func(getField func(prowapi.ProwJob) string, expectedValue string) {
			Expect(getField(job)).To(Equal(expectedValue))
		},
		Entry("job name", func(j prowapi.ProwJob) string { return j.Spec.Job }, "coverage-auto"),
		Entry("cluster", func(j prowapi.ProwJob) string { return j.Spec.Cluster }, "kubevirt-prow-control-plane"),
		Entry("coverage-plugin label", func(j prowapi.ProwJob) string { return j.Labels["coverage-plugin"] }, "true"),
		Entry("container image", func(j prowapi.ProwJob) string { return j.Spec.PodSpec.Containers[0].Image }, "quay.io/kubevirtci/golang:v20251218-e7a7fc9"),
	)
	It("Should run make coverage command", func() {
		Expect(job.Spec.PodSpec.Containers[0].Args).To(ContainElement("make coverage"))
	})

	It("Should set the GO_MOD_PATH env variable", func() {
		Expect(job.Spec.PodSpec.Containers[0].Env).To(ContainElement(corev1.EnvVar{
			Name:  "GO_MOD_PATH",
			Value: "go.mod",
		}))
	})
})

var _ = Describe("Handle", func() {
	var (
		handler          *GitHubEventsHandler
		fakeGithubClient *fakegithub.FakeClient
		fakeProwClient   *fake.FakeProwV1
	)

	BeforeEach(func() {
		logger := logrus.New()
		logger.SetOutput(io.Discard) //Discard logs for testing

		fakeGithubClient = fakegithub.NewFakeClient() //Create fake github client

		//Set fake prow client
		fakeProwClient = &fake.FakeProwV1{
			Fake: &testing.Fake{},
		}

		//Create handler
		handler = &GitHubEventsHandler{
			logger:        logger,
			githubClient:  fakeGithubClient,
			prowJobClient: fakeProwClient.ProwJobs("test-namespace"),
			jobsNamespace: "test-namespace",
			dryrun:        false,
		}
	})

	Context("When a PR is opened with Go files changes in a watched path", func() {
		It("Should create a coverage ProwJob", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-1",
				Payload: createPREventPayload(github.PullRequestActionOpened, 100, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[100] = []github.PullRequestChange{
				{Filename: "external-plugins/coverage/plugin/handler/handler.go"},
				{Filename: "pkg/git/blame_test.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(HaveLen(1))
			Expect(fakeProwClient.Actions()[0].GetVerb()).To(Equal("create"))

			createAction := fakeProwClient.Actions()[0].(testing.CreateAction)
			prowJob := createAction.GetObject().(*prowapi.ProwJob)
			Expect(prowJob.Spec.Job).To(Equal("coverage-auto"))

		})

	})

	Context("When a PR is synchronized with Go files changes in a watched path", func() {
		It("Should create a coverage ProwJob", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-2",
				Payload: createPREventPayload(github.PullRequestActionSynchronize, 101, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[101] = []github.PullRequestChange{
				{Filename: "releng/release-tool/release-tool.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(HaveLen(1))
			Expect(fakeProwClient.Actions()[0].GetVerb()).To(Equal("create"))

			createAction := fakeProwClient.Actions()[0].(testing.CreateAction)
			prowJob := createAction.GetObject().(*prowapi.ProwJob)
			Expect(prowJob.Spec.Job).To(Equal("coverage-auto"))
		})

	})

	Context("When a PR has mixed go and non go file changes in a watched path", func() {
		It("Should create a coverage ProwJob", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-3",
				Payload: createPREventPayload(github.PullRequestActionOpened, 102, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[102] = []github.PullRequestChange{
				{Filename: "external-plugins/coverage/plugin/handler/handler.go"},
				{Filename: "releng/release-tool/release-tool.go"},
				{Filename: "README.md"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(HaveLen(1))
			Expect(fakeProwClient.Actions()[0].GetVerb()).To(Equal("create"))
		})
	})

	Context("When the PR action is not opened or synchronize", func() {
		DescribeTable("Should not create a job",
			func(action github.PullRequestEventAction, prNumber int) {
				event := &GitHubEvent{
					Type:    "pull_request",
					GUID:    "event-guid-4",
					Payload: createPREventPayload(action, prNumber, "kubevirt", "project-infra"),
				}
				fakeGithubClient.PullRequestChanges[prNumber] = []github.PullRequestChange{
					{Filename: "pkg/test.go"},
				}
				handler.Handle(event)

				Expect(fakeProwClient.Actions()).To(BeEmpty())
			},
			Entry("PR closed", github.PullRequestActionClosed, 200),
			Entry("PR labeled", github.PullRequestActionLabeled, 201),
			Entry("PR edited", github.PullRequestActionEdited, 202),
		)
	})

	Context("When the event type is not a pull_request", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "push",
				GUID:    "event-guid-push",
				Payload: []byte("{}"),
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})
	
	Context("When the PR has no Go file changes", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-5",
				Payload: createPREventPayload(github.PullRequestActionOpened, 300, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[300] = []github.PullRequestChange{
				{Filename: "README.md"},
				{Filename: "docs/guide.txt"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When Go files are only in unwatched directories", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-6",
				Payload: createPREventPayload(github.PullRequestActionOpened, 301, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[301] = []github.PullRequestChange{
				{Filename: "github/ci/test.go"},
				{Filename: "scripts/helper.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the PR has no file changes", func() {
		It("Should not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-7",
				Payload: createPREventPayload(github.PullRequestActionOpened, 302, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[302] = []github.PullRequestChange{}

			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the payload is invalid JSON", func() {
		It("Should not panic and not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-bad",
				Payload: []byte("invalid json{{{"),
			}

			Expect(func() { handler.Handle(event)}).NotTo(Panic())
			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When the payload is empty", func() {
		It("Should not panic and not create a job", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-empty",
				Payload: []byte{},
			}

			Expect(func() { handler.Handle(event)}).NotTo(Panic())
			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})

	Context("When dry-run mode is enabled", func() {
		BeforeEach(func() {
			handler.dryrun = true
		})

		It("Should not create a ProwJob even with valid Go file changes", func() {
			event := &GitHubEvent{
				Type:    "pull_request",
				GUID:    "event-guid-dryrun",
				Payload: createPREventPayload(github.PullRequestActionOpened, 400, "kubevirt", "project-infra"),
			}
			fakeGithubClient.PullRequestChanges[400] = []github.PullRequestChange{
				{Filename: "external-plugins/coverage/plugin/handler/handler.go"},
			}
			handler.Handle(event)

			Expect(fakeProwClient.Actions()).To(BeEmpty())
		})
	})
})
