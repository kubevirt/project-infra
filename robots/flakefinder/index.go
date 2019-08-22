package main

import (
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"html/template"
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
			<th>flakefinder reports</th>
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

// CreateReportIndex creates an index.html that links to the X most recent reports in GCS "folder", sorted from most
// recent to oldest
func CreateReportIndex(ctx context.Context, client *storage.Client) (err error) {

	reportDirGcsObjects, err := getReportItemsFromBucketDirectory(client, ctx)
	if err != nil {
		return fmt.Errorf("failed to get report items: %v", err)
	}

	// Prepare template for index.html
	t, err := template.New("index").Parse(indexTpl)
	if err != nil {
		return fmt.Errorf("failed to load report template: %v", err)
	}

	// Prepare data for template
	var reportFiles []reportFile
	for _, reportFileName := range reportDirGcsObjects {
		date := strings.Replace(reportFileName, ReportFilePrefix, "", -1)
		date = strings.Replace(date, ".html", "", -1)
		reportFiles = append(reportFiles, reportFile{Date: date, FileName: reportFileName})
	}

	// Create output writer
	reportIndexObject := client.Bucket(BucketName).Object(path.Join(ReportsPath, "index.html"))
	log.Printf("Report index page will be written to gs://%s/%s", BucketName, reportIndexObject.ObjectName())
	parameters := indexParams{Reports: reportFiles}
	reportIndexObjectWriter := reportIndexObject.NewWriter(ctx)

	// write index page
	err = t.Execute(reportIndexObjectWriter, parameters)
	if err != nil {
		return fmt.Errorf("failed generating index page: %v", err)
	}
	err = reportIndexObjectWriter.Close()
	if err != nil {
		return fmt.Errorf("failed closing index page writer: %v", err)
	}
	return nil
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
	// remove all non report objects by matching start of filename
	for index, fileName := range reportDirGcsObjects {
		if !strings.HasPrefix(fileName, ReportFilePrefix) {
			reportDirGcsObjects[index] = reportDirGcsObjects[len(reportDirGcsObjects)-1]
			reportDirGcsObjects = reportDirGcsObjects[:len(reportDirGcsObjects)-1]
		}
	}
	// keep only the X most recent
	sort.Sort(sort.Reverse(sort.StringSlice(reportDirGcsObjects)))
	if len(reportDirGcsObjects) > MaxNumberOfReportsToLinkTo {
		reportDirGcsObjects = reportDirGcsObjects[:MaxNumberOfReportsToLinkTo]
	}
	return reportDirGcsObjects, nil
}