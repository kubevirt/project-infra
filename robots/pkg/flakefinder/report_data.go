package flakefinder

import (
	"fmt"
	"sort"
	"time"

	"github.com/joshdk/go-junit"
)

type Params struct {
	StartOfReport   string
	EndOfReport     string
	Headers         []string
	Tests           []string
	Data            map[string]map[string]*Details
	PrNumbers       []int
	Org             string
	Repo            string
	FailuresForJobs map[string]*JobFailures
}

type Details struct {
	Succeeded int    `json:"succeeded"`
	Skipped   int    `json:"skipped"`
	Failed    int    `json:"failed"`
	Severity  string `json:"severity"`
	Jobs      []*Job `json:"jobs"`
}

type Job struct {
	BuildNumber int    `json:"buildNumber"`
	Severity    string `json:"severity"`
	PR          int    `json:"pr"`
	Job         string `json:"job"`
}

type JobFailures struct {
	BuildNumber int    `json:"buildNumber"`
	PR          int    `json:"pr"`
	Job         string `json:"job"`
	Failures    int    `json:"failures"`
}

type Jobs []*Job

func (jobs Jobs) Len() int      { return len(jobs) }
func (jobs Jobs) Swap(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] }

type ByBuildNumber struct{ Jobs }

func (b ByBuildNumber) Less(i, j int) bool { return b.Jobs[i].BuildNumber < b.Jobs[j].BuildNumber }

func CreateFlakeReportData(results []*JobResult, prNumbers []int, endOfReport time.Time, org string, repo string, startOfReport time.Time) Params {
	data := map[string]map[string]*Details{}
	headers := []string{}
	tests := []string{}
	failuresForJobs := map[string]*JobFailures{}
	headerMap := map[string]struct{}{}

	createFailuresForJobsKey := func(result *JobResult) string {
		return fmt.Sprintf("%s-%d", result.Job, result.BuildNumber)
	}

	for _, result := range results {

		// first find failing tests to isolate tests which interest us
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if test.Status == junit.StatusFailed || test.Status == junit.StatusError {

					failuresForJobsKey := createFailuresForJobsKey(result)
					_, exists := failuresForJobs[failuresForJobsKey]
					if !exists {
						failuresForJobs[failuresForJobsKey] = &JobFailures{
							BuildNumber: result.BuildNumber,
							PR:          result.PR,
							Job:         result.Job,
							Failures:    0,
						}
					}
					failuresForJobs[failuresForJobsKey].Failures = failuresForJobs[failuresForJobsKey].Failures + 1

					testEntry := data[test.Name]
					if testEntry == nil {
						tests = append(tests, test.Name)
						testEntry = map[string]*Details{}
						data[test.Name] = testEntry
					}

					if _, exists := testEntry[result.Job]; !exists {
						testEntry[result.Job] = &Details{}
					}
					if _, exists := headerMap[result.Job]; !exists {
						headerMap[result.Job] = struct{}{}
						headers = append(headers, result.Job)
					}
					testEntry[result.Job].Failed = testEntry[result.Job].Failed + 1
				}
			}
		}

	}

	// second enrich failed tests with additional information
	for _, result := range results {
		if _, exists := failuresForJobs[createFailuresForJobsKey(result)]; !exists {
			// if not in the map now, then skip it
			continue
		}
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if _, exists := data[test.Name]; !exists {
					continue
				}
				if _, exists := data[test.Name][result.Job]; !exists {
					data[test.Name][result.Job] = &Details{}
				}
				if test.Status == junit.StatusSkipped {
					data[test.Name][result.Job].Skipped = data[test.Name][result.Job].Skipped + 1
				} else if test.Status == junit.StatusPassed {
					data[test.Name][result.Job].Succeeded = data[test.Name][result.Job].Succeeded + 1
					data[test.Name][result.Job].Jobs = append(data[test.Name][result.Job].Jobs, &Job{Severity: "green", BuildNumber: result.BuildNumber, Job: result.Job, PR: result.PR})
				} else {
					data[test.Name][result.Job].Jobs = append(data[test.Name][result.Job].Jobs, &Job{Severity: "red", BuildNumber: result.BuildNumber, Job: result.Job, PR: result.PR})
				}
			}
		}
	}

	// third, calculate the severity
	// second enrich failed tests with additional information
	for _, result := range results {
		if _, exists := failuresForJobs[createFailuresForJobsKey(result)]; !exists {
			// if not in the map now, then skip it
			continue
		}
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if _, exists := data[test.Name]; !exists {
					continue
				}
				if _, exists := data[test.Name][result.Job]; !exists {
					continue
				}

				entry := data[test.Name][result.Job]

				SetSeverity(entry)
			}
		}
	}

	sort.Strings(headers)

	for _, jobsByNames := range data {
		for _, details := range jobsByNames {
			sort.Sort(ByBuildNumber{details.Jobs})
		}
	}

	testsSortedByRelevance := SortTestsByRelevance(data, tests)
	parameters := Params{
		Data:            data,
		Headers:         headers,
		Tests:           testsSortedByRelevance,
		PrNumbers:       prNumbers,
		EndOfReport:     endOfReport.Format(time.RFC3339),
		Org:             org,
		Repo:            repo,
		StartOfReport:   startOfReport.Format(time.RFC3339),
		FailuresForJobs: failuresForJobs,
	}
	return parameters
}

