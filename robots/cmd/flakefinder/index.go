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
	"os"
	"path"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"kubevirt.io/project-infra/pkg/flakefinder"
)

const indexTpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>{{ $.Org }}/{{ $.Repo }} - flakefinder reports</title>
	<style>
		table, th, td {
		  border: 1px solid black;
		}
	</style>
</head>
<body>
	<h1>flakefinder reports for {{ $.Org }}/{{ $.Repo }}</h1>
	<table>
		<tr>
			<th>Date</th>{{ range $key, $value := (index .Reports 0).ReportFiles }}
			<th>{{ $key }}</th>{{ end }}
		</tr>
		{{ range $reportFileRow := $.Reports }}<tr>
			<td>{{ .Date }}</td>{{ range $key, $value := .ReportFiles }}
			<td>{{ if eq $value "" }}&nbsp;{{ else }}<a href="{{ $value }}">{{ $key }}</a>{{ end }}</td>{{ end }}
		</tr>{{ end }}
	</table>
<div>
<a href="../../index.html">Overview</a>
</div>
</body>
</html>
`

type ReportFileMergedDuration string

const (
	Day       ReportFileMergedDuration = "024h"
	Week      ReportFileMergedDuration = "168h"
	FourWeeks ReportFileMergedDuration = "672h"
)

type ReportFilesRow struct {
	Date        string
	ReportFiles map[ReportFileMergedDuration]string
}

type IndexParams struct {
	Reports []ReportFilesRow
	Org     string
	Repo    string
}

// CreateReportIndex creates an index.html that links to the X most recent reports in GCS "folder", sorted from most
// recent to oldest
func CreateReportIndex(ctx context.Context, client *storage.Client, org, repo string, printIndexPageToStdOut bool) (err error) {
	reportDirGcsObjects, err := getReportItemsFromBucketDirectory(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get report items: %v", err)
	}

	if printIndexPageToStdOut {
		err = WriteReportIndexPage(reportDirGcsObjects, os.Stdout, org, repo)
		if err != nil {
			return fmt.Errorf("failed generating index page: %v", err)
		}
	} else {
		reportIndexObjectWriter := flakefinder.CreateOutputWriter(client, ctx, ReportOutputPath)
		err = WriteReportIndexPage(reportDirGcsObjects, reportIndexObjectWriter, org, repo)
		if err != nil {
			return fmt.Errorf("failed generating index page: %v", err)
		}
		err = reportIndexObjectWriter.Close()
		if err != nil {
			return fmt.Errorf("failed closing index page writer: %v", err)
		}
	}
	return nil
}

func WriteReportIndexPage(reportDirGcsObjects []string, reportIndexObjectWriter io.Writer, org, repo string) error {

	// Prepare template for index.html
	t, err := template.New("index").Parse(indexTpl)
	if err != nil {
		return fmt.Errorf("failed to load report template: %v", err)
	}

	parameters := PrepareDataForTemplate(reportDirGcsObjects, org, repo)

	// write index page
	err = t.Execute(reportIndexObjectWriter, parameters)
	return err
}

// PrepareDataForTemplate returns a data structure to easily create a 2d table with the report objects associated
// by date, then by report duration
//
// Assumptions: input data sorted in alphanumeric desc order, i.e.
// [
//
//		"flakefinder-2019-08-24-672h.html",
//		"flakefinder-2019-08-24-168h.html",
//		"flakefinder-2019-08-24-024h.html",
//	 ...
//
// ]
//
// Note: legacy format "flakefinder-2019-08-24.html" is allowed, missing duration leads to taking this for weekly (168h)
func PrepareDataForTemplate(reportDirGcsObjects []string, org string, repo string) IndexParams {
	var reportData []ReportFilesRow
	indexMap := make(map[string]ReportFilesRow)

	for _, reportFileName := range reportDirGcsObjects {
		date := strings.Replace(reportFileName, flakefinder.ReportFilePrefix, "", -1)
		date = strings.Replace(date, ".html", "", -1)
		mergedDuration := ReportFileMergedDuration(date[strings.LastIndex(date, "-")+1:])
		if mergedDuration != Day && mergedDuration != Week && mergedDuration != FourWeeks {
			mergedDuration = Week
		} else {
			date = strings.Replace(date, fmt.Sprintf("-%s", mergedDuration), "", -1)
		}
		if reportFilesRow, ok := indexMap[date]; !ok {
			reportFilesRow = ReportFilesRow{Date: date, ReportFiles: map[ReportFileMergedDuration]string{
				FourWeeks: "",
				Week:      "",
				Day:       "",
			}}
			reportFilesRow.ReportFiles[mergedDuration] = reportFileName
			indexMap[date] = reportFilesRow
			reportData = append(reportData, indexMap[date])
		} else {
			reportFilesRow.ReportFiles[mergedDuration] = reportFileName
		}
	}

	return IndexParams{Reports: reportData, Org: org, Repo: repo}
}

// getReportItemsFromBucketDirectory fetches the X most recent report file names from report directory, returning only
// their basenames
func getReportItemsFromBucketDirectory(client *storage.Client, ctx context.Context) ([]string, error) {
	var reportDirGcsObjects []string
	it := client.Bucket(flakefinder.BucketName).Objects(ctx, &storage.Query{
		Prefix:    ReportOutputPath + "/",
		Delimiter: "/",
	})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating: %v", err)
		}
		reportDirGcsObjects = append(reportDirGcsObjects, path.Base(attrs.Name))
	}
	return FilterReportItemsForIndexPage(reportDirGcsObjects), nil
}

// filterReportItemsForIndexPage removes all non relevant report objects
func FilterReportItemsForIndexPage(fileNames []string) []string {
	result := make([]string, 0)
	for _, fileName := range fileNames {
		if !strings.HasPrefix(fileName, flakefinder.ReportFilePrefix) || !strings.HasSuffix(fileName, ".html") {
			continue
		}
		result = append(result, fileName)
	}
	// keep only the X most recent
	sort.Sort(sort.Reverse(sort.StringSlice(result)))
	if len(result) > MaxNumberOfReportsToLinkTo {
		result = result[:MaxNumberOfReportsToLinkTo]
	}
	return result
}
