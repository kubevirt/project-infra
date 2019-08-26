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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/storage"
	junit "github.com/joshdk/go-junit"
	"github.com/sirupsen/logrus"
	"google.golang.org/api/iterator"
	"k8s.io/test-infra/prow/github"
)

const (
	//finishedJSON is the JSON file that stores build success info
	finishedJSON = "finished.json"
	startedJSON  = "started.json"
)

//listGcsObjects get the slice of gcs objects under a given path
func listGcsObjects(ctx context.Context, client *storage.Client, bucketName, prefix, delim string) (
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

func readGcsObject(ctx context.Context, client *storage.Client, bucket, object string) ([]byte, error) {
	logrus.Infof("Trying to read gcs object '%s' in bucket '%s'\n", object, bucket)
	o := client.Bucket(bucket).Object(object)
	reader, err := o.NewReader(ctx)
	if err == storage.ErrObjectNotExist {
		return nil, err
	} else if err != nil {
		return nil, fmt.Errorf("cannot read object '%s': %v", object, err)
	}
	return ioutil.ReadAll(reader)
}

func FindUnitTestFiles(ctx context.Context, client *storage.Client, bucket, repo string, pr *github.PullRequest) ([]*Result, error) {

	dirOfPrJobs := path.Join("pr-logs", "pull", strings.ReplaceAll(repo, "/", "_"), strconv.Itoa(pr.Number))

	prJobsDirs, err := listGcsObjects(ctx, client, bucket, dirOfPrJobs+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("error listing gcs objects: %v", err)
	}

	junits := []*Result{}
	for _, job := range prJobsDirs {
		junit, err := FindUnitTestFileForJob(ctx, client, bucket, dirOfPrJobs, job, pr)
		if err != nil {
			return nil, err
		}
		if junit != nil {
			junits = append(junits, junit...)
		}
	}
	return junits, err
}

func FindUnitTestFileForJob(ctx context.Context, client *storage.Client, bucket string, dirOfPrJobs string, job string, pr *github.PullRequest) ([]*Result, error) {
	dirOfJobs := path.Join(dirOfPrJobs, job)

	prJobs, err := listGcsObjects(ctx, client, bucket, dirOfJobs+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("error listing gcs objects: %v", err)
	}
	builds := sortBuilds(prJobs)
	profilePath := ""
	buildNumber := 0
	reports := []*Result{}
	for _, build := range builds {
		buildDirPath := path.Join(dirOfJobs, strconv.Itoa(build))
		dirOfFinishedJSON := path.Join(buildDirPath, finishedJSON)
		dirOfStartedJSON := path.Join(buildDirPath, startedJSON)

		_, err := readGcsObject(ctx, client, bucket, dirOfFinishedJSON)
		if err == storage.ErrObjectNotExist {
			// build still running?
			continue
		} else if err != nil {
			return nil, fmt.Errorf("Cannot read finished.json (%s) in bucket '%s'", dirOfFinishedJSON, bucket)
		} else {
			startedJSON, err := readGcsObject(ctx, client, bucket, dirOfStartedJSON)
			if err != nil {
				return nil, fmt.Errorf("Cannot read started.json (%s) in bucket '%s'", dirOfStartedJSON, bucket)
			}
			if !isLatestCommit(startedJSON, pr) {
				break
			}
			buildNumber = build
			artifactsDirPath := path.Join(buildDirPath, "artifacts")
			profilePath = path.Join(artifactsDirPath, "junit.functest.xml")
			data, err := readGcsObject(ctx, client, bucket, profilePath)
			if err == storage.ErrObjectNotExist {
				return nil, nil
			}
			if err != nil {
				return nil, err
			}
			report, err := junit.Ingest(data)
			if err != nil {
				return nil, err
			}
			reports = append(reports, &Result{Job: job, JUnit: report, BuildNumber: buildNumber, PR: pr.Number})
		}
	}

	return reports, nil
}

// sortBuilds converts all build from str to int and sorts all builds in descending order and
// returns the sorted slice
func sortBuilds(strBuilds []string) []int {
	var res []int
	for _, buildStr := range strBuilds {
		num, err := strconv.Atoi(buildStr)
		if err != nil {
			logrus.Infof("Non-int build number found: '%s'", buildStr)
		} else {
			res = append(res, num)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(res)))
	return res
}

type finishedStatus struct {
	Timestamp int
	Passed    bool
}

// {"timestamp":1562772668,"pull":"2473","repo-version":"f3bb83f4377b8b45bd47d33373edfacf85361f0e","repos":{"kubevirt/kubevirt":"release-0.13:577e95c340e1b21ff431cbba25ad33c891554e38,2473:8c33c116def661c69b4a8eb08fac9ca07dfbf03c"}}
type startedStatus struct {
	Timestamp int
	Repos     map[string]string
}

type Result struct {
	Job         string
	JUnit       []junit.Suite
	BuildNumber int
	PR          int
}

func isLatestCommit(jsonText []byte, pr *github.PullRequest) bool {
	var status startedStatus
	err := json.Unmarshal(jsonText, &status)
	for _, v := range status.Repos {
		return err == nil && strings.Contains(v, fmt.Sprintf("%d:%s", pr.Number, pr.Head.SHA))
	}
	return false
}
