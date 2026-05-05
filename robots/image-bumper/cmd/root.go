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

package cmd

import (
	"fmt"

	"kubevirt.io/project-infra/pkg/imagebump"

	"github.com/spf13/cobra"
)

var repoRoot string

func Execute() error {
	rootCmd := &cobra.Command{
		Use:   "bump",
		Short: "Bump kubevirtci and related container image references in project-infra",
	}
	rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "path to the project-infra git checkout")

	jobImagesCmd := &cobra.Command{
		Use:   "job-images",
		Short: "Update kubevirtci images in prow job YAML under github/ci/prow-deploy/files/jobs",
		RunE: func(_ *cobra.Command, _ []string) error {
			return imagebump.BumpJobImages(repoRoot)
		},
	}
	deploymentImagesCmd := &cobra.Command{
		Use:   "prow-deployment-images",
		Short: "Update kubevirtci images in prow kustom deployment YAML",
		RunE: func(_ *cobra.Command, _ []string) error {
			return imagebump.BumpProwDeploymentImages(repoRoot)
		},
	}
	containerfileCmd := &cobra.Command{
		Use:   "containerfile-images",
		Short: "Update FROM lines using quay.io tags (tracked Containerfiles/Dockerfiles)",
		RunE: func(_ *cobra.Command, _ []string) error {
			return imagebump.BumpContainerfileImages(repoRoot)
		},
	}
	allCmd := &cobra.Command{
		Use:   "all",
		Short: "Run job-images, prow-deployment-images, and containerfile-images bumps",
		RunE: func(_ *cobra.Command, _ []string) error {
			tags, err := imagebump.ResolveKubevirtCITagMap(repoRoot)
			if err != nil {
				return fmt.Errorf("resolve kubevirtci tags: %w", err)
			}
			if err := imagebump.BumpJobImagesWithTagMap(repoRoot, tags); err != nil {
				return fmt.Errorf("job-images: %w", err)
			}
			if err := imagebump.BumpProwDeploymentImagesWithTagMap(repoRoot, tags); err != nil {
				return fmt.Errorf("prow-deployment-images: %w", err)
			}
			if err := imagebump.BumpContainerfileImages(repoRoot); err != nil {
				return fmt.Errorf("containerfile-images: %w", err)
			}
			return nil
		},
	}

	rootCmd.AddCommand(jobImagesCmd, deploymentImagesCmd, containerfileCmd, allCmd)

	return rootCmd.Execute()
}
