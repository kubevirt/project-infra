package handler

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"kubevirt.io/project-infra/pkg/testutils"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/git/localgit"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/github/fakegithub"
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
			foc := &testutils.FakeOwnersClient{
				ExistingTopLevelApprovers: sets.New[string]("testuser"),
			}
			froc := &testutils.FakeRepoownersClient{
				Foc: foc,
			}
			eventsServer = NewGitHubEventsHandler(nil, dummyLog, nil, nil, "prow-config.yaml", "", true, gitClientFactory, froc)
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				_ = gitClientFactory.Clean()
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

		It("Should reset SourcePath for postsubmits and periodics", func() {
			prowConfig := config.ProwConfig{}
			jobsConfig := config.JobConfig{
				PostsubmitsStatic: map[string][]config.Postsubmit{
					"foo/bar": {
						{
							JobBase: config.JobBase{
								Name: "a-postsubmit",
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
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Name: "a-periodic",
							Spec: &v1.PodSpec{
								Containers: []v1.Container{
									{
										Image:   "foo/var",
										Command: []string{"/bin/foo"},
									},
								},
							},
						},
						Cron: "0 0 * * *",
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

			outPostsubmits, exists := outConfig.PostsubmitsStatic["foo/bar"]
			Expect(exists).To(BeTrue())
			Expect(outPostsubmits[0].Name).To(Equal("a-postsubmit"))
			Expect(outPostsubmits[0].JobBase.SourcePath).To(HaveSuffix("jobs-config.yaml"))

			Expect(outConfig.Periodics).To(HaveLen(1))
			Expect(outConfig.Periodics[0].Name).To(Equal("a-periodic"))
			Expect(outConfig.Periodics[0].JobBase.SourcePath).To(HaveSuffix("jobs-config.yaml"))
		})

	})

	Context("Unchanged job lookup", func() {
		var gitrepo *localgit.LocalGit
		var gitClientFactory gitv2.ClientFactory
		var eventsServer *GitHubEventsHandler
		var dummyLog *logrus.Logger

		BeforeEach(func() {
			var err error
			gitrepo, gitClientFactory, err = localgit.NewV2()
			Expect(err).ShouldNot(HaveOccurred())
			dummyLog = logrus.New()
			foc := &testutils.FakeOwnersClient{
				ExistingTopLevelApprovers: sets.New[string]("testuser"),
			}
			froc := &testutils.FakeRepoownersClient{Foc: foc}
			eventsServer = NewGitHubEventsHandler(nil, dummyLog, nil, nil, "prow-config.yaml", "jobs/", true, gitClientFactory, froc)
		})

		AfterEach(func() {
			if gitClientFactory != nil {
				_ = gitClientFactory.Clean()
			}
		})

		It("finds an unchanged presubmit by name", func() {
			prowConfig := config.ProwConfig{}
			jobsConfig := config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					"kubevirt/kubevirt": {
						{
							JobBase: config.JobBase{
								Name: "pull-kubevirt-e2e-test",
								Spec: newPodSpec(),
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
				"prow-config.yaml":  prowConfigBytes,
				"jobs/kubevirt.yaml": jobsConfigBytes,
			}
			Expect(gitrepo.AddCommit("foo", "bar", files)).Should(Succeed())
			gitClient, err := gitClientFactory.ClientFor("foo", "bar")
			Expect(err).ShouldNot(HaveOccurred())

			pr := &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "project-infra",
					},
				},
			}
			logEntry := logrus.NewEntry(dummyLog)
			jobs := eventsServer.lookupUnchangedJobs(logEntry, []string{"pull-kubevirt-e2e-test"}, gitClient, pr, "42", nil)
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].Spec.Job).To(Equal("pull-kubevirt-e2e-test"))
		})

		It("finds an unchanged periodic by name", func() {
			prowConfig := config.ProwConfig{}
			jobsConfig := config.JobConfig{
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Name: "periodic-kubevirt-flakefinder",
							Spec: newPodSpec(),
						},
						Cron: "0 0 * * *",
					},
				},
			}

			Expect(gitrepo.MakeFakeRepo("foo", "bar")).Should(Succeed())
			prowConfigBytes, err := json.Marshal(prowConfig)
			Expect(err).ShouldNot(HaveOccurred())
			jobsConfigBytes, err := json.Marshal(jobsConfig)
			Expect(err).ShouldNot(HaveOccurred())
			files := map[string][]byte{
				"prow-config.yaml":    prowConfigBytes,
				"jobs/periodics.yaml": jobsConfigBytes,
			}
			Expect(gitrepo.AddCommit("foo", "bar", files)).Should(Succeed())
			gitClient, err := gitClientFactory.ClientFor("foo", "bar")
			Expect(err).ShouldNot(HaveOccurred())

			pr := &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "project-infra",
					},
				},
			}
			logEntry := logrus.NewEntry(dummyLog)
			jobs := eventsServer.lookupUnchangedJobs(logEntry, []string{"periodic-kubevirt-flakefinder"}, gitClient, pr, "42", nil)
			Expect(jobs).To(HaveLen(1))
			Expect(jobs[0].Spec.Job).To(Equal("periodic-kubevirt-flakefinder"))
			Expect(jobs[0].Spec.Type).To(Equal(prowapi.PeriodicJob))
		})

		It("returns empty when job name is not found", func() {
			prowConfig := config.ProwConfig{}
			jobsConfig := config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					"kubevirt/kubevirt": {
						{
							JobBase: config.JobBase{
								Name: "some-other-job",
								Spec: newPodSpec(),
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
				"prow-config.yaml":  prowConfigBytes,
				"jobs/kubevirt.yaml": jobsConfigBytes,
			}
			Expect(gitrepo.AddCommit("foo", "bar", files)).Should(Succeed())
			gitClient, err := gitClientFactory.ClientFor("foo", "bar")
			Expect(err).ShouldNot(HaveOccurred())

			pr := &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "project-infra",
					},
				},
			}
			logEntry := logrus.NewEntry(dummyLog)
			jobs := eventsServer.lookupUnchangedJobs(logEntry, []string{"nonexistent-job"}, gitClient, pr, "42", nil)
			Expect(jobs).To(BeEmpty())
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

		DescribeTable(
			"Should strip extra refs matching the PR repo",
			func(refs []prowapi.Refs, org, repo string, expected []prowapi.Refs) {
				result := stripExtraRefsForRepo(refs, org, repo)
				Expect(result).To(Equal(expected))
			},
			Entry(
				"Strips matching ref",
				[]prowapi.Refs{
					{Org: "kubevirt", Repo: "project-infra", BaseRef: "main"},
					{Org: "kubevirt", Repo: "kubevirt", BaseRef: "main"},
				},
				"kubevirt", "project-infra",
				[]prowapi.Refs{
					{Org: "kubevirt", Repo: "kubevirt", BaseRef: "main"},
				},
			),
			Entry(
				"No match leaves refs unchanged",
				[]prowapi.Refs{
					{Org: "kubevirt", Repo: "kubevirt", BaseRef: "main"},
				},
				"kubevirt", "project-infra",
				[]prowapi.Refs{
					{Org: "kubevirt", Repo: "kubevirt", BaseRef: "main"},
				},
			),
			Entry(
				"Nil refs returns nil",
				nil,
				"kubevirt", "project-infra",
				nil,
			),
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

})

