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
 * Copyright The KubeVirt Authors.
 * Copyright 2017 The Kubernetes Authors.
 *
 */

// Derived from kubernetes/test-infra robots/commenter, since we need to improve
// on filtering commit status:failure to only comment on required test lanes
// failing

package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/prow/pkg/config"

	"sigs.k8s.io/prow/pkg/config/secret"
	"sigs.k8s.io/prow/pkg/flagutil"
	"sigs.k8s.io/prow/pkg/github"
)

const (
	baseQuery = `
		is:pr
		archived:false
		is:open
		is:unlocked
		-label:do-not-merge
		-label:do-not-merge/blocked-paths
		-label:do-not-merge/cherry-pick-not-approved
		-label:do-not-merge/hold
		-label:do-not-merge/invalid-owners-file
		-label:do-not-merge/release-note-label-needed
		-label:do-not-merge/work-in-progress
		-label:needs-rebase
		%s
		status:failure
		%s
`
)

var (
	repos = []string{
		"kubevirt",
		"kubevirtci",
		"project-infra",
	}

	labelSets = []string{
		"label:lgtm label:approved",
		"label:skip-review",
	}

	fullQueries []string

	comment = `/retest-required
This bot automatically retries required jobs that failed/flaked on 
required test lanes of PRs.
Silence the bot with an ` + "`" + `/lgtm cancel` + "`" + ` or ` + "`" + `/hold` + "`" + ` comment for consistent failures.`

	jobContextMatcher = regexp.MustCompile(`.*/([^/]+)/[0-9]+$`)

	presubmitRequiredMap = map[string]struct{}{}
)

func init() {
	var repoQueries []string
	for _, r := range repos {
		repoQueries = append(repoQueries, fmt.Sprintf("repo:kubevirt/%s", r))
	}
	for _, labelSet := range labelSets {
		fullQueries = append(fullQueries, fmt.Sprintf(baseQuery, labelSet, strings.Join(repoQueries, " ")))
	}
}

func flagOptions() options {
	o := options{
		endpoint: flagutil.NewStrings(github.DefaultAPIEndpoint),
	}
	flag.DurationVar(&o.updated, "updated", 2*time.Hour, "Filter to issues unmodified for at least this long if set")
	flag.BoolVar(&o.confirm, "confirm", false, "Mutate github if set")
	flag.IntVar(&o.ceiling, "ceiling", 1, "Maximum number of issues to modify, 0 for infinite")
	flag.Var(&o.endpoint, "endpoint", "GitHub's API endpoint")
	flag.StringVar(&o.graphqlEndpoint, "graphql-endpoint", github.DefaultGraphQLEndpoint, "GitHub's GraphQL API Endpoint")
	flag.StringVar(&o.token, "token", "", "Path to github token")
	flag.Parse()
	return o
}

type options struct {
	ceiling         int
	endpoint        flagutil.Strings
	graphqlEndpoint string
	token           string
	updated         time.Duration
	confirm         bool
}

type client interface {
	CreateComment(owner, repo string, number int, comment string) error
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	if o.token == "" {
		log.Fatal("empty --token")
	}

	if err := secret.Add(o.token); err != nil {
		log.Fatalf("Error starting secrets agent: %v", err)
	}

	if err := initPresubmitRequiredMap("github/ci/prow-deploy/files/jobs/kubevirt/"); err != nil {
		log.Fatalf("Error reading required presubmits: %v", err)
	}

	var err error
	for _, ep := range o.endpoint.Strings() {
		_, err = url.ParseRequestURI(ep)
		if err != nil {
			log.Fatalf("Invalid --endpoint URL %q: %v.", ep, err)
		}
	}

	var c client
	if o.confirm {
		c, err = github.NewClient(secret.GetTokenGenerator(o.token), secret.Censor, o.graphqlEndpoint, o.endpoint.Strings()...)
	} else {
		c, err = github.NewDryRunClient(secret.GetTokenGenerator(o.token), secret.Censor, o.graphqlEndpoint, o.endpoint.Strings()...)
	}
	if err != nil {
		log.Fatalf("Failed to construct GitHub client: %v", err)
	}

	for _, q := range fullQueries {
		query := makeQuery(q, o.updated)
		sort := ""
		asc := false
		if o.updated > 0 {
			sort = "updated"
			asc = true
		}
		if err := run(c, query, sort, asc, comment, o.ceiling); err != nil {
			log.Fatalf("Failed run: %v", err)
		}
	}
}

