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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package get

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"kubevirt.io/project-infra/pkg/kubevirt/cmd/flags"
	"kubevirt.io/project-infra/pkg/kubevirt/log"
	"kubevirt.io/project-infra/pkg/kubevirt/prowjobconfigs"

	"github.com/spf13/cobra"
	"sigs.k8s.io/prow/pkg/config"
)

const (
	shortUsage = "kubevirt get presubmits describes presubmit job definitions in project-infra for kubevirt/kubevirt repo"
)

var getPresubmitsCommand = &cobra.Command{
	Use:   "presubmits",
	Short: shortUsage,
	Long: shortUsage + `

It reads the job configurations for kubevirt/kubevirt e2e presubmit jobs, extracts information and creates a table in html
format, so that we can quickly see which job is gating the merge and which job is running on every kubevirt/kubevirt PR.

The table is sorted in order gating -> always_run -> conditional_runs -> others and can be filtered by job name.
`,
	RunE: GetPresubmits,
}

func GetPresubmitsCommand() *cobra.Command {
	return getPresubmitsCommand
}

type getPresubmitsJobruntimesOptions struct {
	jobConfigPathKubevirtPresubmits []string
	outputFile                      string
	outputFormat                    string
}

func (o getPresubmitsJobruntimesOptions) Validate() error {
	if len(o.jobConfigPathKubevirtPresubmits) == 0 {
		return fmt.Errorf("no job config pathes for presubmits given")
	}
	for _, file := range o.jobConfigPathKubevirtPresubmits {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
		}
	}
	return nil
}

var getPresubmitsJobruntimesOpts = getPresubmitsJobruntimesOptions{}

//go:embed presubmits.gohtml
var presubmitsHTMLTemplate string

//go:embed presubmits.gocsv
var presubmitsCSVTemplate string

func init() {
	getPresubmitsCommand.PersistentFlags().StringArrayVar(&getPresubmitsJobruntimesOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", nil, "The path to the kubevirt presubmits job definitions")
	getPresubmitsCommand.PersistentFlags().StringVar(&getPresubmitsJobruntimesOpts.outputFile, "output-file", "", "The file to write the output to, if empty, a temp file will be generated. If file exits, it will be overwritten")
	getPresubmitsCommand.PersistentFlags().StringVar(&getPresubmitsJobruntimesOpts.outputFormat, "output-format", "html", "The format of the output file (html or csv)")
}

type presubmits []config.Presubmit

func (d presubmits) Len() int {
	return len(d)
}

var k8sVersionMatcher = regexp.MustCompile(`(kind|k8s)-([0-9]+\.[0-9]+)`)

func (d presubmits) Less(i, j int) bool {
	if isGating(d[i]) != isGating(d[j]) {
		return isGating(d[i])
	}
	if d[i].AlwaysRun != d[j].AlwaysRun {
		return d[i].AlwaysRun
	}
	if d[i].RunBeforeMerge != d[j].RunBeforeMerge {
		return d[i].RunBeforeMerge
	}
	if (d[i].RunIfChanged != "") != (d[j].RunIfChanged != "") {
		return d[i].RunIfChanged != ""
	}
	if (d[i].SkipIfOnlyChanged != "") != (d[j].SkipIfOnlyChanged != "") {
		return d[i].SkipIfOnlyChanged != ""
	}

	// add sorting by k8s version if present
	if k8sVersionMatcher.MatchString(d[i].Name) && k8sVersionMatcher.MatchString(d[j].Name) {
		// extract k8s versions
		iK8sVersionMatch := k8sVersionMatcher.FindAllStringSubmatch(d[i].Name, 2)
		iVersion, err := semver.NewVersion("v" + iK8sVersionMatch[0][2])
		if err != nil {
			panic(err)
		}
		jK8sVersionMatch := k8sVersionMatcher.FindAllStringSubmatch(d[j].Name, 2)
		jVersion, err := semver.NewVersion("v" + jK8sVersionMatch[0][2])
		if err != nil {
			panic(err)
		}
		if iVersion.GreaterThan(jVersion) != jVersion.GreaterThan(iVersion) {
			return iVersion.GreaterThan(jVersion)
		}
	}

	return strings.Compare(d[i].Name, d[j].Name) < 0
}

func (d presubmits) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func isGating(row config.Presubmit) bool {
	return !row.Optional && (row.AlwaysRun || row.RunBeforeMerge)
}

func GetPresubmits(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, getPresubmitsJobruntimesOpts)
	if err != nil {
		return err
	}

	var e2ePresubmits presubmits
	for _, kubevirtPresubmit := range getPresubmitsJobruntimesOpts.jobConfigPathKubevirtPresubmits {
		presubmitJobConfig, err := config.ReadJobConfig(kubevirtPresubmit)
		if err != nil {
			return fmt.Errorf("failed to read jobconfig %s: %v", getPresubmitsJobruntimesOpts.jobConfigPathKubevirtPresubmits, err)
		}

		for _, presubmit := range presubmitJobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
			if !strings.Contains(presubmit.Name, "e2e") {
				continue
			}
			e2ePresubmits = append(e2ePresubmits, presubmit)
		}
	}
	sort.Sort(e2ePresubmits)

	var presubmitsTemplate *template.Template
	switch getPresubmitsJobruntimesOpts.outputFormat {
	case "html":
		presubmitsTemplate, err = template.New("presubmits").Parse(presubmitsHTMLTemplate)
	case "csv":
		presubmitsTemplate, err = template.New("presubmits").Parse(presubmitsCSVTemplate)
	default:
		return fmt.Errorf("invalid output format %s", getPresubmitsJobruntimesOpts.outputFormat)
	}
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	buffer := bytes.NewBuffer([]byte{})
	err = presubmitsTemplate.Execute(buffer, e2ePresubmits)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	outputFile := getPresubmitsJobruntimesOpts.outputFile
	if outputFile == "" {
		tempFile, err := os.CreateTemp("", fmt.Sprintf("presubmits-*.%s", getPresubmitsJobruntimesOpts.outputFormat))
		if err != nil {
			return fmt.Errorf("failed to parse template: %v", err)
		}
		outputFile = tempFile.Name()
	}

	log.Log().Infof("Writing output to %s", outputFile)
	err = os.WriteFile(outputFile, buffer.Bytes(), 0666)
	if err != nil {
		return fmt.Errorf("failed to write output file: %v", err)
	}

	return nil
}
