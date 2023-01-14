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
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/test-infra/prow/config"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
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
	jobConfigPathKubevirtPresubmits string
	outputFile                      string
}

func (o getPresubmitsJobruntimesOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPeriodics is required: %v", err)
	}
	return nil
}

var getPresubmitsJobruntimesOpts = getPresubmitsJobruntimesOptions{}

//go:embed presubmits.gohtml
var presubmitsHTMLTemplate string

func init() {
	getPresubmitsCommand.PersistentFlags().StringVar(&getPresubmitsJobruntimesOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The path to the kubevirt presubmits job definitions")
	getPresubmitsCommand.PersistentFlags().StringVar(&getPresubmitsJobruntimesOpts.outputFile, "output-file", "", "The file to write the output to, if empty, a temp file will be generated. If file exits, it will be overwritten")
}

type presubmits []config.Presubmit

func (d presubmits) Len() int {
	return len(d)
}

func (d presubmits) Less(i, j int) bool {
	// gating jobs have highest priority
	if isGating(d[i]) && !isGating(d[j]) {
		return true
	}
	if !isGating(d[i]) && isGating(d[j]) {
		return false
	}
	// `always_run: true` is next
	if d[i].AlwaysRun && !d[j].AlwaysRun {
		return true
	}
	if !d[i].AlwaysRun && d[j].AlwaysRun {
		return false
	}
	// followed by one of `RunIfChanged` or `SkipIfOnlyChanged`
	if (d[i].RunIfChanged != "" || d[i].SkipIfOnlyChanged != "") && (d[j].RunIfChanged == "" && d[j].SkipIfOnlyChanged == "") {
		return true
	}
	if (d[i].RunIfChanged == "" && d[i].SkipIfOnlyChanged == "") && (d[j].RunIfChanged != "" || d[j].SkipIfOnlyChanged != "") {
		return false
	}
	return d[i].Name < d[j].Name
}

func (d presubmits) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func isGating(row config.Presubmit) bool {
	return row.AlwaysRun && !row.Optional
}

func GetPresubmits(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, getPresubmitsJobruntimesOpts)
	if err != nil {
		return err
	}

	presubmitJobConfig, err := config.ReadJobConfig(getPresubmitsJobruntimesOpts.jobConfigPathKubevirtPresubmits)
	if err != nil {
		return fmt.Errorf("failed to read jobconfig %s: %v", getPresubmitsJobruntimesOpts.jobConfigPathKubevirtPresubmits, err)
	}

	presubmitsTemplate, err := template.New("presubmits").Parse(presubmitsHTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	var e2ePresubmits presubmits
	for _, presubmit := range presubmitJobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		if !strings.Contains(presubmit.Name, "e2e") {
			continue
		}
		e2ePresubmits = append(e2ePresubmits, presubmit)
	}
	sort.Sort(e2ePresubmits)

	buffer := bytes.NewBuffer([]byte{})
	err = presubmitsTemplate.Execute(buffer, e2ePresubmits)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	outputFile := getPresubmitsJobruntimesOpts.outputFile
	if outputFile == "" {
		tempFile, err := os.CreateTemp("", "presubmits-*.html")
		if err != nil {
			return fmt.Errorf("failed to parse template: %v", err)
		}
		outputFile = tempFile.Name()
	}

	log.Log().Infof("Writing output to %s", outputFile)
	return os.WriteFile(outputFile, buffer.Bytes(), 0666)
}