// SetSeverity sets the field Severity on the passed details according to the ratio of failed vs succeeded tests,
// where test results are the more severe the more test failures they contain in relation to succeeded tests.
func SetSeverity(entry *Details) {
	var ratio float32 = 1.0
	if entry.Succeeded > 0 {
		ratio = float32(entry.Failed) / float32(entry.Succeeded)
	}

	entry.Severity = Fine
	if entry.Succeeded == 0 && entry.Failed == 0 {
		entry.Severity = Unimportant
	} else if ratio > 0.5 {
		entry.Severity = HeavilyFlaky
	} else if ratio > 0.2 {
		entry.Severity = MostlyFlaky
	} else if ratio > 0.1 {
		entry.Severity = ModeratelyFlaky
	} else if ratio > 0 {
		entry.Severity = MildlyFlaky
	}
}

// SortTestsByRelevance sorts given tests according to the severity from the test data, where tests with a higher
// severity have a smaller index in the slice than tests with a lower severity.
// The returned slice does not contain
// duplicates, thus if a test has data with several severities the highest one is picked, leading to an earlier
// encounter in the slice.
func SortTestsByRelevance(data map[string]map[string]*Details, tests []string) (testsSortedByRelevance []string) {

	testsToSeveritiesWithNumbers := map[string]map[string]int{}

	foundTests := map[string]struct{}{}

	// Group all tests by severity, ignoring duplicates for the moment, but keeping a record of tests
	// that have been found
	flakinessToTestNames := map[string][]string{}
	for test, jobsToDetails := range data {
		for _, details := range jobsToDetails {
			if _, exists := flakinessToTestNames[details.Severity]; !exists {
				flakinessToTestNames[details.Severity] = []string{}
			}
			flakinessToTestNames[details.Severity] = append(flakinessToTestNames[details.Severity], test)

			// Counting the occurrences a test failed will cause tests with lower number of failures but
			// same number of appearances to appear before tests that have higher number of failures.
			// Also using the ratio of #tests_failed / #tests_succeeded as a factor emphasizes tests
			// with less failures but a high ratio far more than tests with higher number of tests.
			// As we want to emphasize tests that have higher failure numbers over those that have less failures,
			// we tabularize the severity as a factor according to the relation of #tests_failed / #tests_succeeded
			// We then multiply that value by the number of failures.
			var severityRatio float32
			switch {
			case details.Failed > 3*details.Succeeded:
				severityRatio = 1.75
			case details.Failed > details.Succeeded:
				severityRatio = 1.5
			case details.Failed < details.Succeeded:
				severityRatio = 0.75
			default:
				severityRatio = 1
			}
			severityRatio = severityRatio * float32(details.Failed)
			if _, exists := testsToSeveritiesWithNumbers[test]; !exists {
				testsToSeveritiesWithNumbers[test] = map[string]int{details.Severity: int(severityRatio)}
			} else {
				testsToSeveritiesWithNumbers[test][details.Severity] = testsToSeveritiesWithNumbers[test][details.Severity] + int(severityRatio)
			}

			foundTests[test] = struct{}{}
		}
	}

	// Build up the initial sorted result (with duplicates)
	initialTestsSortedByRelevance := BuildUpSortedTestsBySeverity(testsToSeveritiesWithNumbers)

	// Append all tests that have not been found in the data
	for _, test := range tests {
		if _, exists := foundTests[test]; !exists {
			initialTestsSortedByRelevance = append(initialTestsSortedByRelevance, test)
		}
	}

	// Now kill the duplicates by keeping a map of whether the test was encountered before
	encounteredTests := map[string]struct{}{}
	for _, test := range initialTestsSortedByRelevance {
		if _, exists := encounteredTests[test]; !exists {
			encounteredTests[test] = struct{}{}
			testsSortedByRelevance = append(testsSortedByRelevance, test)
		}
	}

	return
}

