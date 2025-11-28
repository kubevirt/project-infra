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
	"sort"
	"strconv"
	"strings"

	"github.com/lnquy/cron"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/pkg/kubevirt/cmd/flags"
	"kubevirt.io/project-infra/pkg/kubevirt/log"
	"sigs.k8s.io/prow/pkg/config"
)

const getPeriodicsShortDescription = "kubevirt get periodics describes periodic job definitions in project-infra for kubevirt/kubevirt repo"

var getPeriodicsCommand = &cobra.Command{
	Use:   "periodics",
	Short: "kubevirt get periodics describes periodic job definitions in project-infra for kubevirt/kubevirt repo",
	Long: getPeriodicsShortDescription + `

It reads the job configurations for kubevirt/kubevirt e2e periodic jobs, extracts information and creates a table in 
html format, so that we can quickly see which job is running how often and when. Also the table is sorted in running 
order, where jobs that run more often are ranked higher in the list. 

The table can be filtered by job name.
`,
	RunE: GetPeriodics,
}

func GetPeriodicsCommand() *cobra.Command {
	return getPeriodicsCommand
}

type getPeriodicJobsOptions struct {
	jobConfigPathKubevirtPeriodics string
	outputFile                     string
	outputFormat                   string
}

func (o getPeriodicJobsOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPeriodics); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPeriodics is required: %v", err)
	}
	return nil
}

var getPeriodicJobsOpts = getPeriodicJobsOptions{}

func init() {
	getPeriodicsCommand.PersistentFlags().StringVar(&getPeriodicJobsOpts.jobConfigPathKubevirtPeriodics, "job-config-path-kubevirt-periodics", "", "The path to the kubevirt periodic job definitions")
	getPeriodicsCommand.PersistentFlags().StringVar(&getPeriodicJobsOpts.outputFile, "output-file", "", "The file to write the output to, if empty, a temp file will be generated. If file exits, it will be overwritten")
	getPeriodicsCommand.PersistentFlags().StringVar(&getPeriodicJobsOpts.outputFormat, "output-format", "html", "The output format of the file (html or csv)")
}

type PeriodicsData struct {
	Periodics        []config.Periodic
	CronDescriptions map[string]string
}

func NewPeriodicsData() PeriodicsData {
	data := PeriodicsData{}
	data.CronDescriptions = map[string]string{}
	return data
}

func (d PeriodicsData) Len() int {
	return len(d.Periodics)
}
func (d PeriodicsData) Less(i, k int) bool {
	cronPartsI, cronPartsK := strings.Split(d.Periodics[i].Cron, " "), strings.Split(d.Periodics[k].Cron, " ")
	if len(cronPartsI) < 4 {
		return false
	}
	if len(cronPartsK) < 4 {
		return true
	}
	for cronPartsIndex := 3; cronPartsIndex >= 0; cronPartsIndex-- {
		iIsSmaller, kIsSmaller := isCronPartSmallerOrPanic(cronPartsI[cronPartsIndex], cronPartsK[cronPartsIndex])
		if iIsSmaller {
			return true
		}
		if kIsSmaller {
			return false
		}
	}
	iIsSmaller, kIsSmaller := isCronPartSmallerOrPanic(cronPartsI[4], cronPartsK[4])
	if iIsSmaller {
		return true
	}
	if kIsSmaller {
		return false
	}
	return d.Periodics[i].Name < d.Periodics[k].Name
}

func isCronPartSmallerOrPanic(cronPartI string, cronPartK string) (iIsSmaller, kIsSmaller bool) {
	return isSmallerOrPanic(getCronPartSegmentsLowestValueOrPanic(cronPartI), getCronPartSegmentsLowestValueOrPanic(cronPartK))
}

func isSmallerOrPanic(i, k string) (iIsSmaller, kIsSmaller bool) {
	if i == "*" && k == "*" {
		return false, false
	}
	if i != "*" && k == "*" {
		return false, true
	}
	if i == "*" && k != "*" {
		return true, false
	}
	iValue, err := strconv.Atoi(i)
	if err != nil {
		panic(fmt.Errorf("unexpected value for cronpart: %s", i))
	}
	kValue, err := strconv.Atoi(k)
	if err != nil {
		panic(fmt.Errorf("unexpected value for cronpart: %s", k))
	}
	return iValue < kValue, kValue < iValue
}

func getCronPartSegmentsLowestValueOrPanic(cronSegment string) string {
	segments := strings.Split(cronSegment, ",")
	lowestValue := 32
	for _, segment := range segments {
		if segment == "*" {
			return "*"
		}
		parts := strings.Split(segment, "-")
		if len(parts) > 0 {
			if parts[0] == "" {
				continue
			}
			first, err := strconv.Atoi(parts[0])
			if err != nil {
				panic(err)
			}
			if first < lowestValue {
				lowestValue = first
			}
			if len(parts) > 1 {
				second, err := strconv.Atoi(parts[1])
				if err != nil {
					panic(err)
				}
				if second < lowestValue {
					lowestValue = second
				}
			}
		}
	}
	return strconv.Itoa(lowestValue)
}

func (d PeriodicsData) Swap(i, k int) {
	d.Periodics[i], d.Periodics[k] = d.Periodics[k], d.Periodics[i]
}

//go:embed periodics.gohtml
var periodicsHTMLTemplate string

//go:embed periodics.gocsv
var periodicsCSVTemplate string

func GetPeriodics(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, getPeriodicJobsOpts)
	if err != nil {
		return err
	}

	descriptor, err := cron.NewDescriptor(cron.Use24HourTimeFormat(true))
	if err != nil {
		return fmt.Errorf("creating descriptor failed: %v", err)
	}

	data := NewPeriodicsData()

	periodicsJobConfig, err := config.ReadJobConfig(getPeriodicJobsOpts.jobConfigPathKubevirtPeriodics)
	if err != nil {
		return fmt.Errorf("failed to read job config: %v", err)
	}
	for _, periodic := range periodicsJobConfig.Periodics {
		if !strings.Contains(periodic.Name, "e2e") {
			continue
		}
		description, err := descriptor.ToDescription(periodic.Cron, cron.Locale_en)
		if err != nil {
			description = "-"
		}
		data.CronDescriptions[periodic.Name] = description
		data.Periodics = append(data.Periodics, periodic)
	}

	sort.Sort(data)

	var periodicsTemplate *template.Template
	switch getPeriodicJobsOpts.outputFormat {
	case "html":
		periodicsTemplate, err = template.New("periodics").Parse(periodicsHTMLTemplate)
	case "csv":
		periodicsTemplate, err = template.New("periodics").Parse(periodicsCSVTemplate)
	default:
		return fmt.Errorf("invalid output format %s", getPeriodicJobsOpts.outputFormat)
	}
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	buffer := bytes.NewBuffer([]byte{})
	err = periodicsTemplate.Execute(buffer, data)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	outputFile := getPeriodicJobsOpts.outputFile
	if outputFile == "" {
		tempFile, err := os.CreateTemp("", fmt.Sprintf("periodics-*.%s", getPeriodicJobsOpts.outputFormat))
		if err != nil {
			return fmt.Errorf("failed to parse template: %v", err)
		}
		outputFile = tempFile.Name()
	}

	log.Log().Infof("Writing output to %s", outputFile)
	return os.WriteFile(outputFile, buffer.Bytes(), 0666)
}
