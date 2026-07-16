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
	"os"
	"strings"

	"kubevirt.io/project-infra/pkg/kubevirt/release"
	"kubevirt.io/project-infra/pkg/querier"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/yaml"
)

const OrgAndRepoForJobConfig = "kubevirt/kubevirtci"

var knownArchs = map[string]struct{}{
	"s390x": {},
}

type options struct {
	dryRun bool

	TokenPath                        string
	endpoint                         string
	jobConfigPathKubevirtciPresubmit string
	extraArchs                       string
}

func (o *options) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtciPresubmit); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtciPresubmit is required: %v", err)
	}
	for _, arch := range parseExtraArchs(o.extraArchs) {
		if _, ok := knownArchs[arch]; !ok {
			return fmt.Errorf("unknown architecture %q in extra-archs", arch)
		}
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
	fs.StringVar(&o.extraArchs, "extra-archs", "", "Comma-separated list of extra architectures whose presubmit jobs should also be removed (e.g. s390x)")
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
		token, err := os.ReadFile(o.TokenPath)
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
	extraArchs := parseExtraArchs(o.extraArchs)
	updated := deletePresubmitJobForRelease(&jobConfig, targetRelease, extraArchs)
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

	err = os.WriteFile(o.jobConfigPathKubevirtciPresubmit, marshalledConfig, os.ModePerm)
	if err != nil {
		log.WithError(err).Error("Failed to write jobconfig")
	}

}

func deletePresubmitJobForRelease(jobConfig *config.JobConfig, targetReleaseSemver *querier.SemVer, extraArchs []string) (updated bool) {
	toDeleteJobNames := map[string]struct{}{}
	toDeleteJobNames[createKubevirtciPresubmitJobName(targetReleaseSemver)] = struct{}{}
	for _, arch := range extraArchs {
		toDeleteJobNames[createKubevirtciPresubmitJobNameArch(targetReleaseSemver, arch)] = struct{}{}
	}

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

func parseExtraArchs(raw string) []string {
	if raw == "" {
		return nil
	}
	var archs []string
	seen := make(map[string]struct{})
	for _, a := range strings.Split(raw, ",") {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		archs = append(archs, a)
	}
	return archs
}

func createKubevirtciPresubmitJobName(latestReleaseSemver *querier.SemVer) string {
	return fmt.Sprintf("check-provision-k8s-%s.%s", latestReleaseSemver.Major, latestReleaseSemver.Minor)
}

func createKubevirtciPresubmitJobNameArch(semver *querier.SemVer, arch string) string {
	return fmt.Sprintf("check-provision-k8s-%s.%s-%s", semver.Major, semver.Minor, arch)
}
