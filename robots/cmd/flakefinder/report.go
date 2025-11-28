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
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"time"

	"kubevirt.io/project-infra/pkg/flakefinder"

	"cloud.google.com/go/storage"
)

//go:embed report.gohtml
var ReportTemplate string

//go:embed reportCSV.gotemplate
var ReportCSVTemplate string

// WriteReportToBucket creates the actual formatted report file from the report data and writes it to the bucket
func WriteReportToBucket(ctx context.Context, client *storage.Client, merged time.Duration, org, repo string, isDryRun bool, reportBaseData flakefinder.ReportBaseData) (err error) {
	var reportOutputWriter *storage.Writer
	var reportCSVOutputWriter *storage.Writer
	var reportJSONOutputWriter *storage.Writer
	if !isDryRun {
		reportObject := client.Bucket(flakefinder.BucketName).Object(path.Join(ReportOutputPath, CreateReportFileNameWithEnding(reportBaseData.EndOfReport, merged, "html")))
		log.Printf("Report will be written to gs://%s/%s", reportObject.BucketName(), reportObject.ObjectName())
		reportCSVObject := client.Bucket(flakefinder.BucketName).Object(path.Join(ReportOutputPath, CreateReportFileNameWithEnding(reportBaseData.EndOfReport, merged, "csv")))
		log.Printf("Report CSV will be written to gs://%s/%s", reportCSVObject.BucketName(), reportCSVObject.ObjectName())
		reportJSONObject := client.Bucket(flakefinder.BucketName).Object(path.Join(ReportOutputPath, CreateReportFileNameWithEnding(reportBaseData.EndOfReport, merged, "json")))
		log.Printf("Report JSON will be written to gs://%s/%s", reportJSONObject.BucketName(), reportJSONObject.ObjectName())
		reportOutputWriter = reportObject.NewWriter(ctx)
		defer reportOutputWriter.Close()
		reportCSVOutputWriter = reportCSVObject.NewWriter(ctx)
		defer reportCSVOutputWriter.Close()
		reportJSONOutputWriter = reportJSONObject.NewWriter(ctx)
		defer reportJSONOutputWriter.Close()
	}
	err = DoReport(reportBaseData.JobResults, reportOutputWriter, reportCSVOutputWriter, reportJSONOutputWriter, org, repo, reportBaseData.PRNumbers, isDryRun, reportBaseData.StartOfReport, reportBaseData.EndOfReport)
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

func DoReport(results []*flakefinder.JobResult, reportOutputWriter, reportCSVOutputWriter, reportJSONOutputWriter *storage.Writer, org, repo string, prNumbers []int, isDryRun bool, startOfReport, endOfReport time.Time) error {
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
		err := json.NewEncoder(reportJSONOutputWriter).Encode(parameters)
		if err != nil {
			return fmt.Errorf("failed to write report json: %v", err)
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
		err := json.NewEncoder(os.Stdout).Encode(parameters)
		if err != nil {
			return fmt.Errorf("failed to write report json: %v", err)
		}
	}

	return nil
}
