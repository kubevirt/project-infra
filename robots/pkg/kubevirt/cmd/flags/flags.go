/*
 * Copyright 2021 The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package flags

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
)

const (
	FlagDryRun          = "dry-run"
	FlagGitHubTokenPath = "github-token-path"
	FlagGitHubEndpoint  = "github-endpoint"
)

type GlobalOptions struct {
	DryRun          bool
	GitHubTokenPath string
	GitHubEndPoint  string
}

type CommandOptions interface {
	Validate() error
}

var Options = GlobalOptions{}

func (o GlobalOptions) Validate() error {
	if o.GitHubTokenPath != "" {
		if fileInfo, err := os.Stat(o.GitHubTokenPath); !os.IsNotExist(err) {
			if fileInfo.IsDir() {
				return fmt.Errorf("File %s is a directory!", o.GitHubTokenPath)
			}
		} else {
			return err
		}
	}
	return nil
}

func AddPersistentFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&Options.DryRun, FlagDryRun, true, "Whether the file should get modified or just modifications printed to stdout.")
	cmd.PersistentFlags().StringVar(&Options.GitHubTokenPath, FlagGitHubTokenPath, "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	cmd.PersistentFlags().StringVar(&Options.GitHubEndPoint, FlagGitHubEndpoint, "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
}

func ParseFlagsOrExit(cmd *cobra.Command, args []string, cmdOpts CommandOptions) {
	err := cmd.InheritedFlags().Parse(args)
	if err != nil {
		fmt.Println(fmt.Errorf("failed to parse args: %v", err))
		os.Exit(1)
	}

	validateOptions(cmd, Options)
	validateOptions(cmd, cmdOpts)
}

func validateOptions(cmd *cobra.Command, cmdOpts CommandOptions) {
	if err := cmdOpts.Validate(); err != nil {
		fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())

		log.Log().WithError(err).Fatal("Invalid arguments provided.")
	}
}