var _ = Describe("PR filtering", func() {

	Context("Handler filtering jobs", func() {

		var handler *GitHubEventsHandler
		var headConfig *config.Config
		var headConfigPresubmit config.Presubmit
		var baseConfig *config.Config
		var baseConfigPresubmit config.Presubmit
		var pr *github.PullRequest

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			headConfigPresubmit = config.Presubmit{
				JobBase: config.JobBase{
					Name: "testJob",
					Spec: newPodSpec(),
				},
				AlwaysRun:           false,
				Optional:            false,
				Trigger:             "",
				RerunCommand:        "",
				Brancher:            config.Brancher{},
				RegexpChangeMatcher: config.RegexpChangeMatcher{},
				Reporter:            config.Reporter{},
				JenkinsSpec:         nil,
			}
			headConfig = &config.Config{
				JobConfig: config.JobConfig{
					Presets: nil,
					PresubmitsStatic: map[string][]config.Presubmit{
						"kubevirt/kubevirt": {
							headConfigPresubmit,
						},
					},
					PostsubmitsStatic: nil,
					Periodics:         nil,
					AllRepos:          nil,
					ProwYAMLGetter:    nil,
					DecorateAllJobs:   false,
				},
			}
			baseConfigPresubmit = config.Presubmit{
				JobBase: config.JobBase{
					Name: "testJob",
					Spec: newPodSpec(),
				},
				AlwaysRun:           false,
				Optional:            false,
				Trigger:             "",
				RerunCommand:        "",
				Brancher:            config.Brancher{},
				RegexpChangeMatcher: config.RegexpChangeMatcher{},
				Reporter:            config.Reporter{},
				JenkinsSpec:         nil,
			}
			baseConfig = &config.Config{
				JobConfig: config.JobConfig{
					Presets: nil,
					PresubmitsStatic: map[string][]config.Presubmit{
						"kubevirt/kubevirt": {
							baseConfigPresubmit,
						},
					},
					PostsubmitsStatic: nil,
					Periodics:         nil,
					AllRepos:          nil,
					ProwYAMLGetter:    nil,
					DecorateAllJobs:   false,
				},
			}
			pr = &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
					},
				},
			}
		})

		It("doesn't generate a prowjob without changes", func() {
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(presubmits).To(BeEmpty())
		})

		It("generates a prowjob if spec changes", func() {
			headConfigPresubmit.Spec.Containers[0].Image = "v2/test37"
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(presubmits).ToNot(BeEmpty())
		})

		It("generates a prowjob if context changes", func() {
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Cluster = "new-cluster"
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(presubmits).ToNot(BeEmpty())
		})

		It("generates a prowjob for branch if context changes", func() {
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Cluster = "new-cluster"
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Branches = []string{"release-42"}
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(presubmits).ToNot(BeEmpty())
			Expect(presubmits[0].Spec.ExtraRefs[0].BaseRef).To(BeEquivalentTo("release-42"))
		})

		It("strips duplicate extra-refs matching the PR repo", func() {
			pr.Base.Repo.Owner = github.User{Login: "kubevirt"}
			pr.Base.Repo.Name = "project-infra"
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main", WorkDir: true},
			}
			baseConfig.PresubmitsStatic["kubevirt/kubevirt"][0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main", WorkDir: true},
			}
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(presubmits).ToNot(BeEmpty())
			for _, ref := range presubmits[0].Spec.ExtraRefs {
				Expect(ref.Org + "/" + ref.Repo).ToNot(Equal("kubevirt/project-infra"),
					"extra-refs should not contain a duplicate of the PR repo")
			}
		})
	})

	Context("Handler filtering postsubmit jobs", func() {

		var handler *GitHubEventsHandler
		var headConfig *config.Config
		var baseConfig *config.Config
		var pr *github.PullRequest

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			headConfig = &config.Config{
				JobConfig: config.JobConfig{
					PostsubmitsStatic: map[string][]config.Postsubmit{
						"kubevirt/kubevirt": {
							{
								JobBase: config.JobBase{
									Name: "testPostsubmitJob",
									Spec: newPodSpec(),
								},
							},
						},
					},
				},
			}
			baseConfig = &config.Config{
				JobConfig: config.JobConfig{
					PostsubmitsStatic: map[string][]config.Postsubmit{
						"kubevirt/kubevirt": {
							{
								JobBase: config.JobBase{
									Name: "testPostsubmitJob",
									Spec: newPodSpec(),
								},
							},
						},
					},
				},
			}
			pr = &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "project-infra",
					},
				},
			}
		})

		It("doesn't generate a prowjob without changes", func() {
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(postsubmits).To(BeEmpty())
		})

		It("generates a prowjob if spec changes", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(postsubmits).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.Type).To(Equal(prowapi.PostsubmitJob))
		})

		It("generates a prowjob with extra refs for cross-repo job", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(postsubmits).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs[0].Org).To(Equal("kubevirt"))
			Expect(postsubmits[0].Spec.ExtraRefs[0].Repo).To(Equal("kubevirt"))
		})

		It("generates a prowjob for branch if spec changes", func() {
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/test37"
			headConfig.PostsubmitsStatic["kubevirt/kubevirt"][0].Branches = []string{"release-42"}
			postsubmits := handler.generatePostsubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(postsubmits).ToNot(BeEmpty())
			Expect(postsubmits[0].Spec.ExtraRefs[0].BaseRef).To(BeEquivalentTo("release-42"))
		})
	})

	Context("Handler filtering periodic jobs", func() {

		var handler *GitHubEventsHandler
		var headConfig *config.Config
		var baseConfig *config.Config
		var pr *github.PullRequest

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			headConfig = &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "testPeriodicJob",
								Spec: newPodSpec(),
							},
							Cron: "0 0 * * *",
						},
					},
				},
			}
			baseConfig = &config.Config{
				JobConfig: config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Name: "testPeriodicJob",
								Spec: newPodSpec(),
							},
							Cron: "0 0 * * *",
						},
					},
				},
			}
			pr = &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "project-infra",
					},
				},
			}
		})

		It("doesn't generate a prowjob without changes", func() {
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42", nil)
			Expect(periodics).To(BeEmpty())
		})

		It("generates a prowjob if spec changes", func() {
			headConfig.Periodics[0].Spec.Containers[0].Image = "v2/test37"
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42", nil)
			Expect(periodics).ToNot(BeEmpty())
			Expect(periodics[0].Spec.Type).To(Equal(prowapi.PeriodicJob))
		})

		It("injects PR refs into extra refs", func() {
			headConfig.Periodics[0].Spec.Containers[0].Image = "v2/test37"
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42", nil)
			Expect(periodics).ToNot(BeEmpty())
			Expect(periodics[0].Spec.ExtraRefs).ToNot(BeEmpty())
			Expect(periodics[0].Spec.ExtraRefs[len(periodics[0].Spec.ExtraRefs)-1].Org).To(Equal("kubevirt"))
			Expect(periodics[0].Spec.ExtraRefs[len(periodics[0].Spec.ExtraRefs)-1].Repo).To(Equal("project-infra"))
		})

		It("replaces duplicate extra-ref for the PR repo instead of duplicating", func() {
			headConfig.Periodics[0].Spec.Containers[0].Image = "v2/test37"
			headConfig.Periodics[0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main", WorkDir: true},
			}
			baseConfig.Periodics[0].ExtraRefs = []prowapi.Refs{
				{Org: "kubevirt", Repo: "project-infra", BaseRef: "main", WorkDir: true},
			}
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42", nil)
			Expect(periodics).ToNot(BeEmpty())
			prRefCount := 0
			for _, ref := range periodics[0].Spec.ExtraRefs {
				if ref.Org == "kubevirt" && ref.Repo == "project-infra" {
					prRefCount++
					Expect(ref.Pulls).ToNot(BeEmpty(), "the PR repo extra-ref should carry the PR's pull refs")
				}
			}
			Expect(prRefCount).To(Equal(1), "project-infra should appear exactly once in extra-refs")
		})

		It("generates a prowjob for new periodic", func() {
			headConfig.Periodics = append(headConfig.Periodics, config.Periodic{
				JobBase: config.JobBase{
					Name: "newPeriodicJob",
					Spec: newPodSpec(),
				},
				Cron: "0 12 * * *",
			})
			periodics := handler.generatePeriodics(headConfig, baseConfig, pr, "42", nil)
			Expect(periodics).To(HaveLen(1))
			Expect(periodics[0].Spec.Job).To(Equal("newPeriodicJob"))
		})
	})

	Context("extracting job names from PR comments", func() {

		var handler *GitHubEventsHandler

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
		})

		It("extracts job names from comment body", func() {
			commentBody := `/rehearse jobname1 
/rehearse jobname2

Gna meh whatever 

/rehearse jobname3    
`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeEquivalentTo([]string{
				"jobname1",
				"jobname2",
				"jobname3",
			}))
		})

		It("extracts no job names from comment body if all is found", func() {
			commentBody := `Gna meh whatever 

/rehearse all


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeNil())
		})

		It("extracts no job names from comment body if no element is found since only whitespace after command", func() {
			commentBody := `Gna meh whatever 

/rehearse    


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeNil())
		})

		It("extracts no job names from comment body", func() {
			commentBody := `Gna meh whatever 

/rehearse


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeNil())
		})

		It("extracts question mark from comment body", func() {
			commentBody := `Gna meh whatever

/rehearse ?


`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeEquivalentTo([]string{"?"}))
		})

		It("extracts job name only when cross-repo ref is present", func() {
			commentBody := `/rehearse pull-kubevirt-e2e-k8s-1.35-sig-compute kubevirt/kubevirt#1234
`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeEquivalentTo([]string{
				"pull-kubevirt-e2e-k8s-1.35-sig-compute",
			}))
		})

		It("treats 'all' with cross-repo ref as run-all, not a job name filter", func() {
			commentBody := `/rehearse all kubevirt/kubevirt#1234
`
			Expect(handler.extractJobNamesFromComment(commentBody)).To(BeEmpty())
		})
	})

	Context("parsing cross-repo PR targets", func() {

		var handler *GitHubEventsHandler

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
		})

		type crossRepoParseTestData struct {
			body           string
			expectNil      bool
			expectedOrg    string
			expectedRepo   string
			expectedNumber int
		}

		DescribeTable("should parse cross-repo targets",
			func(td crossRepoParseTestData) {
				target := handler.parseCrossRepoTarget(td.body)
				if td.expectNil {
					Expect(target).To(BeNil())
				} else {
					Expect(target).ToNot(BeNil())
					Expect(target.org).To(Equal(td.expectedOrg))
					Expect(target.repo).To(Equal(td.expectedRepo))
					Expect(target.number).To(Equal(td.expectedNumber))
				}
			},
			Entry("parses org/repo#number from comment",
				crossRepoParseTestData{
					body:           `/rehearse pull-kubevirt-e2e-k8s-1.35-sig-compute kubevirt/kubevirt#1234`,
					expectedOrg:    "kubevirt",
					expectedRepo:   "kubevirt",
					expectedNumber: 1234,
				},
			),
			Entry("returns nil when no cross-repo target is present",
				crossRepoParseTestData{
					body:      `/rehearse pull-kubevirt-e2e-k8s-1.35-sig-compute`,
					expectNil: true,
				},
			),
			Entry("returns nil for bare /rehearse",
				crossRepoParseTestData{
					body:      "/rehearse",
					expectNil: true,
				},
			),
			Entry("returns nil for empty body",
				crossRepoParseTestData{
					body:      "",
					expectNil: true,
				},
			),
		)
	})

	Context("cross-repo PR targeting in job generation", func() {

		var handler *GitHubEventsHandler
		var headConfig *config.Config
		var baseConfig *config.Config
		var pr *github.PullRequest
		var targetPR *github.PullRequest

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			headConfig = &config.Config{
				JobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						"kubevirt/kubevirt": {
							{
								JobBase: config.JobBase{
									Name: "pull-kubevirt-e2e-test",
									Spec: newPodSpec(),
								},
							},
						},
					},
				},
			}
			baseConfig = &config.Config{
				JobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						"kubevirt/kubevirt": {
							{
								JobBase: config.JobBase{
									Name: "pull-kubevirt-e2e-test",
									Spec: newPodSpec(),
								},
							},
						},
					},
				},
			}
			pr = &github.PullRequest{
				Base: github.PullRequestBranch{
					Repo: github.Repo{
						FullName: "kubevirt/project-infra",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "project-infra",
					},
				},
			}
			targetPR = &github.PullRequest{
				Number: 5678,
				User:   github.User{Login: "contributor"},
				Base: github.PullRequestBranch{
					Ref: "main",
					SHA: "targetbaseSHA",
					Repo: github.Repo{
						FullName: "kubevirt/kubevirt",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "kubevirt",
					},
				},
				Head: github.PullRequestBranch{
					Ref: "feature-branch",
					SHA: "targetheadSHA",
				},
			}
		})

		It("injects target PR refs into ExtraRefs for presubmit", func() {
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/changed"
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", targetPR)
			Expect(presubmits).ToNot(BeEmpty())
			var kubevirtRef *prowapi.Refs
			for i, ref := range presubmits[0].Spec.ExtraRefs {
				if ref.Org == "kubevirt" && ref.Repo == "kubevirt" {
					kubevirtRef = &presubmits[0].Spec.ExtraRefs[i]
					break
				}
			}
			Expect(kubevirtRef).ToNot(BeNil(), "should have kubevirt/kubevirt in extra-refs")
			Expect(kubevirtRef.Pulls).To(HaveLen(1))
			Expect(kubevirtRef.Pulls[0].Number).To(Equal(5678))
			Expect(kubevirtRef.Pulls[0].SHA).To(Equal("targetheadSHA"))
			Expect(kubevirtRef.Pulls[0].HeadRef).To(Equal("feature-branch"))
		})

		It("uses normal refs when targetPR is nil", func() {
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/changed"
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", nil)
			Expect(presubmits).ToNot(BeEmpty())
			var kubevirtRef *prowapi.Refs
			for i, ref := range presubmits[0].Spec.ExtraRefs {
				if ref.Org == "kubevirt" && ref.Repo == "kubevirt" {
					kubevirtRef = &presubmits[0].Spec.ExtraRefs[i]
					break
				}
			}
			Expect(kubevirtRef).ToNot(BeNil())
			Expect(kubevirtRef.Pulls).To(BeEmpty())
		})

		It("does not inject target PR when target repo doesn't match the job's repo", func() {
			mismatchedTargetPR := &github.PullRequest{
				Number: 9999,
				Base: github.PullRequestBranch{
					Ref: "main",
					Repo: github.Repo{
						FullName: "kubevirt/other-repo",
						Owner:    github.User{Login: "kubevirt"},
						Name:     "other-repo",
					},
				},
				Head: github.PullRequestBranch{
					Ref: "some-branch",
					SHA: "someSHA",
				},
			}
			headConfig.PresubmitsStatic["kubevirt/kubevirt"][0].Spec.Containers[0].Image = "v2/changed"
			presubmits := handler.generatePresubmits(headConfig, baseConfig, pr, "42", mismatchedTargetPR)
			Expect(presubmits).ToNot(BeEmpty())
			var kubevirtRef *prowapi.Refs
			for i, ref := range presubmits[0].Spec.ExtraRefs {
				if ref.Org == "kubevirt" && ref.Repo == "kubevirt" {
					kubevirtRef = &presubmits[0].Spec.ExtraRefs[i]
					break
				}
			}
			Expect(kubevirtRef).ToNot(BeNil())
			Expect(kubevirtRef.Pulls).To(BeEmpty(), "should not inject target PR for mismatched repo")
		})
	})

	Context("filtering jobs by name", func() {

		var handler *GitHubEventsHandler
		var prowJobs []prowapi.ProwJob

		BeforeEach(func() {
			handler = &GitHubEventsHandler{}
			prowJobs = []prowapi.ProwJob{
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob1",
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob2",
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob3",
					},
				},
			}
		})

		It("filters nothing if slice is nil", func() {
			Expect(handler.filterProwJobsByJobNames(prowJobs, nil)).To(BeEquivalentTo(prowJobs))
		})

		It("filters one job", func() {
			expected := []prowapi.ProwJob{
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob1",
					},
				},
			}
			Expect(handler.filterProwJobsByJobNames(prowJobs, []string{"prowJob1"})).To(BeEquivalentTo(expected))
		})

		It("filters two jobs", func() {
			expected := []prowapi.ProwJob{
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob1",
					},
				},
				{
					Spec: prowapi.ProwJobSpec{
						Job: "prowJob3",
					},
				},
			}
			Expect(handler.filterProwJobsByJobNames(prowJobs, []string{"prowJob1", "prowJob3"})).To(BeEquivalentTo(expected))
		})

	})

	Context("canUserRehearse", func() {
		var testable *GitHubEventsHandler
		var fakeGHC *fakegithub.FakeClient
		var pr *github.PullRequest
		var fakeOwnersClient *testutils.FakeOwnersClient
		const userName = "testuser"
		const userNameUpperCase = "TestUser"
		BeforeEach(func() {
			fakeGHC = &fakegithub.FakeClient{}
			fakeGHC.OrgMembers = map[string][]string{
				"testorg": {
					"testauthor",
					userName,
				},
			}
			fakeOwnersClient = &testutils.FakeOwnersClient{}
			fakeRepoownersClient := &testutils.FakeRepoownersClient{
				Foc: fakeOwnersClient,
			}
			testable = &GitHubEventsHandler{
				ghClient:     fakeGHC,
				ownersClient: fakeRepoownersClient,
			}
			pr = &github.PullRequest{
				User: github.User{Login: "testauthor"},
			}
		})

		DescribeTable("org and user",
			func(testData CanUserRehearseOrgAndUserTestData) {
				fakeGHC.OrgMembers = testData.OrgMembers
				fakeGHC.IssueLabelsExisting = testData.IssueLabelsExisting
				fakeOwnersClient.ExistingTopLevelApprovers = testData.TopLevelApprovers
				fakeOwnersClient.CurrentLeafApprovers = testData.LeafApprovers

				canUserRehearse, message := testable.canUserRehearse("testorg", "testrepo", pr, testData.GetUserNameOrDefault(), testData.ChangedFiles)

				Expect(canUserRehearse).To(BeEquivalentTo(testData.ExpectedCanUserRehearse))
				for _, part := range testData.ExpectedMessageParts {
					Expect(message).To(ContainSubstring(part))
				}
			},
			Entry("only author in org",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
						},
					},
					ChangedFiles:            []string{},
					TopLevelApprovers:       sets.Set[string]{},
					LeafApprovers:           map[string]sets.Set[string]{},
					ExpectedCanUserRehearse: false,
				},
			),
			Entry("user and author in org - user is not an approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles: []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{
						"topLevelApprover": struct{}{},
					},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {
							"leafApprover": struct{}{},
						},
					},
					ExpectedCanUserRehearse: false,
					ExpectedMessageParts: []string{
						"@testuser",
						"you need to be an approver",
						"leafApprover",
						"topLevelApprover",
					},
				},
			),
			Entry("user and author in org - user is not an approver, but ok-to-rehearse label is present",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles: []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{
						"topLevelApprover": struct{}{},
					},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {
							"leafApprover": struct{}{},
						},
					},
					ExpectedCanUserRehearse: true,
					IssueLabelsExisting:     []string{fmt.Sprintf("testorg/testrepo#0:%s", OKToRehearse)},
				},
			),
			Entry("user and author in org - user is a top level approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("user and author in org - user is a top level approver (case in OWNERS and GitHub don't match)",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					UserName:                userNameUpperCase,
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("user and author in org - user is a leaf approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:      []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userName: struct{}{}},
					},
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("user and author in org - user is a leaf approver - case in OWNERS and GitHub is different for user - OWNERS lowercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:      []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userName: struct{}{}},
					},
					UserName:                userNameUpperCase,
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("user and author in org - user is a leaf approver - case in OWNERS and GitHub is different for user - OWNERS uppercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles:      []string{"changedFile"},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userNameUpperCase: struct{}{}},
					},
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("user and author in org - user is not a leaf approver for all files",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							"testauthor",
							userName,
						},
					},
					ChangedFiles: []string{
						"changedFile",
						"otherChangedFile",
					},
					TopLevelApprovers: sets.Set[string]{},
					LeafApprovers: map[string]sets.Set[string]{
						"changedFile": {userName: struct{}{}},
					},
					ExpectedCanUserRehearse: false,
				},
			),
			Entry("only user in org - user is top level approver",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("only user in org - user is top level approver - case in OWNERS and GitHub don't match - topLeverApprovers lowercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userName: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					UserName:                userNameUpperCase,
					ExpectedCanUserRehearse: true,
				},
			),
			Entry("only user in org - user is top level approver - case in OWNERS and GitHub don't match - topLeverApprovers uppercase",
				CanUserRehearseOrgAndUserTestData{
					OrgMembers: map[string][]string{
						"testorg": {
							userName,
						},
					},
					ChangedFiles:            []string{"changedFile"},
					TopLevelApprovers:       sets.Set[string]{userNameUpperCase: struct{}{}},
					LeafApprovers:           map[string]sets.Set[string]{},
					UserName:                userName,
					ExpectedCanUserRehearse: true,
				},
			),
		)
	})

})