type TestToSeverityOccurrences struct {
	Name                string
	SeverityOccurrences []int
}

type BySeverity []*TestToSeverityOccurrences

func (t BySeverity) Len() int { return len(t) }
func (t BySeverity) Less(i, j int) bool {
	if len(t[i].SeverityOccurrences) != len(t[j].SeverityOccurrences) {
		return len(t[i].SeverityOccurrences) < len(t[j].SeverityOccurrences)
	}
	for index := 0; index < len(t[i].SeverityOccurrences); index++ {
		if t[i].SeverityOccurrences[index] < t[j].SeverityOccurrences[index] {
			return true
		} else if t[i].SeverityOccurrences[index] > t[j].SeverityOccurrences[index] {
			return false
		}
	}
	return t[i].Name > t[j].Name
}
func (t BySeverity) Swap(i, j int) { t[i], t[j] = t[j], t[i] }

func BuildUpSortedTestsBySeverity(testsToSeveritiesWithOccurrences map[string]map[string]int) []string {

	// Create flat array storing test names to numbers of occurrences for each severity order by priority from left
	// to right
	testsWithAllSeverityOccurrences := make([]*TestToSeverityOccurrences, 0)
	severitiesInOrder := []string{HeavilyFlaky, MostlyFlaky, ModeratelyFlaky, MildlyFlaky, Fine, Unimportant}
	for test, severitiesWithOccurrences := range testsToSeveritiesWithOccurrences {
		severityOccurrences := make([]int, len(severitiesInOrder))
		testWithAllSeverityOccurrences := TestToSeverityOccurrences{Name: test, SeverityOccurrences: severityOccurrences}
		for index, severity := range severitiesInOrder {
			occurrencesOfSeverity := 0
			if existingNumber, exists := severitiesWithOccurrences[severity]; exists {
				occurrencesOfSeverity = existingNumber
			}
			severityOccurrences[index] = occurrencesOfSeverity
		}
		testsWithAllSeverityOccurrences = append(testsWithAllSeverityOccurrences, &testWithAllSeverityOccurrences)
	}

	// now sort the array
	sort.Sort(sort.Reverse(BySeverity(testsWithAllSeverityOccurrences)))

	initialTestsSortedByRelevance := make([]string, len(testsWithAllSeverityOccurrences))
	for index, test := range testsWithAllSeverityOccurrences {
		initialTestsSortedByRelevance[index] = test.Name
	}
	return initialTestsSortedByRelevance
}

const (
	HeavilyFlaky    = "red"
	MostlyFlaky     = "orange"
	ModeratelyFlaky = "yellow"
	MildlyFlaky     = "almostgreen"
	Fine            = "green"
	Unimportant     = "unimportant"
)
