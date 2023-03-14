package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/util/errors"
	"log"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
)

type opts struct {
	startFrom time.Duration
}

var (
	options              = opts{}
	performanceJobsRegex = "periodic-kubevirt-e2e-k8s-1.25-sig-performance"
)

// two situations
// 1. get all the historical jobs data organized
// 2. on a regular interval run the job
func main() {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		// return fmt.Errorf("Failed to create new storage client: %v.\n", err)
	}

	allJobs, err := listAllJobs(ctx, storageClient)
	if err != nil {
		log.Fatal(err)
	}

	jobsDirs := []string{}
	for _, jobDir := range allJobs {
		if jobDir == performanceJobsRegex {
			jobsDirs = append(jobsDirs, jobDir)
		}
	}

	startOfReport, endOfReport := GetReportInterval(ReportIntervalOptions{
		Today:  true,
		Merged: 24 * time.Hour,
		Till:   time.Now(),
	})

	jobResults, err := FindUnitTestFilesForPeriodicJob(ctx,
		storageClient,
		BucketName,
		// TODO
		jobsDirs,
		startOfReport,
		endOfReport,
	)
	if err != nil {
		log.Fatalf("Failed to get job results for periodics, %s", err)
	}

	// get all the jobs for each day

	// TODO Now we should have all the perf runs
	// We need to extract the numbers
	// Insert code
	type perfStats struct {
		// whatever we need
		vmiCreationToRunningSecondsP99 float64
	}

	// todo: make this a flag
	since := time.Now().Add(-(time.Duration(24 * 30 * time.Hour)))

	// convert to perfStats
	fmt.Print(jobResults)
	results, err := getJobResults(ctx, storageClient, jobResults, since)

	fmt.Print(results)

	// Now we can use bucket to store our representation of all results
	// This will be then used to compute nice graph

	reportObject := storageClient.Bucket(BucketName).Object(path.Join("reports/performance", "thisShouldBeSomethingUnique"))
	reportWriter := reportObject.NewWriter(ctx)
	defer reportWriter.Close()

	err = json.NewEncoder(reportWriter).Encode(perfStats{})
	if err != nil {
		log.Fatalf("Failed to write results to bucket. %s", err)
	}

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

func getJobResults(ctx context.Context, storageClient *storage.Client, jobResults []*JobResult, since time.Time) (Collection, error) {
	r := Collection{}
	errs := []error{}
	for _, j := range jobResults {
		date, err := getDateForJob(ctx, storageClient, j.Job)
		if err != nil {
			log.Printf("error getting build-log.txt ready for job: %s, err: %#v\n", j.Job, err)
			continue
		}

		if date.Before(since) {
			continue
		}

		vmiResult, err := getVMIResult(ctx, storageClient, j.Job)
		if err != nil {
			errs = append(errs, err)
		}
		vmResult, err := getVMResult(ctx, storageClient, j.Job)
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

func listAllJobs(ctx context.Context, client *storage.Client) ([]string, error) {
	jobDir := "logs"
	jobDirs, err := ListGcsObjects(ctx, client, BucketName, jobDir+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("Failed to list jobs for bucket %s: %s", BucketName, jobDir)
	}
	return jobDirs, nil
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