type CanUserRehearseOrgAndUserTestData struct {
	OrgMembers              map[string][]string
	ChangedFiles            []string
	TopLevelApprovers       sets.Set[string]
	LeafApprovers           map[string]sets.Set[string]
	ExpectedCanUserRehearse bool
	ExpectedMessageParts    []string
	UserName                string
	IssueLabelsExisting     []string
}

func (td CanUserRehearseOrgAndUserTestData) GetUserNameOrDefault() string {
	if td.UserName == "" {
		return "testuser"
	}
	return td.UserName
}

func newPodSpec() *v1.PodSpec {
	return &v1.PodSpec{
		Containers: []v1.Container{
			{
				Name:                     "blah",
				Image:                    "v2/test42",
				Command:                  nil,
				Args:                     nil,
				WorkingDir:               "",
				Ports:                    nil,
				EnvFrom:                  nil,
				Env:                      nil,
				Resources:                v1.ResourceRequirements{},
				VolumeMounts:             nil,
				VolumeDevices:            nil,
				LivenessProbe:            nil,
				ReadinessProbe:           nil,
				StartupProbe:             nil,
				Lifecycle:                nil,
				TerminationMessagePath:   "",
				TerminationMessagePolicy: "",
				ImagePullPolicy:          "",
				SecurityContext:          nil,
				Stdin:                    false,
				StdinOnce:                false,
				TTY:                      false,
			},
		},
	}
}
