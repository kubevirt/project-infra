package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"k8s.io/apimachinery/pkg/util/errors"
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
)

type resultOpts struct {
	outputDir          string
	performanceJobName string
	since              time.Duration
}

type weeklyReportOpts struct {
	since          time.Duration
	resultsDir     string
	outputDir      string
	vmMetricsList  string
	vmiMetricsList string
}

type weeklyGraphOpts struct {
	metric           string
	resource         string
	weeklyReportsDir string
	plotlyHTML       bool
}

func resultsFlagOpts(subcommands []string) resultOpts {
	fs := flag.NewFlagSet("results", flag.ExitOnError)
	r := resultOpts{}
	fs.DurationVar(&r.since, "since", 24*7*time.Hour, "Filter the periodic job in the time window")
	fs.StringVar(&r.performanceJobName, "performance-job-name", "periodic-kubevirt-e2e-k8s-1.25-sig-performance", "usuage, name of the performance job for which data is collected")
	fs.StringVar(&r.outputDir, "output-dir", "output/results", "the output directory were json data will be written")
	err := fs.Parse(subcommands)
	if err != nil {
		fmt.Printf("error parsing flags: %+v\n", err)
		os.Exit(1)
	}
	return r
}

func weeklyReportFlagOpts(subcommands []string) weeklyReportOpts {
	w := weeklyReportOpts{}
	fs := flag.NewFlagSet("weekly-report", flag.ExitOnError)
	fs.DurationVar(&w.since, "since", 24*7*time.Hour, "Filter the periodic job in the time window")
	fs.StringVar(&w.resultsDir, "results-dir", "output/results/periodic-kubevirt-e2e-k8s-1.25-sig-performance", "usuage, name of the performance job for which data is collected")
	fs.StringVar(&w.vmMetricsList, "vm-metrics-list", string(ResultTypeVMICreationToRunningP95), "comma separated list of metrics to be extracted for vms")
	fs.StringVar(&w.vmiMetricsList, "vmi-metrics-list", string(ResultTypeVMICreationToRunningP95), "comma separated list of metrics to be extracted for vmis")
	fs.StringVar(&w.outputDir, "output-dir", "output/weekly", "the output directory were json data will be written")
	err := fs.Parse(subcommands)
	if err != nil {
		fmt.Printf("error parsing flags: %+v\n", err)
		os.Exit(1)
	}
	return w
}

func weeklyGraphFlagOpts(subcommands []string) weeklyGraphOpts {
	w := weeklyGraphOpts{}
	fs := flag.NewFlagSet("weekly-graph", flag.ExitOnError)
	fs.StringVar(&w.metric, "metric", string(ResultTypeVMICreationToRunningP95), "the metric for which graph will be plotted")
	fs.StringVar(&w.resource, "resource", "vmi", "resource for which the graph will be plotted")
	fs.StringVar(&w.weeklyReportsDir, "weekly-reports-dir", "output/weekly", "the output directory from which weekly json data will be read")
	fs.BoolVar(&w.plotlyHTML, "plotly-html", true, "boolean for selecting what kind of graph will be plotted")
	err := fs.Parse(subcommands)
	if err != nil {
		fmt.Printf("error parsing flags: %+v\n", err)
		os.Exit(1)
	}
	return w
}

func main() {
	fs := flag.NewFlagSet("perf-report-creator", flag.ExitOnError)
	err := fs.Parse(os.Args[1:])
	if err != nil {
		fmt.Printf("error parsing flags: %+v\n", err)
		os.Exit(1)
	}
	if len(fs.Args()) < 1 {
		fmt.Println("usuage: perf-report-creator <subcommand> [subcommand options]")
		os.Exit(1)
	}

	subCMD := fs.Arg(0)
	switch subCMD {
	case "results":
		err := runResults(resultsFlagOpts(fs.Args()[1:]))
		if err != nil {
			fmt.Printf("unable to gather results, err: %+v\n", err)
			os.Exit(1)
		}
	case "weekly-report":
		err := runWeeklyReports(weeklyReportFlagOpts(fs.Args()[1:]))
		if err != nil {
			fmt.Printf("unable to gather weekly report, err: %+v\n", err)
			os.Exit(1)
		}
	case "weekly-graph":
		err := plotWeeklyGraph(weeklyGraphFlagOpts(fs.Args()[1:]))
		if err != nil {
			fmt.Printf("unable to plot graph for given configuration, err: %+v\n", err)
			os.Exit(1)
		}
	}

	log.Println("Successfully finished")
}

