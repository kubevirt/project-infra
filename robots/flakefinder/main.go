package main

import (
	"context"
	"flag"
	"fmt"
	"google.golang.org/api/iterator"
	"html/template"
	"log"
	"net/url"
	"path"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
)

func flagOptions() options {
	o := options{
		endpoint: flagutil.NewStrings("https://api.github.com"),
	}
	flag.IntVar(&o.ceiling, "ceiling", 100, "Maximum number of issues to modify, 0 for infinite")
	flag.DurationVar(&o.merged, "merged", 24*7*time.Hour, "Filter to issues merged in the time window")
	flag.Var(&o.endpoint, "endpoint", "GitHub's API endpoint")
	flag.StringVar(&o.token, "token", "", "Path to github token")
	flag.StringVar(&o.graphqlEndpoint, "graphql-endpoint", github.DefaultGraphQLEndpoint, "GitHub's GraphQL API Endpoint")
	flag.Parse()
	return o
}

type options struct {
	ceiling         int
	endpoint        flagutil.Strings
	token           string
	graphqlEndpoint string
	merged          time.Duration
}

type client interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
    GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
}

const BucketName = "kubevirt-prow"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	if o.token == "" {
		log.Fatal("empty --token")
	}

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.token}); err != nil {
		log.Fatalf("Error starting secrets agent: %v", err)
	}

	var err error
	for _, ep := range o.endpoint.Strings() {
		_, err = url.ParseRequestURI(ep)
		if err != nil {
			log.Fatalf("Invalid --endpoint URL %q: %v.", ep, err)
		}
	}

	var c client = github.NewClient(secretAgent.GetTokenGenerator(o.token), o.graphqlEndpoint, o.endpoint.Strings()...)
	query, err := makeQuery("repo:kubevirt/kubevirt is:merged is:pr", o.merged)
	if err != nil {
		log.Fatalf("Bad query: %v", err)
	}
	issues, err := c.FindIssues(query, "", false)
	if err != nil {
		log.Fatalf("Failed run: %v", err)
	}

	prs := []*github.PullRequest{}
	for _, issue := range issues {
		pr, err := c.GetPullRequest("kubevirt", "kubevirt", issue.Number)
		if err != nil {
			log.Fatalf("Failed to fetch PR: %v.\n", err)
		}
		prs = append(prs, pr)
	}

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}
	reports := []*Result{}
	for _, pr := range prs {
		r, err := FindUnitTestFiles(ctx, client, BucketName, "kubevirt/kubevirt", pr)
		if err != nil {
			log.Printf("failed to load JUnit file for %v: %v", pr.Number, err)
		}
		reports = append(reports, r...)
	}


	//
	// Write report to GCS Bucket
	//

	reportFileName := fmt.Sprintf("flakefinder-%s.html", time.Now().Format("2006-01-02"))
	reportsPath := path.Join("reports", "flakefinder")
	reportObject := client.Bucket(BucketName).Object(path.Join(reportsPath, reportFileName))
	log.Printf("Report will be written to gs://%s/%s", BucketName, reportObject.ObjectName())
	reportOutputWriter := reportObject.NewWriter(ctx)
	err = Report(reports, reportOutputWriter)
	if err != nil {
		log.Fatal(fmt.Errorf("Failed on generating report: %v", err))
		//return
	}
	err = reportOutputWriter.Close()
	if err != nil {
		log.Fatal(fmt.Errorf("Failed on closing report object: %v", err))
		//return
	}

	//
	// create index.html that links to the last X reports in GCS "folder", sorted from recent to oldest
	//

	const MaxNumberOfReportsToLinkTo = 50

	// get all items from report directory
	var reportDirGcsObjects []string
	it := client.Bucket(BucketName).Objects(ctx, &storage.Query{
		Prefix:    reportsPath+"/",
		Delimiter: "/",
	})
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(fmt.Errorf("error iterating: %v", err))
			return
		}
		reportDirGcsObjects = append(reportDirGcsObjects, path.Base(attrs.Name))
	}

	// remove all non report objects by matching start of filename
	for index, fileName := range reportDirGcsObjects {
		if !strings.HasPrefix(fileName, "flakefinder-") {
			reportDirGcsObjects[index] = reportDirGcsObjects[len(reportDirGcsObjects)-1]
			reportDirGcsObjects = reportDirGcsObjects[:len(reportDirGcsObjects)-1]
		}
	}

	// keep only the X most recent
	sort.Sort(sort.Reverse(sort.StringSlice(reportDirGcsObjects)))
	if len(reportDirGcsObjects) > MaxNumberOfReportsToLinkTo {
		reportDirGcsObjects = reportDirGcsObjects[:MaxNumberOfReportsToLinkTo]
	}

	// Prepare template for index.html
	t, err := template.New("index").Parse(indexTpl)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to load report template: %v", err))
		return
	}

	var reportFiles []reportFile
	for _, reportFileName := range reportDirGcsObjects {
		date := strings.Replace(reportFileName, "flakefinder-", "", -1)
		date = strings.Replace(date, ".html", "", -1)
		reportFiles = append(reportFiles, reportFile{Date:date, FileName:reportFileName})
	}

	// Create output writer
	reportIndexObject := client.Bucket(BucketName).Object(path.Join(reportsPath, "index.html"))
	log.Printf("Report index page will be written to gs://%s/%s", BucketName, reportIndexObject.ObjectName())
	parameters := indexParams{Reports: reportFiles}
	reportIndexObjectWriter := reportIndexObject.NewWriter(ctx)

	// write index page
	err = t.Execute(reportIndexObjectWriter, parameters)
	if err != nil {
		log.Fatal(fmt.Errorf("Failed on generating index page: %v", err))
		return
	}
	err = reportIndexObjectWriter.Close()
	if err != nil {
		log.Fatal(fmt.Errorf("Failed on closing index page writer: %v", err))
		return
	}

}

func makeQuery(query string, minMerged time.Duration) (string, error) {
	parts := []string{query}
	if minMerged != 0 {
		latest := time.Now().Add(-minMerged)
		parts = append(parts, "merged:>="+latest.Format(time.RFC3339))
	}
	return strings.Join(parts, " "), nil
}
