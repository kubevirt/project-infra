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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"cloud.google.com/go/storage"

	"kubevirt.io/project-infra/robots/pkg/flakefinder"
)

const (
	kubevirtCI  = "kubevirt"
	openshiftCI = "openshift"
	presubmits  = "presubmits"
	periodics   = "periodics"
	bucketName  = "bucketName"
	useSubDirs  = "useSubDirs"
	subDirRegex = "subDirRegex"
	jobDataPath = "jobDataPath"

	ProwReportTemplate = `
<html>
<head>
    <title>flakefinder report</title>
    <meta charset="UTF-8">
    <style>
        table, th, td {
            border: 1px solid black;
        }
        .yellow {
            background-color: #ffff80;
        }
        .almostgreen {
            background-color: #dfff80;
        }
        .green {
            background-color: #9fff80;
        }
        .red {
            background-color: #ff8080;
        }
        .orange {
            background-color: #ffbf80;
        }
        .unimportant {
        }
        .tests_passed {
            color: #226c18;
            font-weight: bold;
        }
        .tests_failed {
            color: #8a1717;
            font-weight: bold;
        }
        .tests_skipped {
            color: #535453;
            font-weight: bold;
        }
        .center {
            text-align:center
        }
        .right {
            text-align: right;
			width: 100%;
        }
	</style>
</head>
<body>
<h1>flakefinder report</h1>

<div>
	Data since {{ $.StartOfReport }}<br/>
	Bucket: {{ $.BucketName }}<br/>
	Pathes: {{ range $path := $.JobDataPathes }}<code>{{ $path }},</code>{{ end }}
</div>
<table>
    <tr>
        <td></td>
        <td></td>
        {{ range $header := $.Headers }}
        <td>{{ $header }}</td>
        {{ end }}
    </tr>
    {{ range $row, $test := $.Tests }}
    <tr>
        <td><div id="row{{$row}}"><a href="#row{{$row}}">{{ $row }}</a><div></td>
        <td>{{ $test }}</td>
        {{ range $col, $header := $.Headers }}
        {{if not (index $.Data $test $header) }}
        <td class="center">
            N/A
        </td>
        {{else}}
        <td class="{{ (index $.Data $test $header).Severity }} center">
            <div id="r{{$row}}c{{$col}}">
                <span class="tests_failed" title="failed tests">{{ (index $.Data $test $header).Failed }}</span>/<span class="tests_passed" title="passed tests">{{ (index $.Data $test $header).Succeeded }}</span>/<span class="tests_skipped" title="skipped tests">{{ (index $.Data $test $header).Skipped }}</span>
            </div>
            {{end}}
        </td>
        {{ end }}
    </tr>
    {{ end }}
</table>
</body>
</html>
`
	shortUsage = "flake-report-creator prow creates an ad-hoc report of any GCS directories that contain kubevirt testing junit files"
)

func init() {
	prowOpts = prowOptions{}
	prowCommand.PersistentFlags().StringVar(&prowOpts.bucketName, "bucketName", "", "The name of the GCS bucket")
	prowCommand.PersistentFlags().StringArrayVar(&prowOpts.jobDataPathes, "jobDataPath", nil, "Path below the same bucket to retrieve the data from. May occur more than once")
	prowCommand.PersistentFlags().StringVar(&prowOpts.matchingSubDirRegExp, "subDirRegex", "", "Regular expression for matching sub directories (will optimize runtime)")
	prowCommand.PersistentFlags().BoolVar(&prowOpts.useSubDirs, "useSubDirs", true, "Whether to fetch the subdirectories of each data path and then retrieve the data or to try to directly retrieve the data")
	prowCommand.PersistentFlags().DurationVar(&prowOpts.startFrom, "startFrom", 14*24*time.Hour, "The duration when the report data should be fetched")
	prowCommand.PersistentFlags().StringVar(&prowOpts.presubmits, "presubmits", "", "Presubmit numbers to report")
	prowCommand.PersistentFlags().StringVar(&prowOpts.periodics, "periodics", "", "periodics to report (either a comma delimited list, or * for all)")
	prowCommand.PersistentFlags().StringVar(&prowOpts.ciSystem, "ci-system", "", fmt.Sprintf("ci-system to report for (one of %s)", strings.Join([]string{kubevirtCI, openshiftCI}, ", ")))
}

