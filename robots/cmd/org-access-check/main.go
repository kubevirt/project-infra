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
 * Copyright the KubeVirt Authors.
 *
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"net/http"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
)

const robotName = "org-access-checker"

type options struct {
	tokenPath string
	endpoint  string

	org string
}

type Permissions map[string]bool
type RepositoriesToPermissions map[string]Permissions
type CollaboratorsWithAdminRights map[string]RepositoriesToPermissions

func (o *options) Validate() error {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&o.tokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	fs.StringVar(&o.endpoint, "github-endpoint", "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
	fs.StringVar(&o.org, "org", "kubevirt", "The GitHub org")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		return err
	}
	if o.tokenPath == "" {
		return fmt.Errorf("github-token-path is required")
	}
	if o.org == "" {
		return fmt.Errorf("org is required")
	}
	return nil
}

var o = options{}

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
}
func main() {
	err := o.Validate()
	if err != nil {
		log().WithError(err).Fatal("failed to validate options")
	}

	rawToken, err := os.ReadFile(o.tokenPath)
	if err != nil {
		log().WithError(err).Fatalf("failed to read token %q", o.tokenPath)
	}
	token := strings.TrimSpace(string(rawToken))
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	githubClient, err := github.NewEnterpriseClient(o.endpoint, o.endpoint, oauth2.NewClient(ctx, ts))
	if err != nil {
		log().WithError(err).Fatal("failed to create github client")
	}

	admins, r, err := githubClient.Organizations.ListMembers(ctx, o.org, &github.ListMembersOptions{
		Role: "admin",
	})
	githubOrgAdmins := make(map[string]struct{})
	switch r.StatusCode {
	case http.StatusOK:
		for _, admin := range admins {
			githubOrgAdmins[admin.GetLogin()] = struct{}{}
		}
		break
	default:
		log().WithError(err).Fatalf("failed to get github repositories for org %q", o.org)
	}

	collaboratorIsAdminOnRepos := CollaboratorsWithAdminRights{}

	repos, r, err := githubClient.Repositories.ListByOrg(ctx, o.org, &github.RepositoryListByOrgOptions{
		Type: "public",
		ListOptions: github.ListOptions{
			PerPage: 99999,
		},
	})
	switch r.StatusCode {
	case http.StatusOK:
		log().Infof("%d repositories found", len(repos))
		for _, repo := range repos {
			log().Infof("repository %s", repo.GetName())
			collaborators, r, err := githubClient.Repositories.ListCollaborators(ctx, o.org, repo.GetName(), &github.ListCollaboratorsOptions{
				Affiliation: "all",
				ListOptions: github.ListOptions{},
			})
			if err != nil {
				log().WithError(err).Fatalf("failed to get github collaborators for repo %q", repo.GetName())
			}
			switch r.StatusCode {
			case http.StatusOK:
				for _, collaborator := range collaborators {
					if _, isAdmin := githubOrgAdmins[collaborator.GetLogin()]; isAdmin {
						continue
					}
					for _, perm := range []string{"admin", "maintain", "push"} {
						if collaborator.GetPermissions()[perm] {
							if _, ok := collaboratorIsAdminOnRepos[collaborator.GetLogin()]; !ok {
								collaboratorIsAdminOnRepos[collaborator.GetLogin()] = RepositoriesToPermissions{}
							}
							collaboratorIsAdminOnRepos[collaborator.GetLogin()][repo.GetName()] = collaborator.GetPermissions()
						}
					}
				}
				break
			default:
				log().WithError(err).Fatalf("failed to get github collaborators for repo %q", repo.GetName())
			}
		}
	default:
		log().WithError(err).Fatalf("failed to get github repositories for org %q", o.org)
	}

	marshal, err := yaml.Marshal(collaboratorIsAdminOnRepos)
	if err != nil {
		log().WithError(err).Fatal("failed to marshall file")
	}
	temp, err := os.CreateTemp("", "org-access-*.yaml")
	err = os.WriteFile(temp.Name(), marshal, 0666)
	if err != nil {
		log().WithError(err).Fatal("failed to write to file %q", temp.Name())
	}
	log().Info("File written to: %s", temp.Name())
}

func log() *logrus.Entry {
	return logrus.StandardLogger().WithField("robot", robotName)
}
