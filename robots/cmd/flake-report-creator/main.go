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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"

	"kubevirt.io/project-infra/robots/pkg/flakefinder"
)

func flagOptions() options {
	o := options{}
	flag.StringVar(&o.bucketName, "bucket-name", "", "The name of the GCS bucket")
	flag.Var(&o.jobDataPathes, "job-data-path", "Path below the same bucket to retrieve the data from. May occur more than once")
	flag.StringVar(&o.outputFile, "output-file", "", "Path to output file, if not given, a temporary file will be used")
	flag.BoolVar(&o.overwrite, "overwrite", false, "Whether to overwrite output file")
	flag.StringVar(&o.matchingSubDirRegExp, "sub-dir-regex", "", "Regular expression for matching sub directories (will optimize runtime)")
	flag.BoolVar(&o.useSubDirs, "use-sub-dirs", true, "Whether to fetch the subdirectories of each data path and then retrieve the data or to try to directly retrieve the data")
	flag.DurationVar(&o.startFrom, "start-from", 14*24*time.Hour, "The duration when the report data should be fetched")
	flag.StringVar(&o.presubmits, "presubmits", "", "Presubmit numbers to report")
	flag.StringVar(&o.periodics, "periodics", "", "periodics to report (either a comma delimited list, or * for all)")
	flag.StringVar(&o.ciSystem, "ci-system", "", fmt.Sprintf("ci-system to report for (one of %s)", strings.Join([]string{kubevirt_ci, openshift_ci}, ", ")))
	flag.Parse()
	return o
}

func (o *options) validate() error {
	if o.outputFile == "" {
		outputFile, err := os.CreateTemp("", "flakefinder-*.html")
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
		o.outputFile = outputFile.Name()
	} else {
		stat, err := os.Stat(o.outputFile)
		if err != nil && err != os.ErrNotExist {
			return fmt.Errorf("failed to write report: %v", err)
		}
		if stat.IsDir() {
			return fmt.Errorf("failed to write report, file %s is a directory", o.outputFile)
		}
		if err == nil && !o.overwrite {
			return fmt.Errorf("failed to write report, file %s exists", o.outputFile)
		}
	}

	var argsToEvaluate map[string]string
	if o.ciSystem != "" {
		var errString []string
		if o.matchingSubDirRegExp != "" {
			errString = append(errString, "--sub-dir-regex must not be set")
		}
		if o.bucketName != "" {
			errString = append(errString, "when --ci-system is used, --bucket-name must not be set")
		}
		if len(o.jobDataPathes) > 0 {
			errString = append(errString, "when --ci-system is used, --job-data-path must not be set")
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
			return fmt.Errorf("ciSystem %s not found, one of { %s } must be used", o.ciSystem, strings.Join([]string{kubevirt_ci, openshift_ci}, " , "))
		}
		o.bucketName = argsToEvaluate[bucketName]
		value, err := strconv.ParseBool(argsToEvaluate[useSubDirs])
		if err != nil {
			return fmt.Errorf("failed to parse use-sub-dirs value %s", argsToEvaluate[useSubDirs])
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
		return fmt.Errorf("no pathes given, check for at least one --job-data-path")
	}
	return nil
}

const (
	kubevirt_ci  = "kubevirt"
	openshift_ci = "openshift"
	presubmits   = "presubmits"
	periodics    = "periodics"
	bucketName   = "bucket-name"
	useSubDirs   = "use-sub-dirs"
	subDirRegex  = "sub-dir-regex"
	jobDataPath  = "job-data-path"
)

var argMatrix = map[string]map[string]map[string]string{
	presubmits: {
		kubevirt_ci: {
			bucketName:  "kubevirt-prow",
			useSubDirs:  "true",
			subDirRegex: "^pull-kubevirt-e2e-k8s-.*",
			jobDataPath: "pr-logs/pull/kubevirt_kubevirt/%s",
		},
		openshift_ci: {
			bucketName:  "origin-ci-test",
			useSubDirs:  "true",
			subDirRegex: ".*-(e2e-[a-z\\d]+)$",
			jobDataPath: "pr-logs/pull/openshift_release/%s",
		},
	},
	periodics: {
		kubevirt_ci: {
			bucketName:  "kubevirt-prow",
			useSubDirs:  "false",
			subDirRegex: ".*",
			//jobDataPath: "logs/periodic-kubevirt-e2e-k8s-1.20-sig-storage",
			// this is actually used as a prefix in the GCS Object query
			jobDataPath: "logs/periodic-kubevirt-e2e-k8s-",
		},
		openshift_ci: {
			bucketName:  "origin-ci-test",
			useSubDirs:  "false",
			subDirRegex: ".*-(e2e-[a-z\\d]+)$",
			//jobDataPath: "logs/periodic-ci-kubevirt-kubevirt-main-0.34_4.6-e2e",
			// this is actually used as a prefix in the GCS Object query
			jobDataPath: "logs/periodic-ci-kubevirt-kubevirt-main-",
		},
	},
}

var minTime = time.Time{}
var maxTime = time.Unix(1<<63-62135596801, 999999999)

type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, " ")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

