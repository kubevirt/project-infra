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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"cloud.google.com/go/storage"
	"github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/pkg/flakefinder"
)

const template = `
<!DOCTYPE html>
<html lang="en">
	<head>
		<meta charset="UTF-8">
		<title>flakefinder projects overview</title>
		<style>
		table, th, td {
			border: 1px solid black;
		}
		</style>
	</head>
	<body>
		<h1>Projects</h1>
		<table>
			<tr>
				<th>Project</th>
			</tr>{{ range $col, $reportDir := $.ReportDirs }}
			<tr>
				<td><a href="{{ $reportDir }}/index.html">{{ $reportDir }}</a></td>
			</tr>{{ end }}
		</table>
		<div>Page created at: {{ $.Date }}</div>
	</body>
</html>
`

type IndexParams struct {
	ReportDirs []string
	Date       string
}

func flagOptions() options {
	o := options{}
	flag.BoolVar(&o.isDryRun, "dry-run", true, "Whether index page should be written to target directory or just printed to console")
	flag.Parse()
	return o
}

type options struct {
	isDryRun bool
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}

	reportDirs, err := getReportDirectories(ctx, client)
	if err != nil {
		log.Fatalf("error listing gcs objects: %v", err)
	}
	logrus.Infof("Report Directories: %v", reportDirs)

	date := time.Now().Format("2006-01-02 15:04:05")
	params := IndexParams{ReportDirs: reportDirs, Date: date}
	if o.isDryRun {
		err = flakefinder.WriteTemplateToOutput(template, params, os.Stdout)
		if err != nil {
			log.Fatalf("error writing report output: %v", err)
		}
	} else {
		reportIndexObjectWriter := flakefinder.CreateOutputWriter(client, ctx, flakefinder.ReportsPath)
		err = flakefinder.WriteTemplateToOutput(template, params, reportIndexObjectWriter)
		if err != nil {
			log.Fatalf("error writing report output: %v", err)
		}
		reportIndexObjectWriter.Close()
	}
}

func getReportDirectories(ctx context.Context, client *storage.Client) (reportDirs []string, err error) {
	directories, err := flakefinder.ListGcsObjects(ctx, client, flakefinder.BucketName, flakefinder.ReportsPath+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("error listing gcs objects: %v", err)
	}
	for _, partialDir := range directories {
		if partialDir == "preview" {
			continue
		}
		orgDir := filepath.Join(flakefinder.ReportsPath, partialDir)
		subdirectories, err := flakefinder.ListGcsObjects(ctx, client, flakefinder.BucketName, orgDir+"/", "/")
		if err != nil {
			return nil, fmt.Errorf("error listing gcs objects: %v", err)
		}
		for _, subdirectory := range subdirectories {
			repoDir := filepath.Join(orgDir, subdirectory)
			_, err := flakefinder.ReadGcsObjectAttrs(ctx, client, flakefinder.BucketName, filepath.Join(repoDir, "index.html"))
			if err == storage.ErrObjectNotExist {
				continue
			}
			reportDirs = append(reportDirs, fmt.Sprintf("%s/%s", partialDir, subdirectory))
		}
	}
	sort.Strings(reportDirs)
	return reportDirs, nil
}
