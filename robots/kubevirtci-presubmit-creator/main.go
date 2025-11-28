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
	"time"

	"kubevirt.io/project-infra/pkg/querier"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	prowjobs "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/yaml"
)

const OrgAndRepoForJobConfig = "kubevirt/kubevirtci"

type options struct {
	dryRun bool

	TokenPath                        string
	endpoint                         string
	jobConfigPathKubevirtciPresubmit string
	k8sReleaseSemver                 string
}

func (o *options) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtciPresubmit); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtciPresubmit is required: %v", err)
	}
	if o.k8sReleaseSemver != "" && !querier.SemVerMinorRegex.MatchString(o.k8sReleaseSemver) {
		return fmt.Errorf("k8s-release-semver does not match SemVerMinorRegex: %s", o.k8sReleaseSemver)
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
	fs.StringVar(&o.k8sReleaseSemver, "k8s-release-semver", "", "The semver of the k8s release to create a presubmit for")
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
	log := logrus.StandardLogger().WithField("robot", "kubevirtci-presubmit-creator")

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

	var latestReleaseSemver *querier.SemVer

	if o.k8sReleaseSemver == "" {
		releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
		if err != nil {
			log.Panicln(err)
		}
		releases = querier.ValidReleases(releases)
		if len(releases) == 0 {
			log.Info("No release found, nothing to do.")
			os.Exit(0)
		}
		latestReleaseSemver = querier.ParseRelease(releases[0])
	} else {
		majorMinor := querier.SemVerMinorRegex.FindStringSubmatch(o.k8sReleaseSemver)
		latestReleaseSemver = &querier.SemVer{
			Major: majorMinor[1],
			Minor: majorMinor[2],
			Patch: "0",
		}
	}

	jobConfig, err := config.ReadJobConfig(o.jobConfigPathKubevirtciPresubmit)
	if err != nil {
		log.Panicln(err)
	}

	newJobConfig, exists := AddNewPresubmitIfNotExists(jobConfig, latestReleaseSemver)
	if exists && !o.dryRun {
		log.Info(fmt.Sprintf("presubmit job for %v exists, nothing to do.", latestReleaseSemver))
		os.Exit(0)
	}

	marshalledConfig, err := yaml.Marshal(newJobConfig)
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

func AddNewPresubmitIfNotExists(jobConfig config.JobConfig, latestReleaseSemver *querier.SemVer) (newJobConfig config.JobConfig, jobExists bool) {
	newJobConfig = jobConfig
	kubevirtciJobs := make(map[string]config.Presubmit, len(newJobConfig.PresubmitsStatic[OrgAndRepoForJobConfig]))
	for _, job := range newJobConfig.PresubmitsStatic[OrgAndRepoForJobConfig] {
		kubevirtciJobs[job.Name] = job
	}

	wantedCheckProvisionJobName := createKubevirtciPresubmitJobName(latestReleaseSemver)
	if _, exists := kubevirtciJobs[wantedCheckProvisionJobName]; exists {
		return newJobConfig, true
	}

	newPresubmitJobForRelease := CreatePresubmitJobForRelease(latestReleaseSemver)
	newJobConfig.PresubmitsStatic[OrgAndRepoForJobConfig] = append(newJobConfig.PresubmitsStatic[OrgAndRepoForJobConfig], newPresubmitJobForRelease)
	return newJobConfig, false
}

func CreatePresubmitJobForRelease(semver *querier.SemVer) config.Presubmit {
	yes := true
	golangImage := "quay.io/kubevirtci/golang:v20230801-94954c0"
	cluster := "prow-workloads"
	res := config.Presubmit{
		AlwaysRun: false,
		Optional:  true,
		JobBase: config.JobBase{
			UtilityConfig: config.UtilityConfig{
				Decorate: &yes,
				DecorationConfig: &prowjobs.DecorationConfig{
					Timeout: &prowjobs.Duration{Duration: 3 * time.Hour},
				},
			},
			Name:           fmt.Sprintf("check-provision-k8s-%s.%s", semver.Major, semver.Minor),
			MaxConcurrency: 1,
			Labels: map[string]string{
				"preset-docker-mirror-proxy":         "true",
				"preset-podman-in-container-enabled": "true",
			},
			Cluster: cluster,
			Spec: &v1.PodSpec{
				NodeSelector: map[string]string{
					"type": "bare-metal-external",
				},
				Containers: []v1.Container{
					{
						Image: golangImage,
						Command: []string{
							"/usr/local/bin/runner.sh",
							"/bin/sh",
							"-c",
							fmt.Sprintf("cd cluster-provision/k8s/%s.%s && KUBEVIRT_PSA='true' ../provision.sh", semver.Major, semver.Minor),
						},
						SecurityContext: &v1.SecurityContext{
							Privileged: &yes,
						},
						Resources: v1.ResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceMemory: *resource.NewQuantity(8*1024*1024*1024, resource.BinarySI),
							},
						},
					},
				},
			},
		},
	}
	return res
}

func createKubevirtciPresubmitJobName(latestReleaseSemver *querier.SemVer) string {
	return fmt.Sprintf("check-provision-k8s-%s.%s", latestReleaseSemver.Major, latestReleaseSemver.Minor)
}
