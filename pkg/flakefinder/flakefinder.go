package flakefinder

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"path"
	"regexp"
	"strings"
	"time"

	"kubevirt.io/project-infra/pkg/flakefinder/api"

	"cloud.google.com/go/storage"
	"github.com/joshdk/go-junit"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
)

const (
	BucketName       = "kubevirt-prow"
	ReportsPath      = "reports/flakefinder"
	ReportFilePrefix = "flakefinder-"
	PreviewPath      = "preview"
)

// ListGcsObjects get the slice of gcs objects under a given path
func ListGcsObjects(ctx context.Context, client *storage.Client, bucketName, prefix, delim string) (
	[]string, error) {

	var objects []string
	it := client.Bucket(bucketName).Objects(ctx, &storage.Query{
		Prefix:    prefix,
		Delimiter: delim,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return objects, fmt.Errorf("error iterating: %v", err)
		}

		if attrs.Prefix != "" {
			objects = append(objects, path.Base(attrs.Prefix))
		}
	}
	logrus.Info("end of listGcsObjects(...)")
	return objects, nil
}

func ReadGcsObjectAttrs(ctx context.Context, client *storage.Client, bucket, object string) (attrs *storage.ObjectAttrs, err error) {
	logrus.Infof("Trying to read gcs object attrs '%s' in bucket '%s'\n", object, bucket)
	attrs, err = client.Bucket(bucket).Object(object).Attrs(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("Cannot read attrs from %s in bucket '%s'", object, bucket)
	}
	return
}

func WriteTemplateToOutput(tpl string, parameters interface{}, writer io.Writer) error {
	t, err := template.New("report").Parse(tpl)
	if err != nil {
		return fmt.Errorf("failed to load template: %v", err)
	}

	err = t.Execute(writer, parameters)
	return err
}

func CreateOutputWriter(client *storage.Client, ctx context.Context, outputPath string) io.WriteCloser {
	reportIndexObject := client.Bucket(BucketName).Object(path.Join(outputPath, "index.html"))
	log.Printf("Report index page will be written to gs://%s/%s", BucketName, reportIndexObject.ObjectName())
	reportIndexObjectWriter := reportIndexObject.NewWriter(ctx)
	return reportIndexObjectWriter
}

type ReportIntervalOptions struct {
	Today  bool
	Merged time.Duration
	Till   time.Time
}

func GetReportInterval(r ReportIntervalOptions) (startOfReport, endOfReport time.Time) {
	if r.Today {
		startOfReportToday := r.Till.Format("2006-01-02") + "T00:00:00Z"
		startOfReport, err := time.Parse(time.RFC3339, startOfReportToday)
		if err != nil {
			log.Fatalf("Failed to parse time %+v: %+v", startOfReportToday, err)
		}
		return startOfReport, r.Till
	} else {
		startOfReport = r.Till.Add(-r.Merged)
	}

	// we normalize the start of the report against start of day vs. start of the hour to avoid working against a
	// moving target.
	// In general a user would expect to find all pull requests of the previous day in a 24h report, regardless of
	// when the report has been run at the current day, which, depending on time of day when the report had been run,
	// would not always be the case.
	// Consider i.e. if the report is run late in the afternoon the user might wonder why the PR Merged in the morning
	// the day before was not included.

	var startOfDayOrHour, endOfDayOrHour string

	// in case of reports for at least a day we are fetching reports from start of previous day Till end of that day
	if r.Merged.Hours() < 24 {
		// in case of less than a day we are fetching reports from start of the hour
		startOfDayOrHour = startOfReport.Format("2006-01-02T15:00:00Z07:00")
		endOfDayOrHour = r.Till.Format("2006-01-02T15:00:00Z07:00")
	} else {
		startOfDayOrHour = startOfReport.Format("2006-01-02") + "T00:00:00Z"
		endOfDayOrHour = r.Till.Format("2006-01-02") + "T00:00:00Z"
	}
	startOfReport, err := time.Parse(time.RFC3339, startOfDayOrHour)
	if err != nil {
		log.Fatalf("Failed to parse time %+v: %+v", startOfDayOrHour, err)
	}
	endOfReport, err = time.Parse(time.RFC3339, endOfDayOrHour)
	if err != nil {
		log.Fatalf("Failed to parse time %+v: %+v", endOfDayOrHour, err)
	}
	millisecond, err := time.ParseDuration("1ms")
	if err != nil {
		log.Fatalf("Failed to parse duration 1ms: %+v", err)
	}
	endOfReport = endOfReport.Add(-millisecond)
	return startOfReport, endOfReport
}

