/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2020 Red Hat, Inc.
 *
 */

package handler

import (
	"fmt"
	"io/ioutil"
	"k8s.io/test-infra/prow/kube"
	"path/filepath"
	"reflect"
	"sigs.k8s.io/yaml"
	"strings"

	"github.com/sirupsen/logrus"

	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git"
	gitv2 "k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pjutil"
)

// Map from a the path the config was found it to the config itself
type jobConfigPath map[string]*config.Config

// HandlePullRequestEvent handles new pull request events
func HandlePullRequestEvent(cl *kube.Client, event *github.PullRequestEvent, prowConfigPath, jobConfigPatterns string) {
	// Make sure we do not crash the program
	defer logOnPanic()

	if !shouldPerformOnEventAction(event) {
		logrus.Infof("nothing to do on pr %d", event.PullRequest.Number)
		return
	}

	gitClient, err := git.NewClient()
	if err != nil {
		logrus.WithError(err).Fatal("could not initialize git client")
		return
	}

	defer gitClient.Clean()

	// Disable authentication since we don't really need it
	gitClient.SetCredentials("", func() []byte { return []byte{} })

	org, name, err := gitv2.OrgRepo(event.Repo.FullName)
	if err != nil {
		logrus.WithError(err).Fatal("could not extract repo org and name")
		return
	}
	repo, err := gitClient.Clone(org, name)
	if err != nil {
		logrus.WithError(err).Fatal("could not clone repo")
		return
	}

	err = repo.CheckoutPullRequest(event.PullRequest.Number)
	if err != nil {
		logrus.WithError(err).Fatal("could not fetch pull request")
		return
	}

	// TODO: handle changes to config.yaml and presets
	modifiedJobConfigs, err := getModifiedConfigs(repo, event, jobConfigPatterns)
	if err != nil {
		logrus.WithError(err).Fatal("could not get the modified configs")
		return
	}

	if len(modifiedJobConfigs) == 0 {
		logrus.Infoln("no job configs were modified - nothing to do")
		return
	}

	modifiedConfigs, err := loadConfigsAtRef(repo, event.PullRequest.Head.SHA, prowConfigPath, modifiedJobConfigs)
	if err != nil {
		logrus.WithError(err).Fatal("could not load the modified configs from pull request's head")
		return
	}

	originalConfigs, err := loadConfigsAtRef(repo, event.PullRequest.Base.SHA, prowConfigPath, modifiedJobConfigs)
	if err != nil {
		logrus.WithError(err).Fatal("could not load the modified configs from pull request's base")
		return
	}

	squashedConfigs := squashConfigPaths(originalConfigs, modifiedConfigs)

	prowJobs := generateProwJobs(squashedConfigs, event)
	createProwJobs(cl, prowJobs)
}

// Creates prow jobs
func createProwJobs(cl *kube.Client, jobs []prowapi.ProwJob) {
	for _, job := range jobs {
		_, err := cl.CreateProwJob(job)
		if err != nil {
			logrus.WithError(err).Fatal("failed to create new ProwJob")
		}
	}
}

// Get the modified job configs from the repo
func getModifiedConfigs(repo *git.Repo, event *github.PullRequestEvent, pattern string) ([]string, error) {
	changes, err := repo.Diff(event.PullRequest.Head.SHA, event.PullRequest.Base.SHA)
	if err != nil {
		return nil, err
	}
	return filterByPattern(changes, pattern)
}

// Filter the input array/slice by the given pattern.
// The pattern has to be a shell file name pattern.
// https://golang.org/pkg/path/filepath/#Match
func filterByPattern(input []string, pattern string) ([]string, error) {
	var out []string
	for _, s := range input {
		match, err := filepath.Match(pattern, s)
		if err != nil {
			// The only possible error is ErrorBadPattern
			return nil, err
		}
		if match {
			out = append(out, s)
		}
	}
	return out, nil
}

// Determine if we should perform on the given event.
func shouldPerformOnEventAction(event *github.PullRequestEvent) bool {
	// See https://developer.github.com/v3/activity/events/types/#pullrequestevent
	// for details
	switch event.Action {
	case "opened":
		return true
	case "edited":
		return true
	case "synchronize":
		return true
	default:
		return false
	}
}

// Load job configs but checkout before
func loadConfigsAtRef(
	repo *git.Repo, ref, prowConfigPath string, jobConfigPaths []string) (jobConfigPath, error) {
	if err := repo.Checkout(ref); err != nil {
		return nil, err
	}

	return loadConfigs(repo.Directory(), prowConfigPath, jobConfigPaths), nil
}

