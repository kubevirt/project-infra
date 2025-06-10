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

package flakestats

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

type WriteOptions struct {
	OutputFile          string
	OverwriteOutputFile bool
	OutputFormat        string
}

type ReportOption func(r *ReportOptions)

func DaysInThePast(d int) func(r *ReportOptions) {
	return func(r *ReportOptions) {
		r.DaysInThePast = d
	}
}
func FilterPeriodicJobRunResults(f bool) func(r *ReportOptions) {
	return func(r *ReportOptions) {
		r.FilterPeriodicJobRunResults = f
	}
}

func NewDefaultReportOpts(opts ...ReportOption) *ReportOptions {
	r := &ReportOptions{
		DaysInThePast:               defaultDaysInThePast,
		Org:                         defaultOrg,
		Repo:                        defaultRepo,
		FilterPeriodicJobRunResults: true,
		FilterLaneRegexString:       "",
		filterLaneRegex:             nil,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

type ReportOptions struct {
	DaysInThePast               int
	Org                         string
	Repo                        string
	FilterPeriodicJobRunResults bool
	FilterLaneRegexString       string
	filterLaneRegex             *regexp.Regexp
}
type Options struct {
	ReportOptions
	WriteOptions
}

func (o *Options) Validate() error {
	if o.DaysInThePast <= 0 {
		return fmt.Errorf("invalid value for DaysInThePast %d", o.DaysInThePast)
	}
	if o.OutputFile == "" {
		file, err := os.CreateTemp("", "flake-stats-*.html")
		if err != nil {
			return fmt.Errorf("failed to generate temp file: %w", err)
		}
		o.OutputFile = file.Name()
	} else {
		if !o.OverwriteOutputFile {
			stats, err := os.Stat(o.OutputFile)
			if stats != nil || !os.IsNotExist(err) {
				return fmt.Errorf("file %q exists or error occurred: %w", o.OutputFile, err)
			}
		}
	}
	if o.FilterLaneRegexString != "" {
		var err error
		o.filterLaneRegex, err = regexp.Compile(o.FilterLaneRegexString)
		if err != nil {
			return fmt.Errorf("failed to compile regex %q for filtering lane: %w", o.FilterLaneRegexString, err)
		}
	}
	outputFormatAllowed := false
	for _, outputFormat := range outputFormats {
		if o.OutputFormat == outputFormat {
			outputFormatAllowed = true
		}
	}
	if !outputFormatAllowed {
		return fmt.Errorf("output format %q not allowed, use one of %v", o.OutputFormat, outputFormats)
	}
	return nil
}

func ParseFlags() (*Options, error) {
	flakeStatsOptions := &Options{}

	flag.IntVar(&flakeStatsOptions.DaysInThePast, "days-in-the-past", defaultDaysInThePast, "determines how much days in the past till today are covered")
	flag.StringVar(&flakeStatsOptions.OutputFile, "output-file", "", "outputfile to write to, default is a tempfile in folder")
	flag.BoolVar(&flakeStatsOptions.OverwriteOutputFile, "overwrite-output-file", false, "whether outputfile is set to be overwritten if it exists")
	flag.StringVar(&flakeStatsOptions.Org, "Org", defaultOrg, "GitHub Org to use for fetching report data from gcs dir")
	flag.StringVar(&flakeStatsOptions.Repo, "Repo", defaultRepo, "GitHub Repo to use for fetching report data from gcs dir")
	flag.BoolVar(&flakeStatsOptions.FilterPeriodicJobRunResults, "filter-periodic-job-run-results", false, "whether results of periodic jobs should be filtered out of the report")
	flag.StringVar(&flakeStatsOptions.FilterLaneRegexString, "filter-lane-regex", "", "regex defining jobs to be filtered out of the report")
	flag.StringVar(&flakeStatsOptions.OutputFormat, "output-format", defaultOutputFormatHTML, "output format of file")
	flag.Parse()

	err := flakeStatsOptions.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate flags: %v", err)
	}

	return flakeStatsOptions, nil
}
