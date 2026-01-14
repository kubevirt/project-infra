/*
Copyright 2010 The KubeVirt Authors.
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
	"strconv"
	"text/template"

	"kubevirt.io/project-infra/pkg/querier"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type options struct {
	org              string
	repo             string
	TokenPath        string
	endpoint         string
	latest           bool
	latestPatchOf    string
	latestThreeMinor string
	template         string
	major            int
	minor            int
}

func (o *options) Validate() error {
	queries := 0
	if o.latest {
		queries++
	}
	if o.latestThreeMinor != "" {
		queries++
		if !querier.SemVerMajorRegex.MatchString(o.latestThreeMinor) {
			return fmt.Errorf("Invalid format given to -latest-three-minor-of")
		}
		semver := querier.SemVerMajorRegex.FindStringSubmatch(o.latestThreeMinor)
		o.major, _ = strconv.Atoi(semver[1])
	}
	if o.latestPatchOf != "" {
		queries++
		if !querier.SemVerMinorRegex.MatchString(o.latestPatchOf) {
			return fmt.Errorf("Invalid format given to -latest-patch-of")
		}
		semver := querier.SemVerMinorRegex.FindStringSubmatch(o.latestPatchOf)
		o.major, _ = strconv.Atoi(semver[1])
		o.minor, _ = strconv.Atoi(semver[2])
	}

	if queries == 0 {
		return fmt.Errorf("Either -latest, -last-three-minor-of or -last-patch-of must be specified.")
	} else if queries > 1 {
		return fmt.Errorf("Only one of -latest, -last-three-minor-of or -last-patch-of can be specified at the same time.")
	}
	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&o.org, "org", "kubevirt", "Organization")
	fs.StringVar(&o.repo, "repo", "kubevirt", "Organization")
	fs.StringVar(&o.TokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	fs.StringVar(&o.endpoint, "github-endpoint", "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
	fs.BoolVar(&o.latest, "latest", false, "Query for the latest release")
	fs.StringVar(&o.latestThreeMinor, "last-three-minor-of", "", "Query for the last three minor releases of a given release (e.g. v1 or 2)")
	fs.StringVar(&o.latestPatchOf, "last-patch-of", "", "Latest patch release of the given release (e.g. v1.14 or 0.12)")
	fs.StringVar(&o.template, "template", "v{{.Major}}.{{.Minor}}.{{.Patch}}", "How to print the detected versions")
	fs.Parse(os.Args[1:])
	return o
}

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)
	log := logrus.StandardLogger().WithField("robot", "release-querier")

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

	releases, _, err := client.Repositories.ListReleases(ctx, o.org, o.repo, nil)
	if err != nil {
		log.Panicln(err)
	}
	tmpl, err := template.New("test").Parse(o.template)
	if err != nil {
		log.Panicln(err)
	}

	if o.latest {
		latest := querier.LatestRelease(releases)
		if latest != nil {
			tmpl.Execute(os.Stdout, querier.ParseRelease(latest))
			fmt.Print("\n")
		}
	} else if o.latestPatchOf != "" {
		latestPatchOf := querier.LastPatchOf(uint(o.major), uint(o.minor), releases)
		if latestPatchOf != nil {
			tmpl.Execute(os.Stdout, querier.ParseRelease(latestPatchOf))
			fmt.Print("\n")
		}
	} else if o.latestThreeMinor != "" {
		minors := querier.LastThreeMinor(uint(o.major), releases)
		for _, release := range minors {
			tmpl.Execute(os.Stdout, querier.ParseRelease(release))
			fmt.Print("\n")
		}
	}
}
