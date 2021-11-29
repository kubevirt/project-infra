/*
Copyright 2021 The KubeVirt Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"io/ioutil"
	"k8s.io/test-infra/prow/config"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/release"
	"os"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/querier"
)

const OrgAndRepoForJobConfig = "kubevirt/kubevirtci"

type options struct {
	port int

	dryRun bool

	TokenPath                        string
	endpoint                         string
	jobConfigPathKubevirtciPresubmit string
}

func (o *options) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtciPresubmit); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtciPresubmit is required: %v", err)
	}
	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&o.dryRun, "dry-run", true, "Whether the file should get modified or just modifications printed to stdout.")
	fs.StringVar(&o.TokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	fs.StringVar(&o.endpoint, "github-endpoint", "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
	fs.StringVar(&o.jobConfigPathKubevirtciPresubmit, "job-config-path-kubevirtci-presubmit", "", "The directory of the k8s providers")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Println(fmt.Errorf("failed to parse args: %v", err))
		os.Exit(1)
	}
	return o
}

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("robot", "kubevirtci-presubmit-remover")

	o := gatherOptions()
	if err := o.Validate(); err != nil {
		log.WithError(err).Error("Invalid arguments provided.")
		os.Exit(1)
	}

	ctx := context.Background()
	var client *github.Client
	if o.TokenPath == "" {
		var err error
		client, err = github.NewEnterpriseClient(o.endpoint, o.endpoint, nil)
		if err != nil {
			log.Panicln(err)
		}
	} else {
		token, err := ioutil.ReadFile(o.TokenPath)
		if err != nil {
			log.Panicln(err)
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: string(token)},
		)
		client, err = github.NewEnterpriseClient(o.endpoint, o.endpoint, oauth2.NewClient(ctx, ts))
		if err != nil {
			log.Panicln(err)
		}
	}

	jobConfig, err := config.ReadJobConfig(o.jobConfigPathKubevirtciPresubmit)
	if err != nil {
		log.Panicln(err)
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		log.Panicln(err)
	}
	releases = querier.ValidReleases(releases)
	if len(releases) == 0 {
		log.Info("No release found, nothing to do.")
		os.Exit(0)
	}

	latestMinorReleases := release.GetLatestMinorReleases(release.AsSemVers(releases))
	if len(latestMinorReleases) < 3 {
		log.Info("Not enough minor releases found, nothing to do.")
	}

	targetRelease := latestMinorReleases[3]
	updated := deletePresubmitJobForRelease(&jobConfig, targetRelease)
	if !updated {
		log.Info("Not updated, nothing to do")
		os.Exit(0)
	}

	marshalledConfig, err := yaml.Marshal(jobConfig)
	if err != nil {
		log.WithError(err).Error("Failed to marshall jobconfig")
	}

	if o.dryRun {
		_, err = os.Stdout.Write(marshalledConfig)
		if err != nil {
			log.WithError(err).Error("Failed to write jobconfig")
		}
		os.Exit(0)
	}

	err = ioutil.WriteFile(o.jobConfigPathKubevirtciPresubmit, marshalledConfig, os.ModePerm)
	if err != nil {
		log.WithError(err).Error("Failed to write jobconfig")
	}

}

func deletePresubmitJobForRelease(jobConfig *config.JobConfig, targetReleaseSemver *querier.SemVer) (updated bool) {
	toDeleteJobNames := map[string]struct{}{}
	toDeleteJobNames[createKubevirtciPresubmitJobName(targetReleaseSemver)] = struct{}{}

	var newPresubmits []config.Presubmit

	for _, presubmit := range jobConfig.PresubmitsStatic[OrgAndRepoForJobConfig] {
		if _, exists := toDeleteJobNames[presubmit.Name]; exists {
			updated = true
			continue
		}
		newPresubmits = append(newPresubmits, presubmit)
	}

	if updated {
		jobConfig.PresubmitsStatic[OrgAndRepoForJobConfig] = newPresubmits
	}

	return
}

func createKubevirtciPresubmitJobName(latestReleaseSemver *querier.SemVer) string {
	return fmt.Sprintf("check-provision-k8s-%s.%s", latestReleaseSemver.Major, latestReleaseSemver.Minor)
}
