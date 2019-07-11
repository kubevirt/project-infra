package main

import (
	"fmt"
	"html/template"
	"os"

	"github.com/joshdk/go-junit"
)

const tpl = `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<style>
table, th, td {
  border: 1px solid black;
}
.yellow {
  background-color: #ffff80;
}
.almostgreen {
  background-color: #dfff80;
}
.green {
  background-color: #9fff80;
}
.red {
  background-color: #ff8080;
}
.orange {
  background-color: #ffbf80;
}
.unimportant {
}
.center {
  text-align:center
}
</style>
</head>
<body>

<table>
<tr>
<td></td>
{{ range $header := $.Headers }}
  <td>{{ $header }}</td>
{{ end }}
</tr>
{{ range $test := $.Tests }}
<tr>
  <td>{{ $test }}</td>
{{ range $header := $.Headers }}
    {{if not (index $.Data $test $header) }}
  <td class="center">
      N/A
  </td>
    {{else}}
  <td class="{{ (index $.Data $test $header).Severity }} center">
      {{ (index $.Data $test $header).Failed }}/{{ (index $.Data $test $header).Succeeded }}/{{ (index $.Data $test $header).Skipped }}
    {{end}}
  </td>
{{ end }}
</tr>
{{ end }}

</table>

</body>
</html>
`

type params struct {
	Data    map[string]map[string]*details
	Headers []string
	Tests   []string
}

type details struct {
	Succeeded int
	Skipped   int
	Failed    int
	Severity  string
}

func Report(results []*Result) error {
	t, err := template.New("report").Parse(tpl)
	if err != nil {
		return fmt.Errorf("failed to load report template: %v", err)
	}

	data := map[string]map[string]*details{}
	headers := []string{}
	tests := []string{}
	buildNumberMap := map[int]struct{}{}
	headerMap := map[string]struct{}{}

	for _, result := range results {
		if _, exists := buildNumberMap[result.BuildNumber]; exists {
			// merge pool > 1
			continue
		}
		buildNumberMap[result.BuildNumber] = struct{}{}

		// first find failing tests to isolate tests which interest us
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if test.Status == junit.StatusFailed || test.Status == junit.StatusError {
					testEntry := data[test.Name]
					if testEntry == nil {
						tests = append(tests, test.Name)
						testEntry = map[string]*details{}
						data[test.Name] = testEntry
					}

					if _, exists := testEntry[result.Job]; !exists {
						testEntry[result.Job] = &details{}
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
		if _, exists := buildNumberMap[result.BuildNumber]; !exists {
			// if not in the map now, then skip it
			continue
		}
		for _, suite := range result.JUnit {
			for _, test := range suite.Tests {
				if _, exists := data[test.Name]; !exists {
					continue
				}
				if _, exists := data[test.Name][result.Job]; !exists {
					data[test.Name][result.Job] = &details{}
				}
				if test.Status == junit.StatusSkipped {
					data[test.Name][result.Job].Skipped = data[test.Name][result.Job].Skipped + 1
				} else if test.Status == junit.StatusPassed {
					data[test.Name][result.Job].Succeeded = data[test.Name][result.Job].Succeeded + 1
				}
			}
		}
	}

	// third, calculate the severity
	// second enrich failed tests with additional information
	for _, result := range results {
		if _, exists := buildNumberMap[result.BuildNumber]; !exists {
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

				var ratio float32 = 1.0
				if entry.Succeeded > 0 {
					ratio = float32(entry.Failed) / float32(entry.Succeeded)
				}

				entry.Severity = "green"
				if entry.Succeeded == 0 && entry.Failed == 0 {
					entry.Severity = "unimportant"
				} else if ratio > 0.5 {
					entry.Severity = "red"
				} else if ratio > 0.2 {
					entry.Severity = "orange"
				} else if ratio > 0.1 {
					entry.Severity = "yellow"
				} else if ratio > 0 {
					entry.Severity = "almostgreen"
				}
			}
		}
	}

	err = t.Execute(os.Stdout, params{Data: data, Headers: headers, Tests: tests})
	if err != nil {
		return fmt.Errorf("failed to render report template: %v", err)
	}
	return nil
}
