package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"k8s.io/apimachinery/pkg/util/errors"
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
)

type options struct {
	since              time.Duration
	performanceJobName string
	vmMetricsList      string
	vmiMetricsList     string
	outputDir          string
}

func flagOptions() options {
	o := options{}
	flag.DurationVar(&o.since, "since", 24*7*time.Hour, "Filter the periodic job in the time window")
	flag.StringVar(&o.performanceJobName, "performance-job-name", "periodic-kubevirt-e2e-k8s-1.25-sig-performance", "usuage, name of the performance job for which data is collected")
	flag.StringVar(&o.vmMetricsList, "vm-metrics-list", string(ResultTypeVMICreationToRunningP95), "comma separated list of metrics to be extracted for vms")
	flag.StringVar(&o.vmiMetricsList, "vmi-metrics-list", string(ResultTypeVMICreationToRunningP95), "comma separated list of metrics to be extracted for vmis")
	flag.StringVar(&o.outputDir, "output-dir", "output", "the output directory were json data will be written")
	flag.Parse()
	return o
}

func main() {
	opts := flagOptions()

	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		// return fmt.Errorf("Failed to create new storage client: %v.\n", err)
	}

	jobsDirs, err := listAllRunsForJob(ctx, storageClient, opts.performanceJobName)
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
	since := time.Now().Add(-opts.since)

	// convert to perfStats
	//fmt.Print(jobsDirs)
	collection, err := extractCollectionFromLogs(ctx, storageClient, jobsDirs, since)
	if err != nil {
		log.Fatalf("error getting job collection %#+v\n", err)
	}

	//fmt.Print(collection)

	weeklyVMIResults, err := getWeeklyVMIResults(collection)
	if err != nil {
		log.Fatalf("error getting weekly vmi collection %#+v\n", err)
	}

	weeklyVMResults, err := getWeeklyVMResults(collection)
	if err != nil {
		log.Fatalf("error getting weekly vm results %#+v\n", err)
	}

	err = calculateAVGAndWriteOutput(weeklyVMIResults, "vmi", opts.outputDir, strings.Split(opts.vmiMetricsList, ",")...)
	if err != nil {
		log.Fatalf("error writing vmi avg results to json file %#+v\n", err)
	}

	err = calculateAVGAndWriteOutput(weeklyVMResults, "vm", opts.outputDir, strings.Split(opts.vmiMetricsList, ",")...)
	if err != nil {
		log.Fatalf("error writing vm avg results to json file %#+v\n", err)
	}
	// Now we can use bucket to store our representation of all results
	// This will be then used to compute nice graph

	//reportObject := storageClient.Bucket(BucketName).Object(path.Join("reports/performance", "thisShouldBeSomethingUnique"))
	//reportWriter := reportObject.NewWriter(ctx)
	//defer reportWriter.Close()
	//
	//err = json.NewEncoder(reportWriter).Encode(perfStats{})
	//if err != nil {
	//	log.Fatalf("Failed to write results to bucket. %s", err)
	//}

	log.Println("Successfully finished")
}

type YearWeek struct {
	Year int
	Week int
}

type Collection map[string]struct {
	VMIResult Result
	VMResult  Result
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
		date, err := getDateForJob(ctx, storageClient, j)
		if err != nil {
			log.Printf("error getting build-log.txt ready for job: %s, err: %#v\n", j, err)
			continue
		}

		if date.Before(since) {
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

		d := date.Format("2006-01-02T00:00:00Z00:00")
		r[d] = struct {
			VMIResult Result
			VMResult  Result
		}{
			vmiResult,
			vmResult,
		}
	}
	return r, errors.NewAggregate(errs)
}

func getWeeklyVMIResults(results Collection) (map[YearWeek][]Result, error) {
	// todo: aggregate error if needed
	//errs := []error{}
	weeklyData := map[YearWeek][]Result{}
	// loop over the original map and aggregate the values for each Week
	for dateStr, value := range results {
		date, err := time.Parse("2006-01-02", dateStr) // convert the string to a time.Time object
		if err != nil {
			return nil, err
		}

		year, week := date.ISOWeek() // get the Year and Week number of the date
		//weekStr := fmt.Sprintf("%d-W%02d", Year, Week) // format the Year and Week number as a string
		yw := YearWeek{Year: year, Week: week}
		_, ok := weeklyData[yw]
		if ok {
			weeklyData[yw] = append(weeklyData[yw], value.VMIResult)
			continue
		} // add the value to the weekly map
		weeklyData[yw] = []Result{value.VMIResult}
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

func getWeeklyVMResults(results Collection) (map[YearWeek][]Result, error) {
	// todo: aggregate error if needed
	//errs := []error{}
	weeklyData := map[YearWeek][]Result{}
	// loop over the original map and aggregate the values for each Week
	for dateStr, value := range results {
		date, err := time.Parse("2006-01-02", dateStr) // convert the string to a time.Time object
		if err != nil {
			return nil, err
		}

		year, week := date.ISOWeek() // get the Year and Week number of the date
		//weekStr := fmt.Sprintf("%d-W%02d", Year, Week) // format the Year and Week number as a string
		yw := YearWeek{Year: year, Week: week}
		_, ok := weeklyData[yw]
		if ok {
			weeklyData[yw] = append(weeklyData[yw], value.VMResult)
			continue
		}
		// add the value to the weekly map
		weeklyData[yw] = []Result{value.VMResult}
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
