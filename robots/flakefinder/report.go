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
	"html/template"
	"io"
	"log"
	"os"
	"path"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"github.com/joshdk/go-junit"
)

const tpl = `
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
            bottom: 125%;
            left: 50%;
            margin-left: -110px;
        }

        .nowrap {
            white-space: nowrap;
        }

        /* Popup arrow */
        .popup .popuptext::after {
            content: "";
            position: absolute;
            top: 100%;
            left: 50%;
            margin-left: -5px;
            border-width: 5px;
            border-style: solid;
            border-color: #555 transparent transparent transparent;
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

<h1>flakefinder report for {{ $.Org }}/{{ $.Repo }} - {{ $.Date }}</h1>

{{ if not .Headers }}
	<div>No failing tests! :)</div>
{{ else }}
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
                    <div class="{{.Severity}} nowrap"><a href="https://prow.apps.ovirt.org/view/gcs/kubevirt-prow/pr-logs/pull/{{ $.Org }}_{{ $.Repo }}/{{.PR}}/{{.Job}}/{{.BuildNumber}}">{{.BuildNumber}}</a> (<a href="https://github.com/{{ $.Org }}/{{ $.Repo }}/pull/{{.PR}}">#{{.PR}}</a>)</div>
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

<div>
    Source PRs: {{ range $key := $.PrNumbers }}<a href="https://github.com/{{ $.Org }}/{{ $.Repo }}/pull/{{ $key }}">#{{ $key }}</a>, {{ end }}
</div>

<script>
    function popup(id) {
        var popup = document.getElementById("target" + id);
        popup.classList.toggle("show");
    }
</script>

</body>
</html>
`

type Params struct {
	Data      map[string]map[string]*Details
	Headers   []string
	Tests     []string
	PrNumbers []int
	Date      string
	Org       string
	Repo      string
}

type Details struct {
	Succeeded int
	Skipped   int
	Failed    int
	Severity  string
	Jobs      []*Job
}

type Job struct {
	BuildNumber int
	Severity    string
	PR          int
	Job         string
}

// WriteReportToBucket creates the actual formatted report file from the report data and writes it to the bucket
func WriteReportToBucket(ctx context.Context, client *storage.Client, reports []*Result, merged time.Duration, org, repo string, prNumbers []int, writeToStdout bool) (err error) {
	reportObject := client.Bucket(BucketName).Object(path.Join(ReportOutputPath, CreateReportFileName(time.Now(), merged)))
	log.Printf("Report will be written to gs://%s/%s", BucketName, reportObject.ObjectName())
	reportOutputWriter := reportObject.NewWriter(ctx)
	err = Report(reports, reportOutputWriter, org, repo, prNumbers, writeToStdout)
	if err != nil {
		return fmt.Errorf("failed on generating report: %v", err)
	}
	err = reportOutputWriter.Close()
	if err != nil {
		return fmt.Errorf("failed on closing report object: %v", err)
	}
	return nil
}

func CreateReportFileName(reportTime time.Time, merged time.Duration) string {
	return fmt.Sprintf(ReportFilePrefix+"%s-%03dh.html", reportTime.Format("2006-01-02"), int(merged.Hours()))
}

