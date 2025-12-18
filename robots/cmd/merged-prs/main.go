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
	"encoding/json"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	githubAPIURL = "https://api.github.com/search/issues"
)

type User struct {
	Login string `json:"login"`
}

type Item struct {
	Number        int    `json:"number"`
	HTMLURL       string `json:"html_url"`
	Title         string `json:"title"`
	User          User   `json:"user"`
	RepositoryURL string `json:"repository_url"`
}

type SearchResult struct {
	TotalCount int    `json:"total_count"`
	Items      []Item `json:"items"`
}

func main() {
	startDate := flag.String("start-date", time.Now().AddDate(0, 0, -7).Format("2006-01-02"), "Start date for PR search, format YYYY-MM-DD")
	authors := flag.String("authors", "dhiller,dollierp", "Comma-separated list of GitHub author handles")
	flag.Parse()

	query, searchResults, err := queryMergedPRsForAuthors(authors, startDate)
	if err != nil {
		log.WithError(err).Fatal()
	}

	generateMarkdownForMergedPRs(query, searchResults)
}

func queryMergedPRsForAuthors(authors *string, startDate *string) (query string, searchResults []Item, err error) {
	var authorQueries []string
	for _, author := range strings.Split(*authors, ",") {
		authorQueries = append(authorQueries, "author:"+strings.TrimSpace(author))
	}

	query = fmt.Sprintf("is:pr is:merged merged:>=%s %s org:kubevirt", *startDate, strings.Join(authorQueries, " "))
	perPage := 20
	totalCount := perPage + 1
	page := 0

	for totalCount > page*perPage {
		page++

		var result SearchResult
		reqURL := fmt.Sprintf("%s?page=%d&per_page=%d&q=%s", githubAPIURL, page, perPage, url.QueryEscape(query))
		result, err = queryPageOfMergedPRsForAuthors(reqURL)
		if err != nil {
			return reqURL, nil, err
		}
		totalCount = result.TotalCount
		searchResults = append(searchResults, result.Items...)
	}
	return
}

func queryPageOfMergedPRsForAuthors(reqURL string) (SearchResult, error) {
	log.Debugf("GitHub query: %q", reqURL)
	resp, err := http.Get(reqURL)
	if err != nil {
		return SearchResult{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return SearchResult{}, fmt.Errorf("GitHub API returned status code %d", resp.StatusCode)
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return SearchResult{}, fmt.Errorf("error decoding JSON response: %v", err)
	}
	return result, nil
}

func generateMarkdownForMergedPRs(query string, searchResults []Item) {
	fmt.Printf("* %d recently merged PRs authored by SIG CI (query: %s”)\n\n", len(searchResults), query)
	for _, item := range searchResults {
		repoPath := strings.TrimPrefix(item.RepositoryURL, "https://api.github.com/repos/")
		fmt.Printf("  * [%s#%d](%s): %s (by @%s)\n", repoPath, item.Number, item.HTMLURL, item.Title, item.User.Login)
	}
}
