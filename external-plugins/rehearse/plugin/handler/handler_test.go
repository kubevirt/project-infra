package handler

import (
	"encoding/json"
	"k8s.io/client-go/testing"
	"k8s.io/test-infra/prow/client/clientset/versioned/typed/prowjobs/v1/fake"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/localgit"
	gitv2 "k8s.io/test-infra/prow/git/v2"
)

const (
	testUserName = "testuser"
	repo         = "foo"
	org          = "bar"
	baseBranch   = "main"
	prBranchName = "my-update-branch"
)

var _ = Describe("Events", func() {

	Context("With a git repo", func() {
		var gitrepo *localgit.LocalGit
		var gitClientFactory gitv2.ClientFactory
		var eventsServer *GitHubEventsHandler
		var dummyLog *logrus.Logger

		BeforeEach(func() {

			var err error
			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred(), "Could not create local git repo and client factory")
			dummyLog = logrus.New()
			eventsServer = NewGitHubEventsHandler(
				nil,
				dummyLog,
				nil,
				nil,
				"prow-config.yaml",
				"",
				true,
				gitClientFactory)
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				gitClientFactory.Clean()
			}
		})

		It("Should load jobs from git refspec", func() {
			prowConfig := config.ProwConfig{}
			jobsConfig := config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					"foo/bar": {
						{
							JobBase: config.JobBase{
								Name: "a-presubmit",
								Spec: &v1.PodSpec{
									Containers: []v1.Container{
										{
											Image:   "foo/var",
											Command: []string{"/bin/foo"},
										},
									},
								},
							},
						},
					},
				},
			}

			Expect(gitrepo.MakeFakeRepo("foo", "bar")).Should(Succeed())
			prowConfigBytes, err := json.Marshal(prowConfig)
			Expect(err).ShouldNot(HaveOccurred())
			jobsConfigBytes, err := json.Marshal(jobsConfig)
			Expect(err).ShouldNot(HaveOccurred())
			files := map[string][]byte{
				"prow-config.yaml": prowConfigBytes,
				"jobs-config.yaml": jobsConfigBytes,
			}
			Expect(gitrepo.AddCommit("foo", "bar", files)).Should(Succeed())
			headref, err := gitrepo.RevParse("foo", "bar", "HEAD")
			Expect(err).ShouldNot(HaveOccurred())
			gitClient, err := gitClientFactory.ClientFor("foo", "bar")
			Expect(err).ShouldNot(HaveOccurred())
			out, err := eventsServer.loadConfigsAtRef([]string{"jobs-config.yaml"}, gitClient, headref)
			Expect(err).ShouldNot(HaveOccurred())
			outConfig, exists := out["jobs-config.yaml"]
			Expect(exists).To(BeTrue())
			outJobs, exists := outConfig.PresubmitsStatic["foo/bar"]
			Expect(exists).To(BeTrue())
			Expect(outJobs[0].Name).To(Equal(jobsConfig.PresubmitsStatic["foo/bar"][0].Name))
		})

	})

	Context("Utility functions", func() {

		It("Should return correct repo from job key", func() {
			ret := repoFromJobKey("foo/bar#baz-something/something-else")
			Expect(ret).To(Equal("foo/bar"))
		})

		DescribeTable(
			"Should calculate extra refs",
			func(refs []prowapi.Refs, expected prowapi.Refs) {
				ret := makeTargetRepoRefs(refs, "foo", "bar", "baz")
				Expect(ret).To(Equal(expected))
				Expect(refs).ToNot(Equal(expected), "Input refs should not be modified")
			},
			Entry(
				"Refs exists and there is no workdir defined",
				[]prowapi.Refs{
					{
						WorkDir: false,
					},
				},
				prowapi.Refs{
					Org:     "foo",
					Repo:    "bar",
					WorkDir: true,
					BaseRef: "baz",
				},
			),
			Entry(
				"Refs is nil",
				nil,
				prowapi.Refs{
					Org:     "foo",
					Repo:    "bar",
					WorkDir: true,
					BaseRef: "baz",
				},
			),
		)

		DescribeTable(
			"Should calculate if a workdir is already defined",
			func(refs []prowapi.Refs, expected bool) {
				Expect(workdirAlreadyDefined(refs)).To(Equal(expected))
			},
			Entry(
				"When workdir is already defined",
				[]prowapi.Refs{
					{
						WorkDir: false,
					},
					{
						WorkDir: true,
					},
				},
				true),
			Entry(
				"When workdir is not defined",
				[]prowapi.Refs{
					{
						WorkDir: false,
					},
					{
						WorkDir: false,
					},
				},
				false),
		)

		It("Should discover HEAD branch name from remote", func() {
			headBranchName, err := discoverHeadBranchName("kubevirt", "kubevirt", "")
			Expect(err).ToNot(HaveOccurred())
			Expect(headBranchName).To(Equal("main"))
		})

		It("Should discover HEAD branch name from cloneURI", func() {
			headBranchName, err := discoverHeadBranchName("foo", "bar", "https://github.com/nmstate/nmstate")
			Expect(err).ToNot(HaveOccurred())
			Expect(headBranchName).To(Equal("base"))
		})

	})

	// TODO:
	// - align duplication with rehearse_test.go
	// - add testcases for
	//   - single job rehearsal (argument job_name)
	//   - presets
	//   - run_if_changed
	//   - skip_if_only_changed
	Context("handleRehearsalForPR", func() {

		var gitrepo *localgit.LocalGit
		var gitClientFactory gitv2.ClientFactory
		var baseref string

		var gh *fakegithub.FakeClient

		var prowc *fake.FakeProwV1
		var fakelog *logrus.Logger

		var eventsChan chan *GitHubEvent
		var eventsHandler *GitHubEventsHandler

		BeforeEach(func() {
			var err error

			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())

			Expect(makeFakeGitRepositoryWithEmptyProwConfig(gitrepo, repo, org)).Should(Succeed())
			Expect(localgit.DefaultBranch(gitrepo.Dir)).To(BeEquivalentTo(baseBranch))

			gh = &fakegithub.FakeClient{}

			prowc = &fake.FakeProwV1{
				Fake: &testing.Fake{},
			}
			fakelog = logrus.New()
			eventsChan = make(chan *GitHubEvent)
			eventsHandler = NewGitHubEventsHandler(
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
				baseref, err = gitrepo.RevParse(repo, org, "HEAD")
				Expect(err).ShouldNot(HaveOccurred())
			})

			By("Checking out a new branch for the PR", func() {
				gitrepo.CheckoutNewBranch(org, repo, prBranchName)
			})

			registerTestUserAsOrgMember(gh)
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				Expect(gitClientFactory.Clean()).To(Succeed())
			}
		})

		It("Should generate Prow jobs for the changed configs", func() {

			headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "modified-image"),
				makePresubmit("existing-job", "other-image"),
			})

			pr := makePullRequest(baseref, headref, nil)
			gh.PullRequests = map[int]*github.PullRequest{
				17: &pr,
			}

			eventLog := log.WithField("event-guid", "guid")
			eventsHandler.handleRehearsalForPR(eventLog, &pr, "guid")

			Expect(prowc.Actions()).Should(HaveLen(1))
			pjAction := prowc.Actions()[0].GetResource()
			Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
		})

		It("Should generate Prow jobs for the changed configs (existing-job has been updated on base branch before PR commit)", func() {

			gitrepo.Checkout(org, repo, baseBranch)
			_ = commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "some-image"),
				makePresubmit("existing-job", "other-modified-image"),
			})
			gitrepo.Checkout(org, repo, prBranchName)

			headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "modified-image"),
				makePresubmit("existing-job", "other-image"),
			})

			pr := makePullRequest(baseref, headref, nil)
			gh.PullRequests = map[int]*github.PullRequest{
				17: &pr,
			}

			eventLog := log.WithField("event-guid", "guid")
			eventsHandler.handleRehearsalForPR(eventLog, &pr, "guid")

			Expect(prowc.Actions()).Should(HaveLen(1))
			pjAction := prowc.Actions()[0].GetResource()
			Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
		})

		It("Should generate Prow jobs for the changed configs (existing-job has been updated on base branch after PR commit)", func() {

			headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "modified-image"),
				makePresubmit("existing-job", "other-image"),
			})

			gitrepo.Checkout(org, repo, baseBranch)
			_ = commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "some-image"),
				makePresubmit("existing-job", "other-modified-image"),
			})
			gitrepo.Checkout(org, repo, prBranchName)

			pr := makePullRequest(baseref, headref, nil)
			gh.PullRequests = map[int]*github.PullRequest{
				17: &pr,
			}

			eventLog := log.WithField("event-guid", "guid")
			eventsHandler.handleRehearsalForPR(eventLog, &pr, "guid")

			Expect(prowc.Actions()).Should(HaveLen(1))
			pjAction := prowc.Actions()[0].GetResource()
			Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
		})

		It("Should generate Prow jobs for the changed configs (modified-job has been updated on base branch before PR commit)", func() {

			gitrepo.Checkout(org, repo, baseBranch)
			_ = commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "other-modified-image"),
				makePresubmit("existing-job", "other-image"),
			})
			gitrepo.Checkout(org, repo, prBranchName)

			headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "modified-image"),
				makePresubmit("existing-job", "other-image"),
			})

			pr := makePullRequest(baseref, headref, nil)
			gh.PullRequests = map[int]*github.PullRequest{
				17: &pr,
			}

			eventLog := log.WithField("event-guid", "guid")
			eventsHandler.handleRehearsalForPR(eventLog, &pr, "guid")

			Expect(prowc.Actions()).Should(HaveLen(1))
			pjAction := prowc.Actions()[0].GetResource()
			Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
		})

		It("Should generate Prow jobs for the changed configs (modified-job has been updated on base branch after PR commit)", func() {

			headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "modified-image"),
				makePresubmit("existing-job", "other-image"),
			})

			gitrepo.Checkout(org, repo, baseBranch)
			_ = commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("modified-job", "other-modified-image"),
				makePresubmit("existing-job", "other-image"),
			})
			gitrepo.Checkout(org, repo, prBranchName)

			pr := makePullRequest(baseref, headref, nil)
			gh.PullRequests = map[int]*github.PullRequest{
				17: &pr,
			}

			eventLog := log.WithField("event-guid", "guid")
			eventsHandler.handleRehearsalForPR(eventLog, &pr, "guid")

			Expect(prowc.Actions()).Should(HaveLen(1))
			pjAction := prowc.Actions()[0].GetResource()
			Expect(pjAction).To(Equal(prowapi.SchemeGroupVersion.WithResource("prowjobs")))
		})

		It("Should not generate Prow jobs if there are unrelated changes", func() {

			err := gitrepo.AddCommit(repo, org, map[string][]byte{
				"some-file": []byte(""),
			})
			Expect(err).ShouldNot(HaveOccurred())
			headref, err := gitrepo.RevParse(repo, org, "HEAD")
			Expect(err).ShouldNot(HaveOccurred())

			pr := makePullRequest(baseref, headref, nil)
			gh.PullRequests = map[int]*github.PullRequest{
				17: &pr,
			}

			eventLog := log.WithField("event-guid", "guid")
			eventsHandler.handleRehearsalForPR(eventLog, &pr, "guid")

			Expect(prowc.Actions()).Should(HaveLen(0))
		})

		It("Should not generate Prow jobs if a job was deleted", func() {

			headref := commitChangedJobConfigurations(gitrepo, []config.Presubmit{
				makePresubmit("existing-job", "other-image"),
			})

			registerTestUserAsOrgMember(gh)

			pr := makePullRequest(baseref, headref, nil)
			gh.PullRequests = map[int]*github.PullRequest{
				17: &pr,
			}

			eventLog := log.WithField("event-guid", "guid")
			eventsHandler.handleRehearsalForPR(eventLog, &pr, "guid")

			Expect(prowc.Actions()).Should(HaveLen(0))
		})

	})
})

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

func commitChangedJobConfigurations(gitrepo *localgit.LocalGit, presubmits []config.Presubmit) (headref string) {
	headConfig := expectGenerateMarshalledConfigWithJobsToSucceed(presubmits)
	expectAddCommitForJobsConfigToSucceed(gitrepo, headConfig)
	headref, err := gitrepo.RevParse("foo", "bar", "HEAD")
	Expect(err).ShouldNot(HaveOccurred())
	return headref
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

func makeFakeGitRepositoryWithEmptyProwConfig(lg *localgit.LocalGit, repo, org string) error {
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