func runResults(r resultOpts) error {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		// return fmt.Errorf("Failed to create new storage client: %v.\n", err)
		return err
	}

	jobsDirs, err := listAllRunsForJob(ctx, storageClient, r.performanceJobName)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: find a way to handle opts.since it would be great if there is a way to get
	//    objects after a specific timestamp

	// currently it is equivalent of zero
	// The zero value of type Time is January 1, year 1, 00:00:00.000000000 UTC.
	// As this time is unlikely to come up in practice, the IsZero method gives
	// a simple way of detecting a time that has not been initialized explicitly.
	//since := time.Date(1, 1, 0, 0, 0, 0, 0, time.UTC)
	since := time.Now().Add(-r.since)

	// convert to perfStats
	//fmt.Print(jobsDirs)
	collection, err := extractCollectionFromLogs(ctx, storageClient, jobsDirs, since)
	if err != nil {
		log.Fatalf("error getting job collection %#+v\n", err)
	}

	err = writeCollection(&collection, r.outputDir, r.performanceJobName)
	if err != nil {
		return err
	}
	return nil
}

func writeCollection(collection *Collection, outputDir string, performanceJobName string) error {
	for job, result := range *collection {
		outputPath := filepath.Join(outputDir, performanceJobName, job, "results.json")
		err := os.MkdirAll(filepath.Dir(outputPath), 0755)
		if err != nil {
			return err
		}
		f, err := os.Create(outputPath)
		if err != nil {
			return err
		}
		e := json.NewEncoder(f)
		e.SetIndent("", "  ")
		if err = e.Encode(&result); err != nil {
			return err
		}
	}
	return nil
}

func readCollection(resultsDir string) (*Collection, error) {
	dirs, err := os.ReadDir(resultsDir)
	if err != nil {
		return nil, err
	}
	c := &Collection{}
	for _, entry := range dirs {
		f, err := os.Open(filepath.Join(resultsDir, entry.Name(), "results.json"))
		if err != nil {
			return nil, err
		}
		record := struct {
			JobDirCreationTime time.Time
			VMIResult          Result
			VMResult           Result
		}{}
		d := json.NewDecoder(f)
		err = d.Decode(&record)
		if err != nil {
			return nil, err
		}
		(*c)[entry.Name()] = record
	}
	return c, nil
}

func runWeeklyReports(w weeklyReportOpts) error {
	collection, err := readCollection(w.resultsDir)
	if err != nil {
		return err
	}
	weeklyVMIResults, err := getWeeklyVMIResults(collection)
	if err != nil {
		log.Fatalf("error getting weekly vmi collection %#+v\n", err)
	}

	weeklyVMResults, err := getWeeklyVMResults(collection)
	if err != nil {
		log.Fatalf("error getting weekly vm results %#+v\n", err)
	}

	err = calculateAVGAndWriteOutput(weeklyVMIResults, "vmi", w.outputDir, strings.Split(w.vmiMetricsList, ",")...)
	if err != nil {
		log.Fatalf("error writing vmi avg results to json file %#+v\n", err)
	}

	err = calculateAVGAndWriteOutput(weeklyVMResults, "vm", w.outputDir, strings.Split(w.vmMetricsList, ",")...)
	if err != nil {
		log.Fatalf("error writing vm avg results to json file %#+v\n", err)
	}
	return nil
}

type YearWeek struct {
	Year int
	Week int
}

type Collection map[string]struct {
	JobDirCreationTime time.Time
	VMIResult          Result
	VMResult           Result
}

func listAllRunsForJob(ctx context.Context, client *storage.Client, jobName string) ([]string, error) {
	jobDir := "logs"
	jobDirs, err := ListGcsObjects(ctx, client, BucketName, jobDir+"/"+jobName+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("Failed to list jobs for bucket %s: %s", BucketName, jobDir)
	}
	return jobDirs, nil
}

func extractCollectionFromLogs(ctx context.Context, storageClient *storage.Client, jobResults []string, since time.Time) (Collection, error) {
	r := Collection{}
	errs := []error{}
	for _, j := range jobResults {
		creationTime, err := getDateForJob(ctx, storageClient, j)
		if err != nil {
			log.Printf("error getting build-log.txt ready for job: %s, err: %#v\n", j, err)
			continue
		}

		if creationTime.Before(since) {
			continue
		}

		vmiResult, err := getVMIResult(ctx, storageClient, j)
		if err != nil {
			errs = append(errs, err)
		}
		vmResult, err := getVMResult(ctx, storageClient, j)
		if err != nil {
			errs = append(errs, err)
		}

		//d := creationTime.Format("2006-01-02T00:00:00Z00:00")
		r[j] = struct {
			JobDirCreationTime time.Time
			VMIResult          Result
			VMResult           Result
		}{
			creationTime,
			vmiResult,
			vmResult,
		}
	}
	return r, errors.NewAggregate(errs)
}

func getWeeklyVMIResults(results *Collection) (map[YearWeek][]ResultWithDate, error) {
	// todo: aggregate error if needed
	//errs := []error{}
	weeklyData := map[YearWeek][]ResultWithDate{}
	// loop over the original map and aggregate the values for each Week
	for _, value := range *results {
		value := value
		year, week := value.JobDirCreationTime.ISOWeek() // get the Year and Week number of the date
		//weekStr := fmt.Sprintf("%d-W%02d", Year, Week) // format the Year and Week number as a string
		yw := YearWeek{Year: year, Week: week}
		_, ok := weeklyData[yw]
		if ok {
			weeklyData[yw] = append(weeklyData[yw], ResultWithDate{Values: value.VMIResult.Values, Date: &value.JobDirCreationTime})
			continue
		} // add the value to the weekly map
		weeklyData[yw] = []ResultWithDate{{Values: value.VMIResult.Values, Date: &value.JobDirCreationTime}}
	}
	return weeklyData, nil
}

