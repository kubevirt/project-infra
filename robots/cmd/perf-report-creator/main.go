package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "kubevirt.io/project-infra/pkg/flakefinder"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/util/errors"
)

type resultOpts struct {
	outputDir          string
	performanceJobName string
	since              time.Duration
	credentialsFile    string
}

type weeklyReportOpts struct {
	since          time.Duration
	resultsDir     string
	outputDir      string
	vmMetricsList  string
	vmiMetricsList string
}

type weeklyGraphOpts struct {
	metricList       string
	resource         string
	weeklyReportsDir string
	plotlyHTML       bool
	isDuringRelease  bool
	since            string
	releaseConfig    string
}

func resultsFlagOpts(subcommands []string) resultOpts {
	fs := flag.NewFlagSet("results", flag.ExitOnError)
	r := resultOpts{}
	fs.DurationVar(&r.since, "since", 24*time.Hour, "Filter the periodic job in the time window")
	fs.StringVar(&r.performanceJobName, "performance-job-name", "periodic-kubevirt-e2e-k8s-1.25-sig-performance", "usuage, name of the performance job for which data is collected")
	fs.StringVar(&r.outputDir, "output-dir", "output/results", "the output directory were json data will be written")
	fs.StringVar(&r.credentialsFile, "credentials-file", "", "the credentials json file for GCS storage client")
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
	fs.DurationVar(&w.since, "since", 24*time.Hour, "Filter the periodic job in the time window")
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
	fs.StringVar(&w.metricList, "metrics-list", string(ResultTypeVMICreationToRunningP95), "comma separated list of metrics to be plotted")
	fs.BoolVar(&w.isDuringRelease, "is-during-release", false, "boolean for selecting if the graph is plotted during a release")
	fs.StringVar(&w.resource, "resource", "vmi", "resource for which the graph will be plotted")
	fs.StringVar(&w.weeklyReportsDir, "weekly-reports-dir", "output/weekly", "the output directory from which weekly json data will be read")
	fs.BoolVar(&w.plotlyHTML, "plotly-html", true, "boolean for selecting what kind of graph will be plotted")
	fs.StringVar(&w.since, "since", "", "Specify the date (format: yyyy-mm-dd)")
	fs.StringVar(&w.releaseConfig, "release-config", "./robots/cmd/perf-report-creator/shape.yaml", "Path to release configuration file (contains shape.yaml)")
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
	// todo: make the credentials file a flag
	var storageClient *storage.Client
	var err error
	if r.credentialsFile == "" {
		storageClient, err = storage.NewClient(ctx)
	} else {
		storageClient, err = storage.NewClient(ctx, option.WithCredentialsFile(r.credentialsFile))
	}
	if err != nil {
		return fmt.Errorf("Failed to create new storage client: %v.\n", err)
	}

	jobsDirs, err := listAllRunsForJob(ctx, storageClient, r.performanceJobName)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: find a way to handle opts.since it would be great if there is a way to get
	//    objects after a specific timestamp
	since := time.Now().Add(-r.since)

	// convert to perfStats
	collection, err := extractCollectionFromAuditFiles(ctx, storageClient, jobsDirs, since, r.performanceJobName)
	if err != nil {
		log.Printf("error getting job collection %+v\n", err)
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
			VMIResult          *Result
			VMResult           *Result
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
	VMIResult          *Result
	VMResult           *Result
}

func listAllRunsForJob(ctx context.Context, client *storage.Client, jobName string) ([]string, error) {
	jobDir := "logs"
	jobDirs, err := ListGcsObjects(ctx, client, BucketName, jobDir+"/"+jobName+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("Failed to list jobs for bucket %s: %s", BucketName, jobDir)
	}
	return jobDirs, nil
}

func extractCollectionFromAuditFiles(ctx context.Context, storageClient *storage.Client, jobResults []string, since time.Time, performanceJobName string) (Collection, error) {
	r := Collection{}
	errs := []error{}
	for _, j := range jobResults {
		creationTime, err := getDateForJob(ctx, storageClient, j, performanceJobName)
		if err != nil {
			log.Printf("error getting build-log.txt ready for job: %s, err: %#v\n", j, err)
			continue
		}

		if creationTime.Before(since) {
			log.Printf("job: %s, before creation time. %v, %v\n", j, creationTime, since)
			continue
		}

		vmiResult, err := getVMIResult(ctx, storageClient, j, performanceJobName)
		if err != nil {
			log.Printf("job: %s, error getting VMI Result. %+v\n", j, err)
			errs = append(errs, err)
		}
		var vmResult *Result
		if !strings.Contains(performanceJobName, "density") {
			vmResult, err = getVMResult(ctx, storageClient, j, performanceJobName)
			if err != nil {
				log.Printf("job: %s, error getting VM Result. %+v\n", j, err)
				errs = append(errs, err)
			}
		}

		r[j] = struct {
			JobDirCreationTime time.Time
			VMIResult          *Result
			VMResult           *Result
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
		yw := YearWeek{Year: year, Week: week}
		_, ok := weeklyData[yw]
		if value.VMIResult == nil {
			log.Printf("VMIResult for date %s is empty, skipping\n", value.JobDirCreationTime)
			continue
		}
		if ok {
			weeklyData[yw] = append(weeklyData[yw], ResultWithDate{Values: value.VMIResult.Values, Date: &value.JobDirCreationTime})
			continue
		} // add the value to the weekly map
		weeklyData[yw] = []ResultWithDate{{Values: value.VMIResult.Values, Date: &value.JobDirCreationTime}}
	}
	return weeklyData, nil
}

func getDateForJob(ctx context.Context, client *storage.Client, jobID string, performanceJobName string) (time.Time, error) {
	objPath := filepath.Join("logs", performanceJobName, jobID, "build-log.txt")

	attrs, err := ReadGcsObjectAttrs(ctx, client, BucketName, objPath)
	if err != nil {
		return time.Time{}, err
	}
	return attrs.Created, err
}

func getVMIResult(ctx context.Context, client *storage.Client, jobID string, performanceJobName string) (*Result, error) {
	prefixedFileName := ""
	if strings.Contains(performanceJobName, "density") {
		prefixedFileName = "perfscale-audit-results.json"
	} else {
		prefixedFileName = "VMI-perf-audit-results.json"
	}
	reader, err := getAuditFileReaderForJob(ctx, client, jobID, performanceJobName, prefixedFileName)
	if err != nil {
		return nil, err
	}

	return getResult(reader)
}

func getVMResult(ctx context.Context, client *storage.Client, jobID string, performanceJobName string) (*Result, error) {
	reader, err := getAuditFileReaderForJob(ctx, client, jobID, performanceJobName, "performance/VM-perf-audit-results.json")
	if err != nil {
		log.Printf("job: %s, error getting BuildLogReaderForJob. %+v\n", jobID, err)
		return nil, err
	}

	return getResult(reader)
}

func getResult(reader io.Reader) (*Result, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	r := &Result{}
	err = json.Unmarshal(data, r)
	if err != nil {
		log.Printf("error unmarshaling json: %+v\ntext: %v\n", err, string(data))
		return nil, err
	}
	return r, nil
}

func getAuditFileReaderForJob(ctx context.Context, client *storage.Client, jobID, performanceJobName, prefixedFileName string) (io.Reader, error) {
	objPath := filepath.Join("logs", performanceJobName, jobID, "artifacts", prefixedFileName)
	return client.Bucket(BucketName).Object(objPath).NewReader(ctx)
}

func getWeeklyVMResults(results *Collection) (map[YearWeek][]ResultWithDate, error) {
	// todo: aggregate error if needed
	//errs := []error{}
	weeklyData := map[YearWeek][]ResultWithDate{}
	// loop over the original map and aggregate the values for each Week
	for _, value := range *results {
		value := value
		year, week := value.JobDirCreationTime.ISOWeek() // get the Year and Week number of the date
		yw := YearWeek{Year: year, Week: week}
		_, ok := weeklyData[yw]
		if value.VMResult == nil {
			log.Printf("VMResult for date %s is empty, skipping\n", value.JobDirCreationTime)
			continue
		}
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
	return weekMonday.Format("2006-01-02")
}