// Squash original and modified configs at the same path, returning an array of configs
// that contain only new and modified job configs.
func squashConfigPaths(originalConfigs, modifiedConfigs jobConfigPath) []*config.Config {
	configs := make([]*config.Config, 0, len(modifiedConfigs))
	for path, headConfig := range modifiedConfigs {
		baseConfig, exists := originalConfigs[path]
		if !exists {
			// new config
			configs = append(configs, headConfig)
			continue
		}
		squashedConfig := new(config.Config)
		squashedPresubmitsStatic := squashPresubmitsStatic(
			baseConfig.PresubmitsStatic, headConfig.PresubmitsStatic)
		// Don't initialize a config if not needed
		if len(squashedPresubmitsStatic) == 0 {
			continue
		}
		squashedConfig.PresubmitsStatic = squashedPresubmitsStatic
		configs = append(configs, squashedConfig)
	}
	return configs
}

// Squash PresubmitsStatic config
func squashPresubmitsStatic(
	originalPresubmits, modifiedPresubmits map[string][]config.Presubmit) map[string][]config.Presubmit {
	squashedPresubmitConfigs := make(map[string][]config.Presubmit)
	for repo, headPresubmits := range modifiedPresubmits {
		basePresubmits, exists := originalPresubmits[repo]
		if !exists {
			// new presubmits
			squashedPresubmitConfigs[repo] = headPresubmits
			continue
		}
		squashedPresubmitConfigs[repo] = squashPresubmits(basePresubmits, headPresubmits)
	}
	return squashedPresubmitConfigs
}

// Given two arrays of presubmits, return a new array containing only the modified and new ones.
func squashPresubmits(originalPresubmits, modifiedPresubmits []config.Presubmit) []config.Presubmit {
	squashedPresubmits := make([]config.Presubmit, 0, len(modifiedPresubmits))
	for _, headPresubmit := range modifiedPresubmits {
		presubmitIsNew := true
		for _, basePresubmit := range originalPresubmits {
			if basePresubmit.Name != headPresubmit.Name {
				continue
			}
			presubmitIsNew = false
			if reflect.DeepEqual(headPresubmit.Spec, basePresubmit.Spec) {
				continue
			}
			squashedPresubmits = append(squashedPresubmits, headPresubmit)
		}
		if presubmitIsNew {
			squashedPresubmits = append(squashedPresubmits, headPresubmit)
		}
	}
	return squashedPresubmits
}

func loadConfigs(root, prowConfPath string, jobConfPaths []string) jobConfigPath {
	configPaths := make(jobConfigPath)
	for _, jobConfPath := range jobConfPaths {
		prowConfInRepo := filepath.Join(root, prowConfPath)
		jobConfInRepo := filepath.Join(root, jobConfPath)
		conf, err := config.Load(prowConfInRepo, jobConfInRepo)
		if err != nil {
			logrus.WithError(err).Warnf("could not load config %s", jobConfInRepo)
			continue
		}
		configPaths[jobConfPath] = conf
	}
	return configPaths
}

func writeJobs(jobs []prowapi.ProwJob) {
	for _, job := range jobs {
		y, _ := yaml.Marshal(&job)
		filename := fmt.Sprintf("/tmp/%s.yaml", job.GetName())
		err := ioutil.WriteFile(filename, y, 0644)
		if err != nil {
			logrus.Errorln(err.Error())
		}
	}
}

// Generate the ProwJobs from the configs and the PR event.
func generateProwJobs(
	configs []*config.Config, pre *github.PullRequestEvent) []prowapi.ProwJob {
	var jobs []prowapi.ProwJob

	logrus.Infoln("Will process jobs from ", len(configs), "configs")
	for _, conf := range configs {
		jobs = append(jobs, generatePresubmits(conf, pre)...)
	}

	return jobs
}

func generatePresubmits(conf *config.Config, pre *github.PullRequestEvent) []prowapi.ProwJob {
	var jobs []prowapi.ProwJob

	for repo, presubmits := range conf.PresubmitsStatic {
		for _, presubmit := range presubmits {
			pj := pjutil.NewPresubmit(
				pre.PullRequest, pre.PullRequest.Base.SHA, presubmit, pre.GUID)
			addRepoRef(&pj, repo)
			logrus.Infof("Adding job: %s", pj.Name)
			jobs = append(jobs, pj)
		}
	}

	return jobs
}

// Add ref of the original repo which we want to check.
func addRepoRef(prowJob *prowapi.ProwJob, repo string) {
	// pj.Spec.Refs[0] is being notified by reportlib.
	for _, ref := range prowJob.Spec.ExtraRefs {
		if ref.WorkDir {
			// A work dir is already configured, so we don't need to
			// clone anything to fulfill the assumption that the source
			// is there.
			return
		}
	}
	repoSplit := strings.Split(repo, "/")
	org, name := repoSplit[0], repoSplit[1]
	ref := prowapi.Refs{
		Org:        org,
		Repo:       name,
		RepoLink:   fmt.Sprintf("https://github.com/%s", repo),
		BaseRef:    "refs/heads/master",
		WorkDir:    true,
		CloneDepth: 50,
	}
	prowJob.Spec.ExtraRefs = append(prowJob.Spec.ExtraRefs, ref)
}

func logOnPanic() {
	if r := recover(); r != nil {
		logrus.Errorln(r)
	}
}