type JobResult struct {
	Job         string
	JUnit       []junit.Suite
	BuildNumber int
	PR          int
	BatchPRs    []int
	CommitID    string
}

type ReportBaseDataOptions struct {
	prBaseBranch            string
	today                   bool
	merged                  time.Duration
	org                     string
	repo                    string
	skipBeforeStartOfReport bool
	periodicJobDirRegex     *regexp.Regexp
	batchJobDirRegex        *regexp.Regexp
}

func NewReportBaseDataOptions(
	prBaseBranch string,
	today bool,
	merged time.Duration,
	org string,
	repo string,
	skipBeforeStartOfReport bool,
) ReportBaseDataOptions {
	return ReportBaseDataOptions{prBaseBranch, today, merged, org, repo, skipBeforeStartOfReport, nil, nil}
}

// SetPeriodicJobDirRegex sets the regex to use for finding periodic job directories if the string is non empty. If the regex does not compile it will panic.
func (r *ReportBaseDataOptions) SetPeriodicJobDirRegex(regex string) {
	if regex != "" {
		r.periodicJobDirRegex = regexp.MustCompile(regex)
	}
}

func (r *ReportBaseDataOptions) SetBatchJobDirRegex(regex string) {
	if regex != "" {
		r.batchJobDirRegex = regexp.MustCompile(regex)
	}
}

type ReportBaseData struct {
	StartOfReport time.Time
	EndOfReport   time.Time
	PRNumbers     []int
	JobResults    []*JobResult
}

func GetReportBaseData(ctx context.Context, q api.Query, client *storage.Client, o ReportBaseDataOptions) ReportBaseData {

	startOfReport, endOfReport := GetReportInterval(ReportIntervalOptions{o.today, o.merged, time.Now()})
	changes, err := q.Query(ctx, startOfReport, endOfReport)
	if err != nil {
		logrus.Fatal(err)
	}

	var reports []*JobResult
	var changeNumbers []int
	for _, change := range changes {
		changeNumbers = append(changeNumbers, change.ID())
		r, err := FindUnitTestFiles(ctx, client, BucketName, strings.Join([]string{o.org, o.repo}, "/"), change, startOfReport, o.skipBeforeStartOfReport)
		if err != nil {
			log.Printf("failed to load JUnit file for %v: %v", change.ID(), err)
		}
		reports = append(reports, r...)
	}

	batchJobResults, err := FindUnitTestFilesForBatchJobs(ctx, client, BucketName, o.batchJobDirRegex, changes, startOfReport, endOfReport)
	if err != nil {
		log.Printf("failed to load JUnit file for batch jobs: %v", err)
	}
	reports = append(reports, batchJobResults...)

	if o.periodicJobDirRegex != nil {
		jobDir := "logs"
		periodicJobDirs, err := ListGcsObjects(ctx, client, BucketName, jobDir+"/", "/")
		if err != nil {
			log.Printf("failed to load periodicJobDirs for %v: %v", fmt.Sprintf("%s*", o.periodicJobDirRegex), fmt.Errorf("error listing gcs objects: %v", err))
		}

		for _, periodicJobDir := range periodicJobDirs {
			if !o.periodicJobDirRegex.MatchString(periodicJobDir) {
				continue
			}
			results, err := FindUnitTestFilesForPeriodicJob(ctx, client, BucketName, []string{jobDir, periodicJobDir}, startOfReport, endOfReport)
			if err != nil {
				log.Printf("failed to load JUnit files for job %v: %v", periodicJobDir, err)
			}
			reports = append(reports, results...)
		}
	}

	return ReportBaseData{startOfReport, endOfReport, changeNumbers, reports}
}