func Report(results []*Result, reportOutputWriter *storage.Writer, org string, repo string, prNumbers []int, writeToStdout bool) error {
	data := map[string]map[string]*Details{}
	headers := []string{}
	tests := []string{}
	buildNumberMap := map[int]struct{}{}
	headerMap := map[string]struct{}{}

	for _, result := range results {

		// first find failing tests to isolate tests which interest us
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if test.Status == junit.StatusFailed || test.Status == junit.StatusError {
					buildNumberMap[result.BuildNumber] = struct{}{}
					testEntry := data[test.Name]
					if testEntry == nil {
						tests = append(tests, test.Name)
						testEntry = map[string]*Details{}
						data[test.Name] = testEntry
					}

					if _, exists := testEntry[result.Job]; !exists {
						testEntry[result.Job] = &Details{}
					}
					if _, exists := headerMap[result.Job]; !exists {
						headerMap[result.Job] = struct{}{}
						headers = append(headers, result.Job)
					}
					testEntry[result.Job].Failed = testEntry[result.Job].Failed + 1
				}
			}
		}

	}

	// second enrich failed tests with additional information
	for _, result := range results {
		if _, exists := buildNumberMap[result.BuildNumber]; !exists {
			// if not in the map now, then skip it
			continue
		}
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if _, exists := data[test.Name]; !exists {
					continue
				}
				if _, exists := data[test.Name][result.Job]; !exists {
					data[test.Name][result.Job] = &Details{}
				}
				if test.Status == junit.StatusSkipped {
					data[test.Name][result.Job].Skipped = data[test.Name][result.Job].Skipped + 1
				} else if test.Status == junit.StatusPassed {
					data[test.Name][result.Job].Succeeded = data[test.Name][result.Job].Succeeded + 1
					data[test.Name][result.Job].Jobs = append(data[test.Name][result.Job].Jobs, &Job{Severity: "green", BuildNumber: result.BuildNumber, Job: result.Job, PR: result.PR})
				} else {
					data[test.Name][result.Job].Jobs = append(data[test.Name][result.Job].Jobs, &Job{Severity: "red", BuildNumber: result.BuildNumber, Job: result.Job, PR: result.PR})
				}
			}
		}
	}

	// third, calculate the severity
	// second enrich failed tests with additional information
	for _, result := range results {
		if _, exists := buildNumberMap[result.BuildNumber]; !exists {
			// if not in the map now, then skip it
			continue
		}
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if _, exists := data[test.Name]; !exists {
					continue
				}
				if _, exists := data[test.Name][result.Job]; !exists {
					continue
				}

				entry := data[test.Name][result.Job]

				SetSeverity(entry)
			}
		}
	}

	testsSortedByRelevance := SortTestsByRelevance(data, tests)
	parameters := Params{
		Data:      data,
		Headers:   headers,
		Tests:     testsSortedByRelevance,
		PrNumbers: prNumbers,
		Date:      time.Now().Format("2006-01-02"),
		Org:       org,
		Repo:      repo,
	}
	var err error
	if reportOutputWriter != nil {
		err = WriteReportToOutput(reportOutputWriter, parameters)
	}
	if writeToStdout {
		err = WriteReportToOutput(os.Stdout, parameters)
	}

	if err != nil {
		return fmt.Errorf("failed to render report template: %v", err)
	}

	return nil
}

// SetSeverity sets the field Severity on the passed details according to the ratio of failed vs succeeded tests,
// where test results are the more severe the more test failures they contain in relation to succeeded tests.
func SetSeverity(entry *Details) {
	var ratio float32 = 1.0
	if entry.Succeeded > 0 {
		ratio = float32(entry.Failed) / float32(entry.Succeeded)
	}

	entry.Severity = Fine
	if entry.Succeeded == 0 && entry.Failed == 0 {
		entry.Severity = Unimportant
	} else if ratio > 0.5 {
		entry.Severity = HeavilyFlaky
	} else if ratio > 0.2 {
		entry.Severity = MostlyFlaky
	} else if ratio > 0.1 {
		entry.Severity = ModeratelyFlaky
	} else if ratio > 0 {
		entry.Severity = MildlyFlaky
	}
}

