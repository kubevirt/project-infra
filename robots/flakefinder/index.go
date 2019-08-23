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
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"html/template"
	"io"
	"log"
	"path"
	"sort"
	"strings"
)

const indexTpl = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>kubevirt.io - flakefinder reports</title>
	<style>
		table, th, td {
		  border: 1px solid black;
		}
	</style>
</head>
<body>
	<table>
		<tr>
			<th colspan="3">flakefinder reports</th>
		</tr>
		<tr>
			<th>672h</th>
			<th>168h</th>
			<th>024h</th>
		</tr>
{{ range $reportFile := $.Reports }}
		<tr>
			<td><a href="{{ .FileName }}">{{ .Date }}</a></td>
		</tr>
{{ end }}

	</table>
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

type reportFile struct {
	Date     string
	FileName string
}

type indexParams struct {
	Reports []reportFile
}

// CreateReportIndex creates an index.html that links to the X most recent reports in GCS "folder", sorted from most
// recent to oldest
func CreateReportIndex(ctx context.Context, client *storage.Client) (err error) {
	reportDirGcsObjects, err := getReportItemsFromBucketDirectory(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get report items: %v", err)
	}

	reportIndexObjectWriter := CreateOutputWriter(client, ctx)

	err = WriteReportIndexPage(reportDirGcsObjects, reportIndexObjectWriter)
	if err != nil {
		return fmt.Errorf("failed generating index page: %v", err)
	}

	err = reportIndexObjectWriter.Close()
	if err != nil {
		return fmt.Errorf("failed closing index page writer: %v", err)
	}
	return nil
}

func CreateOutputWriter(client *storage.Client, ctx context.Context) io.WriteCloser {
	reportIndexObject := client.Bucket(BucketName).Object(path.Join(ReportsPath, "index.html"))
	log.Printf("Report index page will be written to gs://%s/%s", BucketName, reportIndexObject.ObjectName())
	reportIndexObjectWriter := reportIndexObject.NewWriter(ctx)
	return reportIndexObjectWriter
}

func WriteReportIndexPage(reportDirGcsObjects []string, reportIndexObjectWriter io.Writer) error {

	// Prepare template for index.html
	t, err := template.New("index").Parse(indexTpl)
	if err != nil {
		return fmt.Errorf("failed to load report template: %v", err)
	}

	parameters := PrepareDataForTemplate(reportDirGcsObjects)

	// write index page
	err = t.Execute(reportIndexObjectWriter, parameters)
	return err
}

func PrepareDataForTemplate(reportDirGcsObjects []string) indexParams {
	var reportFiles []reportFile
	for _, reportFileName := range reportDirGcsObjects {
		date := strings.Replace(reportFileName, ReportFilePrefix, "", -1)
		date = strings.Replace(date, ".html", "", -1)
		reportFiles = append(reportFiles, reportFile{Date: date, FileName: reportFileName})
	}
	parameters := indexParams{Reports: reportFiles}
	return parameters
}

// getReportItemsFromBucketDirectory fetches the X most recent report file names from report directory, returning only
// their basenames
func getReportItemsFromBucketDirectory(client *storage.Client, ctx context.Context) ([]string, error) {
	var reportDirGcsObjects []string
	it := client.Bucket(BucketName).Objects(ctx, &storage.Query{
		Prefix:    ReportsPath + "/",
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
		if !strings.HasPrefix(fileName, ReportFilePrefix) || !strings.HasSuffix(fileName, ".html") {
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
