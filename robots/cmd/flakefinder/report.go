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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"cloud.google.com/go/storage"

	"kubevirt.io/project-infra/robots/pkg/flakefinder"
)

const ReportTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>{{ $.Org }}/{{ $.Repo }} - flakefinder report</title>
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

        /* Popup container - can be anything you want */
        .popup {
            position: relative;
            display: inline-block;
            cursor: pointer;
            -webkit-user-select: none;
            -moz-user-select: none;
            -ms-user-select: none;
            user-select: none;
        }

        /* The actual popup */
        .popup .popuptext {
            visibility: hidden;
            width: 220px;
            background-color: #555;
            text-align: center;
            border-radius: 6px;
            padding: 8px 8px;
            position: absolute;
            z-index: 1;
            left: 50%;
            margin-left: -110px;
        }

        .popup .popuptextjoblist {
            visibility: hidden;
            width: 350px;
            background-color: #FFFFFF;
            text-align: center;
            border-radius: 6px;
            padding: 8px 8px;
            position: absolute;
            z-index: 1;
            left: 100%;
            margin-left: -350px;
        }

        .nowrap {
            white-space: nowrap;
        }

        /* Toggle this class - hide and show the popup */
        .popup .show {
            visibility: visible;
            -webkit-animation: fadeIn 1s;
            animation: fadeIn 1s;
        }

        /* Add animation (fade in the popup) */
        @-webkit-keyframes fadeIn {
            from {opacity: 0;}
            to {opacity: 1;}
        }

        @keyframes fadeIn {
            from {opacity: 0;}
            to {opacity:1 ;}
        }
    </style>
</head>
<body>

<h1>flakefinder report for {{ $.Org }}/{{ $.Repo }}</h1>

<div>
	Data range from {{ $.StartOfReport }} till {{ $.EndOfReport }}
</div>

<div>
{{ if $.PrNumbers }} Source PRs: {{ range $key := $.PrNumbers }}<a href="https://github.com/{{ $.Org }}/{{ $.Repo }}/pull/{{ $key }}">#{{ $key }}</a>, {{ end }}
{{ else }} No PRs merged 😞
{{ end }}
</div>