func initPresubmitRequiredMap(orgJobConfigDir string) error {
	presubmitFileNameRegexp := regexp.MustCompile(`.*-presubmits.*.yaml`)
	for _, dir := range repos {
		repoJobConfigDirName := filepath.Join(orgJobConfigDir, dir)
		repoJobConfigFiles, err := os.ReadDir(repoJobConfigDirName)
		if err != nil {
			return fmt.Errorf("error reading job dir: %v", err)
		}
		for _, file := range repoJobConfigFiles {
			if file.IsDir() {
				continue
			}
			if !presubmitFileNameRegexp.MatchString(file.Name()) {
				continue
			}
			fileName := filepath.Join(repoJobConfigDirName, file.Name())
			log.Printf("reading file %q", fileName)
			jobConfig, err := config.ReadJobConfig(fileName)
			if err != nil {
				return fmt.Errorf("error parsing kubevirt job file: %v", err)
			}
			for _, presubmits := range jobConfig.PresubmitsStatic {
				for _, presubmit := range presubmits {
					if !(presubmit.AlwaysRun || presubmit.RunBeforeMerge) ||
						presubmit.Optional ||
						presubmit.RunIfChanged != "" ||
						presubmit.SkipIfOnlyChanged != "" {
						continue
					}
					presubmitRequiredMap[presubmit.Name] = struct{}{}
				}
			}
		}
	}
	return nil
}

func makeQuery(query string, minUpdated time.Duration) string {
	// GitHub used to allow \n but changed it at some point to result in no results at all
	toReplace := regexp.MustCompile(`[\n\t ]+`)
	query = toReplace.ReplaceAllString(query, " ")
	if minUpdated != 0 {
		latest := time.Now().Add(-minUpdated)
		query += " updated:<=" + latest.Format(time.RFC3339)
	}
	return query
}

func run(c client, query, sort string, asc bool, comment string, ceiling int) error {
	log.Printf("Searching: %s", query)
	issues, err := c.FindIssues(query, sort, asc)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}
	var problems []string
	log.Printf("Found %d matches", len(issues))
	var modified int
	for _, i := range issues {
		if ceiling > 0 && modified == ceiling {
			log.Printf("Stopping at --ceiling=%d of %d results", modified, len(issues))
			break
		}
		log.Printf("Matched %s (%s)", i.HTMLURL, i.Title)
		org, repo, number, err := parseHTMLURL(i.HTMLURL)
		if err != nil {
			msg := fmt.Sprintf("Failed to parse %s: %v", i.HTMLURL, err)
			log.Print(msg)
			problems = append(problems, msg)
		}
		pullRequest, err := c.GetPullRequest(org, repo, number)
		if err != nil {
			msg := fmt.Sprintf("Failed to get pull request %s: %v", i.HTMLURL, err)
			log.Print(msg)
			problems = append(problems, msg)
		}
		combinedStatus, err := c.GetCombinedStatus(org, repo, pullRequest.Head.SHA)
		if err != nil {
			msg := fmt.Sprintf("Failed to get combined status %s: %v", pullRequest.Head.SHA, err)
			log.Print(msg)
			problems = append(problems, msg)
		}
		requiredStatusFailed := false
		for _, s := range combinedStatus.Statuses {
			if s.State != "failure" {
				continue
			}
			stringSubmatch := jobContextMatcher.FindStringSubmatch(s.TargetURL)
			if len(stringSubmatch) > 2 {
				continue
			}
			presubmitName := stringSubmatch[1]
			if _, isRequired := presubmitRequiredMap[presubmitName]; !isRequired {
				log.Printf("skipping non-required status for %s", presubmitName)
				continue
			}
			log.Printf("found required status for %s", presubmitName)
			requiredStatusFailed = true
			break
		}
		if !requiredStatusFailed {
			log.Printf("no failure on a required status detected for %s", i.HTMLURL)
			continue
		}
		if err := c.CreateComment(org, repo, number, comment); err != nil {
			msg := fmt.Sprintf("Failed to apply comment to %s/%s#%d: %v", org, repo, number, err)
			log.Print(msg)
			problems = append(problems, msg)
			continue
		}
		modified++
		log.Printf("Commented on %s", i.HTMLURL)
	}
	if len(problems) > 0 {
		return fmt.Errorf("encountered %d failures: %v", len(problems), problems)
	}
	return nil
}

func parseHTMLURL(url string) (string, string, int, error) {
	// Example: https://github.com/batterseapower/pinyin-toolkit/issues/132
	re := regexp.MustCompile(`.+/(.+)/(.+)/(issues|pull)/(\d+)$`)
	mat := re.FindStringSubmatch(url)
	if mat == nil {
		return "", "", 0, fmt.Errorf("failed to parse: %s", url)
	}
	n, err := strconv.Atoi(mat[4])
	if err != nil {
		return "", "", 0, err
	}
	return mat[1], mat[2], n, nil
}