type prowOptions struct {
	org                  string
	repo                 string
	gcsBaseUrl           string
	bucketName           string
	jobDataPathes        []string
	matchingSubDirRegExp string
	useSubDirs           bool
	startFrom            time.Duration
	presubmits           string
	periodics            string
	ciSystem             string
}

func (o *prowOptions) validate() error {
	var argsToEvaluate map[string]string
	if o.ciSystem != "" {
		var errString []string
		if o.matchingSubDirRegExp != "" {
			errString = append(errString, "--subDirRegex must not be set")
		}
		if o.bucketName != "" {
			errString = append(errString, "when --ci-system is used, --bucketName must not be set")
		}
		if len(o.jobDataPathes) > 0 {
			errString = append(errString, "when --ci-system is used, --jobDataPath must not be set")
		}
		if len(errString) > 0 {
			return fmt.Errorf("when --ci-system is used, %s", strings.Join(errString, " and "))
		}
		var jobType string
		if o.presubmits != "" {
			jobType = presubmits
		} else if o.periodics != "" {
			jobType = periodics
		} else {
			return fmt.Errorf("either --periodics or --presubmits need to be set")
		}
		if v, exists := argMatrix[jobType][o.ciSystem]; exists {
			argsToEvaluate = v
		} else {
			return fmt.Errorf("ciSystem %s not found, one of { %s } must be used", o.ciSystem, strings.Join([]string{kubevirtCI, openshiftCI}, " , "))
		}
		o.bucketName = argsToEvaluate[bucketName]
		value, err := strconv.ParseBool(argsToEvaluate[useSubDirs])
		if err != nil {
			return fmt.Errorf("failed to parse useSubDirs value %s", argsToEvaluate[useSubDirs])
		}
		o.useSubDirs = value
		o.matchingSubDirRegExp = argsToEvaluate[subDirRegex]
		switch jobType {
		case presubmits:
			var jobDataPathes []string
			for _, presubmit := range strings.Split(o.presubmits, ",") {
				jobDataPathes = append(jobDataPathes, fmt.Sprintf(argsToEvaluate[jobDataPath], presubmit))
			}
			o.jobDataPathes = jobDataPathes
			break
		case periodics:
			var jobDataPathes []string
			for _, periodic := range strings.Split(o.periodics, ",") {
				jobDataPathes = append(jobDataPathes, argsToEvaluate[jobDataPath]+periodic)
			}
			o.jobDataPathes = jobDataPathes
			break
		default:
			return fmt.Errorf("unknown jobType %s", jobType)
		}
	}

	if len(o.jobDataPathes) == 0 {
		return fmt.Errorf("no pathes given, check for at least one --jobDataPath")
	}
	return nil
}

var (
	argMatrix = map[string]map[string]map[string]string{
		presubmits: {
			kubevirtCI: {
				bucketName:  "kubevirt-prow",
				useSubDirs:  "true",
				subDirRegex: "^pull-kubevirt-e2e-k8s-.*",
				jobDataPath: "pr-logs/pull/kubevirt_kubevirt/%s",
			},
			openshiftCI: {
				bucketName:  "origin-ci-test",
				useSubDirs:  "true",
				subDirRegex: ".*-(e2e-[a-z\\d]+)$",
				jobDataPath: "pr-logs/pull/openshift_release/%s",
			},
		},
		periodics: {
			kubevirtCI: {
				bucketName:  "kubevirt-prow",
				useSubDirs:  "false",
				subDirRegex: ".*",
				//jobDataPath: "logs/periodic-kubevirt-e2e-k8s-1.20-sig-storage",
				// this is actually used as a prefix in the GCS Object query
				jobDataPath: "logs/periodic-kubevirt-e2e-k8s-",
			},
			openshiftCI: {
				bucketName:  "origin-ci-test",
				useSubDirs:  "false",
				subDirRegex: ".*-(e2e-[a-z\\d]+)$",
				//jobDataPath: "logs/periodic-ci-kubevirt-kubevirt-main-0.34_4.6-e2e",
				// this is actually used as a prefix in the GCS Object query
				jobDataPath: "logs/periodic-ci-kubevirt-kubevirt-main-",
			},
		},
	}

	maxTime = time.Unix(1<<63-62135596801, 999999999)

	prowCommand = &cobra.Command{
		Use:   "prow",
		Short: shortUsage,
		Long: shortUsage + `

It can create a report from prow job runs of presubmits and periodics on openshift-ci or kubevirt-ci.

Examples:

# run a report for prs from openshift ci
$ flake-report-creator prow --ci-system=openshift --presubmits 22352,23021

# create a report over a set of selected pull requests for kubevirtci
$ flake-report-creator prow --ci-system=kubevirt --presubmits 6812,6815,6818

# create a report over a set of periodics on openshift-ci
$ flake-report-creator prow --ci-system=openshift --periodics 0.34,0.36,0.41,4.10

# create a report over a set of selected periodics for kubevirtci
$ flake-report-creator prow --ci-system=kubevirt --periodics 1.21,1.22
`,
		RunE: runProwReport,
	}

	prowOpts prowOptions
)