// SortTestsByRelevance sorts given tests according to the severity from the test data, where tests with a higher
// severity have a smaller index in the slice than tests with a lower severity.
// The returned slice does not contain
// duplicates, thus if a test has data with several severities the highest one is picked, leading to an earlier
// encounter in the slice.
func SortTestsByRelevance(data map[string]map[string]*Details, tests []string) (testsSortedByRelevance []string) {

	testsToSeveritiesWithNumbers := map[string]map[string]int{}

	foundTests := map[string]struct{}{}

	// Group all tests by severity, ignoring duplicates for the moment, but keeping a record of tests
	// that have been found
	flakinessToTestNames := map[string][]string{}
	for test, jobsToDetails := range data {
		for _, details := range jobsToDetails {
			if _, exists := flakinessToTestNames[details.Severity]; !exists {
				flakinessToTestNames[details.Severity] = []string{}
			}
			flakinessToTestNames[details.Severity] = append(flakinessToTestNames[details.Severity], test)

			if _, exists := testsToSeveritiesWithNumbers[test]; !exists {
				testsToSeveritiesWithNumbers[test] = map[string]int{details.Severity: 0}
			}
			testsToSeveritiesWithNumbers[test][details.Severity] = testsToSeveritiesWithNumbers[test][details.Severity] + 1

			foundTests[test] = struct{}{}
		}
	}

	// Build up the initial sorted result (with duplicates)
	initialTestsSortedByRelevance := BuildUpSortedTestsBySeverity(testsToSeveritiesWithNumbers)

	// Append all tests that have not been found in the data
	for _, test := range tests {
		if _, exists := foundTests[test]; !exists {
			initialTestsSortedByRelevance = append(initialTestsSortedByRelevance, test)
		}
	}

	// Now kill the duplicates by keeping a map of whether the test was encountered before
	encounteredTests := map[string]struct{}{}
	for _, test := range initialTestsSortedByRelevance {
		if _, exists := encounteredTests[test]; !exists {
			encounteredTests[test] = struct{}{}
			testsSortedByRelevance = append(testsSortedByRelevance, test)
		}
	}

	return
}

type TestToSeverityOccurrences struct {
	Name                string
	SeverityOccurrences []int
}

type BySeverity []*TestToSeverityOccurrences

func (t BySeverity) Len() int { return len(t) }
func (t BySeverity) Less(i, j int) bool {
	if len(t[i].SeverityOccurrences) != len(t[j].SeverityOccurrences) {
		return len(t[i].SeverityOccurrences) < len(t[j].SeverityOccurrences)
	}
	for index := 0; index < len(t[i].SeverityOccurrences); index++ {
		if t[i].SeverityOccurrences[index] < t[j].SeverityOccurrences[index] {
			return true
		} else if t[i].SeverityOccurrences[index] > t[j].SeverityOccurrences[index] {
			return false
		}
	}
	return t[i].Name > t[j].Name
}
func (t BySeverity) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

func BuildUpSortedTestsBySeverity(testsToSeveritiesWithOccurrences map[string]map[string]int) []string {

	// Create flat array storing test names to numbers of occurrences for each severity order by priority from left
	// to right
	testsWithAllSeverityOccurrences := make([]*TestToSeverityOccurrences, 0)
	severitiesInOrder := []string{HeavilyFlaky, MostlyFlaky, ModeratelyFlaky, MildlyFlaky, Fine, Unimportant}
	for test, severitiesWithOccurrences := range testsToSeveritiesWithOccurrences {
		severityOccurrences := make([]int, len(severitiesInOrder))
		testWithAllSeverityOccurrences := TestToSeverityOccurrences{Name: test, SeverityOccurrences: severityOccurrences}
		for index, severity := range severitiesInOrder {
			occurrencesOfSeverity := 0
			if existingNumber, exists := severitiesWithOccurrences[severity]; exists {
				occurrencesOfSeverity = existingNumber
			}
			severityOccurrences[index] = occurrencesOfSeverity
		}
		testsWithAllSeverityOccurrences = append(testsWithAllSeverityOccurrences, &testWithAllSeverityOccurrences)
	}

	// now sort the array
	sort.Sort(sort.Reverse(BySeverity(testsWithAllSeverityOccurrences)))

	initialTestsSortedByRelevance := make([]string, len(testsWithAllSeverityOccurrences))
	for index, test := range testsWithAllSeverityOccurrences {
		initialTestsSortedByRelevance[index] = test.Name
	}
	return initialTestsSortedByRelevance
}

const (
	HeavilyFlaky    = "red"
	MostlyFlaky     = "orange"
	ModeratelyFlaky = "yellow"
	MildlyFlaky     = "almostgreen"
	Fine            = "green"
	Unimportant     = "unimportant"
)

func WriteReportToOutput(writer io.Writer, parameters Params) error {
	t, err := template.New("report").Parse(tpl)
	if err != nil {
		return fmt.Errorf("failed to load report template: %v", err)
	}

	err = t.Execute(writer, parameters)
	return err
}
