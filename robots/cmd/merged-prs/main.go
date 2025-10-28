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
	Items []Item `json:"items"`
}

func main() {
	startDate := flag.String("start-date", time.Now().AddDate(0, 0, -7).Format("2006-01-02"), "Start date for PR search, format YYYY-MM-DD")
	authors := flag.String("authors", "dhiller,brianmcarey", "Comma-separated list of GitHub author handles")
	flag.Parse()

	authorQueries := []string{}
	for _, author := range strings.Split(*authors, ",") {
		authorQueries = append(authorQueries, "author:"+strings.TrimSpace(author))
	}

	query := fmt.Sprintf("is:pr is:merged merged:>=%s %s org:kubevirt", *startDate, strings.Join(authorQueries, " "))
	reqURL := fmt.Sprintf("%s?q=%s", githubAPIURL, url.QueryEscape(query))

	resp, err := http.Get(reqURL)
	if err != nil {
		fmt.Printf("Error making request to GitHub API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: GitHub API returned status code %d\n", resp.StatusCode)
		return
	}

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("Error decoding JSON response: %v\n", err)
		return
	}

	fmt.Printf("recently merged PRs authored by SIG CI (query: “is:pr is:merged merged:>=%s %s”)\n\n", *startDate, strings.Join(authorQueries, " "))

	for _, item := range result.Items {
		repoPath := strings.TrimPrefix(item.RepositoryURL, "https://api.github.com/repos/")
		fmt.Printf("* [%s#%d](%s): %s (by @%s)\n", repoPath, item.Number, item.HTMLURL, item.Title, item.User.Login)
	}
}