func getDateForJob(ctx context.Context, client *storage.Client, jobID string) (time.Time, error) {
	objPath := filepath.Join("logs", jobID, "build-log.txt")

	attrs, err := ReadGcsObjectAttrs(ctx, client, BucketName, objPath)
	if err != nil {
		return time.Time{}, err
	}
	return attrs.Created, err
}

func getVMIResult(ctx context.Context, client *storage.Client, jobID string) (Result, error) {
	reader, err := getBuildLogReaderForJob(ctx, client, jobID)
	if err != nil {
		return Result{}, err
	}

	jsonText, err := readLinesAndMatchRegex(reader, "create a batch of 100 VMIs should sucessfully create all VMIS")
	if err != nil {
		return Result{}, err
	}
	return unmarshalJson(jsonText)
}

func getVMResult(ctx context.Context, client *storage.Client, jobID string) (Result, error) {
	reader, err := getBuildLogReaderForJob(ctx, client, jobID)
	if err != nil {
		return Result{}, err
	}

	lines, err := readLinesAndMatchRegex(reader, "create a batch of 100 running VMs should sucessfully create all VMS")
	if err != nil {
		return Result{}, err
	}
	lines = lines[3:]
	return unmarshalJson(lines)
}

func getBuildLogReaderForJob(ctx context.Context, client *storage.Client, jobID string) (io.Reader, error) {
	objPath := filepath.Join("logs", jobID, "build-log.txt")
	return client.Bucket(BucketName).Object(objPath).NewReader(ctx)
}

func readLinesAndMatchRegex(file io.Reader, jsonStartRegex string) (string, error) {
	//jsonStartRegex := "^\\{$"
	jsonEndRegex := "^\\}$"
	// Open the file for reading

	// Create a regular expression object
	startRegex := regexp.MustCompile(jsonStartRegex)
	endRegex := regexp.MustCompile(jsonEndRegex)

	// Create a scanner to read the file line by line
	scanner := bufio.NewScanner(file)

	// Read each line of the file and compare it against the regular expression
	for scanner.Scan() {
		line := scanner.Text()
		if startRegex.MatchString(line) {
			lines := []string{line}
			for scanner.Scan() {
				line := scanner.Text()
				if !endRegex.MatchString(line) {
					// If the regular expression matches the line, add it to the list of lines
					lines = append(lines, line)
					continue
				}
				// json end match
				lines = append(lines, line)
				break
			}
			lines = lines[3:]
			jsonText := strings.ReplaceAll(strings.Join(lines, ""), "\\n", " ")
			if err := scanner.Err(); err != nil {
				return "", err
			}
			return jsonText, nil
		}
	}

	return "", nil
}

func unmarshalJson(jsonText string) (Result, error) {
	r := Result{}
	err := json.Unmarshal([]byte(jsonText), &r)
	if err != nil {
		return Result{}, err
	}

	return r, nil
}

func getWeeklyVMResults(results *Collection) (map[YearWeek][]ResultWithDate, error) {
	// todo: aggregate error if needed
	//errs := []error{}
	weeklyData := map[YearWeek][]ResultWithDate{}
	// loop over the original map and aggregate the values for each Week
	for _, value := range *results {
		value := value
		year, week := value.JobDirCreationTime.ISOWeek() // get the Year and Week number of the date
		//weekStr := fmt.Sprintf("%d-W%02d", Year, Week) // format the Year and Week number as a string
		yw := YearWeek{Year: year, Week: week}
		_, ok := weeklyData[yw]
		if ok {
			weeklyData[yw] = append(weeklyData[yw], ResultWithDate{Values: value.VMResult.Values, Date: &value.JobDirCreationTime})
			continue
		}
		// add the value to the weekly map
		weeklyData[yw] = []ResultWithDate{{Values: value.VMResult.Values, Date: &value.JobDirCreationTime}}
	}
	return weeklyData, nil
}

func getMondayOfWeekDate(year, week int) string {
	// Get the first Monday of the Year
	firstDayOfYear := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	daysUntilFirstMonday := int(time.Monday - firstDayOfYear.Weekday())
	if daysUntilFirstMonday < 0 {
		daysUntilFirstMonday += 7
	}

	// create a time.Time object representing the Monday of the ISO Week
	weekMonday := firstDayOfYear.AddDate(0, 0, daysUntilFirstMonday+((week-1)*7))

	// print the Monday in ISO format
	return fmt.Sprintf("%s", weekMonday.Format("2006-01-02"))
}
