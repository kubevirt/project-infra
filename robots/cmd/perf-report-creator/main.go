package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path"
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

	// TODO Now we should have all the perf runs
	// We need to extract the numbers
	// Insert code
	type perfStats struct {
		// whatever we need
		vmiCreationToRunningSecondsP99 float64
	}

	// convert to perfStats
	fmt.Print(jobResults)

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

func listAllJobs(ctx context.Context, client *storage.Client) ([]string, error) {
	jobDir := "logs"
	jobDirs, err := ListGcsObjects(ctx, client, BucketName, jobDir+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("Failed to list jobs for bucket %s: %s", BucketName, jobDir)
	}
	return jobDirs, nil
}