type SimpleReportParams struct {
	flakefinder.Params
	BucketName    string
	JobDataPathes []string
}

func ProwCommand() *cobra.Command {
	return prowCommand
}

func runProwReport(cmd *cobra.Command, args []string) error {
	err := cmd.InheritedFlags().Parse(args)
	if err != nil {
		return err
	}

	if err = globalOpts.Validate(); err != nil {
		_, err2 := fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString(), err)
		if err2 != nil {
			return err2
		}
		return fmt.Errorf("invalid arguments provided: %v", err)
	}

	if err = prowOpts.validate(); err != nil {
		_, err = fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		if err != nil {
			return err
		}

		return fmt.Errorf("invalid arguments provided: %v", err)
	}

	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("Failed to create new storage client: %v.\n", err)
	}

	reportOutputWriter, err := os.OpenFile(globalOpts.outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("failed to write report: %v", err)
	}

	startOfReport := time.Now().Add(-1 * prowOpts.startFrom)

	reports := []*flakefinder.JobResult{}
	var subDirRegExp *regexp.Regexp
	if prowOpts.matchingSubDirRegExp != "" {
		subDirRegExp = regexp.MustCompile(prowOpts.matchingSubDirRegExp)
	}
	if prowOpts.periodics != "" {
		basePath := path.Dir(prowOpts.jobDataPathes[0])
		dirs, err := flakefinder.ListGcsObjects(ctx, storageClient, prowOpts.bucketName, basePath+"/", "/")
		if err != nil {
			log.Printf("failed to list objects for dataPath %v: %v", path.Base(prowOpts.jobDataPathes[0])+"/", err)
		}
		var realJobDataPathes []string
		for _, dataPath := range prowOpts.jobDataPathes {
			for _, dir := range dirs {
				if !strings.HasPrefix(path.Join(basePath, dir), dataPath) {
					continue
				}
				results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, prowOpts.bucketName, []string{basePath, dir}, startOfReport, maxTime)
				if err != nil {
					log.Printf("failed to load JUnit files for job %v: %v", dataPath, err)
				}
				realJobDataPathes = append(realJobDataPathes, path.Join(basePath, dir))
				reports = append(reports, results...)
			}
		}
		prowOpts.jobDataPathes = realJobDataPathes
	} else {
		for _, dataPath := range prowOpts.jobDataPathes {
			if prowOpts.useSubDirs {
				subDirs, err := flakefinder.ListGcsObjects(ctx, storageClient, prowOpts.bucketName, dataPath+"/", "/")
				if err != nil {
					log.Printf("failed to list objects for dataPath %v: %v", dataPath, err)
				}
				for _, subDir := range subDirs {
					if subDirRegExp != nil && !subDirRegExp.MatchString(subDir) {
						continue
					}
					results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, prowOpts.bucketName, []string{dataPath, subDir}, startOfReport, maxTime)
					if err != nil {
						log.Printf("failed to load JUnit files for job %v: %v", path.Join(dataPath, subDir), err)
					}
					reports = append(reports, results...)
				}
			} else {
				results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, prowOpts.bucketName, []string{dataPath}, startOfReport, maxTime)
				if err != nil {
					log.Printf("failed to load JUnit files for job %v: %v", dataPath, err)
				}
				reports = append(reports, results...)
			}
		}
	}

	parameters := flakefinder.CreateFlakeReportData(reports, []int{}, maxTime, prowOpts.org, prowOpts.repo, startOfReport)

	log.Printf("writing output file to %s", globalOpts.outputFile)

	err = flakefinder.WriteTemplateToOutput(ProwReportTemplate, SimpleReportParams{parameters, prowOpts.bucketName, prowOpts.jobDataPathes}, reportOutputWriter)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to write report: %v", err))
	}
	return nil
}