type options struct {
	org                  string
	repo                 string
	gcsBaseUrl           string
	bucketName           string
	jobDataPathes        arrayFlags
	outputFile           string
	overwrite            bool
	matchingSubDirRegExp string
	useSubDirs           bool
	startFrom            time.Duration
	presubmits           string
	periodics            string
	ciSystem             string
}

type SimpleReportParams struct {
	flakefinder.Params
	BucketName    string
	JobDataPathes []string
}

const ReportTemplate = `
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

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()
	err := o.validate()
	if err != nil {
		log.Fatalf("invalid options given: %v", err)
	}

	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}

	reportOutputWriter, err := os.OpenFile(o.outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil && err != os.ErrNotExist {
		log.Fatal(fmt.Errorf("failed to write report: %v", err))
	}

	startOfReport := time.Now().Add(-1 * o.startFrom)

	reports := []*flakefinder.JobResult{}
	var subDirRegExp *regexp.Regexp
	if o.matchingSubDirRegExp != "" {
		subDirRegExp = regexp.MustCompile(o.matchingSubDirRegExp)
	}
	if o.periodics != "" {
		basePath := path.Dir(o.jobDataPathes[0])
		dirs, err := flakefinder.ListGcsObjects(ctx, storageClient, o.bucketName, basePath+"/", "/")
		if err != nil {
			log.Printf("failed to list objects for dataPath %v: %v", path.Base(o.jobDataPathes[0])+"/", err)
		}
		var realJobDataPathes []string
		for _, dataPath := range o.jobDataPathes {
			for _, dir := range dirs {
				if !strings.HasPrefix(path.Join(basePath, dir), dataPath) {
					continue
				}
				results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, o.bucketName, []string{basePath, dir}, startOfReport, maxTime)
				if err != nil {
					log.Printf("failed to load JUnit files for job %v: %v", dataPath, err)
				}
				realJobDataPathes = append(realJobDataPathes, path.Join(basePath, dir))
				reports = append(reports, results...)
			}
		}
		o.jobDataPathes = realJobDataPathes
	} else {
		for _, dataPath := range o.jobDataPathes {
			if o.useSubDirs {
				subDirs, err := flakefinder.ListGcsObjects(ctx, storageClient, o.bucketName, dataPath+"/", "/")
				if err != nil {
					log.Printf("failed to list objects for dataPath %v: %v", dataPath, err)
				}
				for _, subDir := range subDirs {
					if subDirRegExp != nil && !subDirRegExp.MatchString(subDir) {
						continue
					}
					results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, o.bucketName, []string{dataPath, subDir}, startOfReport, maxTime)
					if err != nil {
						log.Printf("failed to load JUnit files for job %v: %v", path.Join(dataPath, subDir), err)
					}
					reports = append(reports, results...)
				}
			} else {
				results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, o.bucketName, []string{dataPath}, startOfReport, maxTime)
				if err != nil {
					log.Printf("failed to load JUnit files for job %v: %v", dataPath, err)
				}
				reports = append(reports, results...)
			}
		}
	}

	parameters := flakefinder.CreateFlakeReportData(reports, []int{}, maxTime, o.org, o.repo, startOfReport)

	log.Printf("writing output file to %s", o.outputFile)

	err = flakefinder.WriteTemplateToOutput(ReportTemplate, SimpleReportParams{parameters, o.bucketName, o.jobDataPathes}, reportOutputWriter)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to write report: %v", err))
	}
}
