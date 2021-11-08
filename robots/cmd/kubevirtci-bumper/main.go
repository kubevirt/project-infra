/*
Copyright 2020 The KubeVirt Authors.
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
	"io/ioutil"
	"os"
	"strconv"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"kubevirt.io/project-infra/robots/pkg/kubevirtci"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

type options struct {
	port int

	dryRun bool

	webhookSecretFile      string
	mirrorRegex            string
	TokenPath              string
	endpoint               string
	ensureLatest           bool
	ensureLatestThreeMinor string
	ensureOnlyLatestThree  bool
	major                  int
	providerDir            string
	clusterUpDir           string
}

func (o *options) Validate() error {
	tasks := 0
	if o.ensureLatest {
		tasks++
	}
	if o.ensureLatestThreeMinor != "" {
		tasks++
		if !querier.SemVerMajorRegex.MatchString(o.ensureLatestThreeMinor) {
			return fmt.Errorf("Invalid format given to -latest-three-minor-of")
		}
		semver := querier.SemVerMajorRegex.FindStringSubmatch(o.ensureLatestThreeMinor)
		o.major, _ = strconv.Atoi(semver[1])
	}
	if o.ensureOnlyLatestThree {
		o.major = 1
		tasks++
	}

	if tasks == 0 {
		return fmt.Errorf("Either -ensure-latest or -ensure-last-three-minor-of must be specified.")
	} else if tasks > 1 {
		return fmt.Errorf("only one of -ensure-latest, -ensure-last-three-minor-of or -ensure-only-latest-three can be specified at the same time")
	}
	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&o.TokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	fs.StringVar(&o.endpoint, "github-endpoint", "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
	fs.BoolVar(&o.ensureLatest, "ensure-latest", false, "Ensure that we have a provider for the latest k8s release")
	fs.StringVar(&o.ensureLatestThreeMinor, "ensure-last-three-minor-of", "", "Ensure that the last three minor releases of the given major release are up to date (e.g. v1 or 2)")
	fs.BoolVar(&o.ensureOnlyLatestThree, "ensure-only-latest-three", false, "Ensure that only the latest three minor releases of the given major release exist (aka remove older providers)")
	fs.StringVar(&o.providerDir, "k8s-provider-dir", "", "The directory of the k8s providers")
	fs.StringVar(&o.clusterUpDir, "cluster-up-dir", "", "The directory of the cluster up configurations")
	fs.Parse(os.Args[1:])
	return o
}

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("robot", "kubevirtci-bumper")

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

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		log.Panicln(err)
	}
	releases = querier.ValidReleases(releases)

	_, err = os.Stat(o.providerDir)
	if os.IsNotExist(err) {
		log.Errorf("Directory '%s' does not exist", o.providerDir)
		os.Exit(1)
	} else if err != nil {
		log.WithError(err).Errorf("Failed to check directory '%s'", o.providerDir)
	}

	if len(releases) == 0 {
		log.Info("No release found.")
		os.Exit(0)
	}

	if o.ensureLatest {
		err := kubevirtci.EnsureProviderExists(o.providerDir, o.clusterUpDir, releases[0])
		if err != nil {
			log.WithError(err).Info("Failed to ensure that a provider for the given release exists.")
		}
	} else if o.ensureLatestThreeMinor != "" {
		minors := querier.LastThreeMinor(uint(o.major), releases)
		err := kubevirtci.BumpMinorReleaseOfProvider(o.providerDir, minors)
		if err != nil {
			log.WithError(err).Info("Failed to update the providers for the last minor releases.")
		}
	} else if o.ensureOnlyLatestThree {
		minors := querier.LastThreeMinor(uint(o.major), releases)
		err := kubevirtci.DropUnsupportedProviders(o.providerDir, o.clusterUpDir, minors)
		if err != nil {
			log.WithError(err).Info("Failed to remove unsupported providers.")
		}
	}
}
