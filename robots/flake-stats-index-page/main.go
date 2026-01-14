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
 * Copyright the KubeVirt Authors.
 *
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// File represents each file with its URL and date.
type File struct {
	URL  string
	Date time.Time
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>File Links Calendar</title>
    <style>
        table {
            width: 100%;
            border-collapse: collapse;
            margin-bottom: 20px;
        }
        th, td {
            border: 1px solid #ccc;
            padding: 8px;
            text-align: center;
        }
        th {
            background-color: #f2f2f2;
        }
        .link {
            color: blue;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <h1>Flake Stats Calendar</h1>
    {{range .}}
    <h2>{{.MonthName}} {{.Year}}</h2>
    <table>
        <tr>
            <th>Sun</th>
            <th>Mon</th>
            <th>Tue</th>
            <th>Wed</th>
            <th>Thu</th>
            <th>Fri</th>
            <th>Sat</th>
        </tr>
        {{range .Weeks}}
        <tr>
            {{range .}}
            <td>
                {{if .URL}}
                    <a href="{{.URL}}" class="link">{{.Date.Day}}</a>
                {{else}}
                    &nbsp;
                {{end}}
            </td>
            {{end}}
        </tr>
        {{end}}
    </table>
    {{end}}
</body>
</html>
`

type Weekday struct {
	Date time.Time
	URL  string
}

type Calendar struct {
	MonthName string
	Year      int
	Weeks     [][]Weekday
	Month     int
}

type options struct {
	outputFilePath string
}

func (opts options) validate() error {
	if opts.outputFilePath == "" {
		return fmt.Errorf("no output file path provided")
	}
	_, err := os.Stat(opts.outputFilePath)
	if err == nil {
		return fmt.Errorf("file %q exists", opts.outputFilePath)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

var o = options{}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
}

func main() {
	flag.StringVar(&o.outputFilePath, "output-file-path", "index.html", "path to test file")
	flag.Parse()

	if err := o.validate(); err != nil {
		log.Fatalf("invalid flags: %v", err)
	}

	files, err := listFilesFromGCS("kubevirt-prow", "reports/flakefinder/kubevirt/kubevirt/", "flake-stats-14days")
	if err != nil {
		log.Fatalf("Error retrieving files from GCS: %v", err)
	}

	calendars := createCalendars(files)

	f, err := os.Create(o.outputFilePath)
	if err != nil {
		log.Fatalf("Error creating HTML file: %v", err)
	}
	defer f.Close()

	tmpl := template.Must(template.New("htmlPage").Parse(htmlTemplate))
	if err := tmpl.Execute(f, calendars); err != nil {
		log.Fatalf("Error executing template: %v", err)
	}
	log.Infof("HTML file generated successfully: %q", o.outputFilePath)
}

// listFilesFromGCS retrieves files from the GCS bucket matching the pattern.
func listFilesFromGCS(bucketName, prefix, pattern string) ([]File, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}
	defer client.Close()

	bucket := client.Bucket(bucketName)
	query := &storage.Query{Prefix: prefix}
	it := bucket.Objects(ctx, query)

	var files []File
	dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)

	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error iterating over objects: %w", err)
		}

		if strings.Contains(objAttrs.Name, pattern) && strings.HasSuffix(objAttrs.Name, ".html") {
			// Extract date from the file name.
			matches := dateRegex.FindStringSubmatch(objAttrs.Name)
			if len(matches) < 2 {
				continue
			}

			date, err := time.Parse("2006-01-02", matches[1])
			if err != nil {
				log.Infof("Error parsing date in file name: %s\n", err)
				continue
			}

			fileURL := fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucketName, objAttrs.Name)

			files = append(files, File{
				URL:  fileURL,
				Date: date,
			})
		}
	}

	return files, nil
}

// createCalendars generates a list of Calendar structs, one for each unique month, sorted in chronological order.
func createCalendars(files []File) []Calendar {
	fileURLsGroupedByYearByMonthByDay := make(map[int]map[int]map[int]string)
	for _, file := range files {
		year, month, day := file.Date.Year(), int(file.Date.Month()), file.Date.Day()
		if _, ok := fileURLsGroupedByYearByMonthByDay[year]; !ok {
			fileURLsGroupedByYearByMonthByDay[year] = make(map[int]map[int]string)
		}
		if _, ok := fileURLsGroupedByYearByMonthByDay[year][month]; !ok {
			fileURLsGroupedByYearByMonthByDay[year][month] = make(map[int]string)
		}
		fileURLsGroupedByYearByMonthByDay[year][month][day] = file.URL
	}

	var calendarList []Calendar
	for year, months := range fileURLsGroupedByYearByMonthByDay {
		for month := range months {
			calendar := createCalendar(fileURLsGroupedByYearByMonthByDay[year][month], time.Month(month), year)
			calendarList = append(calendarList, calendar)
		}
	}

	// Sort in descending timely order
	sort.Slice(calendarList, func(i, j int) bool {
		if calendarList[i].Year == calendarList[j].Year {
			return calendarList[i].Month > calendarList[j].Month
		}
		return calendarList[i].Year > calendarList[j].Year
	})

	return calendarList
}

// createCalendar creates a Calendar struct with weeks and days based on the provided month and year.
func createCalendar(fileMap map[int]string, month time.Month, year int) Calendar {
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, -1)

	var weeks [][]Weekday
	var week []Weekday

	// Fill the initial empty cells up to the first day of the month.
	for i := 0; i < int(firstDay.Weekday()); i++ {
		week = append(week, Weekday{})
	}

	for day := firstDay; !day.After(lastDay); day = day.AddDate(0, 0, 1) {
		weekday := Weekday{Date: day, URL: fileMap[day.Day()]}
		week = append(week, weekday)
		if day.Weekday() == time.Saturday || day == lastDay {
			weeks = append(weeks, week)
			week = []Weekday{}
		}
	}

	return Calendar{
		MonthName: firstDay.Month().String(),
		Month:     int(firstDay.Month()),
		Year:      year,
		Weeks:     weeks,
	}
}