{{ if not .Headers }}
	<div>No failing tests! 🙂</div>
{{ else }}
<div id="failuresForJobs" onClick="popup(this.id)" class="popup right" >
	<u>list of job runs</u>
	<div class="popuptextjoblist right" id="targetfailuresForJobs">
		<table width="100%">
			{{ range $key, $jobFailures := $.FailuresForJobs }}<tr class="unimportant">
				<td>
					{{ if ne .PR 0 }}<a href="https://prow.ci.kubevirt.io/view/gcs/kubevirt-prow/pr-logs/pull/{{ $.Org }}_{{ $.Repo }}/{{.PR}}/{{.Job}}/{{.BuildNumber}}"><span title="job build number">{{.BuildNumber}}</span></a>{{ else }}<a href="https://prow.ci.kubevirt.io/view/gcs/kubevirt-prow/logs/{{.Job}}/{{.BuildNumber}}"><span title="job build number">{{.BuildNumber}}</span></a>{{ end }}
				</td>
				<td>
					<a href="https://github.com/{{ $.Org }}/{{ $.Repo }}/pull/{{.PR}}"><span title="pr number">#{{.PR}}</span></a>
				</td>
				<td>
					<div class="tests_failed"><span title="test failures">{{ .Failures }}</span></div>
				</td>
			</tr>{{ end }}
		</table>
	</div>
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
            <div id="r{{$row}}c{{$col}}" onClick="popup(this.id)" class="popup" >
                <span class="tests_failed" title="failed tests">{{ (index $.Data $test $header).Failed }}</span>/<span class="tests_passed" title="passed tests">{{ (index $.Data $test $header).Succeeded }}</span>/<span class="tests_skipped" title="skipped tests">{{ (index $.Data $test $header).Skipped }}</span>
                <div class="popuptext" id="targetr{{$row}}c{{$col}}">
                    {{ range $Job := (index $.Data $test $header).Jobs }}
                    <div class="{{.Severity}} nowrap">{{ if ne .PR 0 }}<a href="https://prow.ci.kubevirt.io/view/gcs/kubevirt-prow/pr-logs/pull/{{ $.Org }}_{{ $.Repo }}/{{.PR}}/{{.Job}}/{{.BuildNumber}}">{{.BuildNumber}}</a> (<a href="https://github.com/{{ $.Org }}/{{ $.Repo }}/pull/{{.PR}}">#{{.PR}}</a>){{ else }}<a href="https://prow.ci.kubevirt.io/view/gcs/kubevirt-prow/logs/{{.Job}}/{{.BuildNumber}}">{{.BuildNumber}}</a>{{ end }}</div>
                    {{ end }}
                </div>
            </div>
            {{end}}
        </td>
        {{ end }}
    </tr>
    {{ end }}
</table>
{{ end }}

<script>
    function popup(id) {
        var popup = document.getElementById("target" + id);
        popup.classList.toggle("show");
    }
</script>
<div>
<a href="index.html">Overview</a>
</div>

</body>
</html>
`

const ReportCSVTemplate = `"Test Name","Test Lane","Severity","Failed","Succeeded","Skipped","Jobs (JSON)"
{{ range $testName, $results := $.Data }}{{ range $jobName, $result := $results }}"{{ $testName }}","{{ $jobName }}","{{ $result.Severity }}",{{ $result.Failed }},{{ $result.Succeeded }},{{ $result.Skipped }},{{ range $job := $result.Jobs }}"{BuildNumber: {{ $job.BuildNumber }},Severity: ""{{ $job.Severity }}"",PR: {{ $job.PR }},Job: ""{{ $job.Job }}"",},"{{ end }}
{{ end }}{{ end }}`

// WriteReportToBucket creates the actual formatted report file from the report data and writes it to the bucket
func WriteReportToBucket(ctx context.Context, client *storage.Client, merged time.Duration, org, repo string, isDryRun bool, reportBaseData flakefinder.ReportBaseData) (err error) {
	var reportOutputWriter *storage.Writer
	var reportCSVOutputWriter *storage.Writer
	if !isDryRun {
		reportObject := client.Bucket(flakefinder.BucketName).Object(path.Join(ReportOutputPath, CreateReportFileNameWithEnding(reportBaseData.EndOfReport, merged, "html")))
		log.Printf("Report will be written to gs://%s/%s", reportObject.BucketName(), reportObject.ObjectName())
		reportCSVObject := client.Bucket(flakefinder.BucketName).Object(path.Join(ReportOutputPath, CreateReportFileNameWithEnding(reportBaseData.EndOfReport, merged, "csv")))
		log.Printf("Report CSV will be written to gs://%s/%s", reportCSVObject.BucketName(), reportCSVObject.ObjectName())
		reportOutputWriter = reportObject.NewWriter(ctx)
		defer reportOutputWriter.Close()
		reportCSVOutputWriter = reportCSVObject.NewWriter(ctx)
		defer reportCSVOutputWriter.Close()
	}
	err = Report(reportBaseData.JobResults, reportOutputWriter, reportCSVOutputWriter, org, repo, reportBaseData.PRNumbers, isDryRun, reportBaseData.StartOfReport, reportBaseData.EndOfReport)
	if err != nil {
		return fmt.Errorf("failed on generating report: %v", err)
	}
	return nil
}

func CreateReportFileNameWithEnding(reportTime time.Time, merged time.Duration, fileEnding string) string {
	return fmt.Sprintf(flakefinder.ReportFilePrefix+"%s-%03dh.%s", reportTime.Format("2006-01-02"), int(merged.Hours()), fileEnding)
}

type CSVParams struct {
	Data map[string]map[string]*flakefinder.Details
}

func Report(results []*flakefinder.JobResult, reportOutputWriter, reportCSVOutputWriter *storage.Writer, org, repo string, prNumbers []int, isDryRun bool, startOfReport, endOfReport time.Time) error {
	parameters := flakefinder.CreateFlakeReportData(results, prNumbers, endOfReport, org, repo, startOfReport)
	csvParams := CSVParams{Data: parameters.Data}
	var err error
	if !isDryRun {
		err = flakefinder.WriteTemplateToOutput(ReportTemplate, parameters, reportOutputWriter)
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
		err = flakefinder.WriteTemplateToOutput(ReportCSVTemplate, csvParams, reportCSVOutputWriter)
		if err != nil {
			return fmt.Errorf("failed to write report csv: %v", err)
		}
	} else {
		err = flakefinder.WriteTemplateToOutput(ReportTemplate, parameters, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to render report template: %v", err)
		}
		err = flakefinder.WriteTemplateToOutput(ReportCSVTemplate, csvParams, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to write report csv: %v", err)
		}
	}

	return nil
}
