package flakefinder_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/pkg/flakefinder"
)

var _ = Describe("flakefinder.go", func() {

	parseAndFailTime := func(timeAsString string) time.Time {
		time, err := time.Parse(time.RFC3339, timeAsString)
		Expect(err).To(Not(HaveOccurred()))
		return time
	}

	parseAndFailDuration := func(durationAsString string) time.Duration {
		duration, err := time.ParseDuration(durationAsString)
		Expect(err).To(Not(HaveOccurred()))
		return duration
	}

	reportIntervalOptionsWithDurationTodayAndTill := func(durationAsString string, today bool, timeAsString string) flakefinder.ReportIntervalOptions {
		return flakefinder.ReportIntervalOptions{
			Today:  today,
			Merged: parseAndFailDuration(durationAsString),
			Till:   parseAndFailTime(timeAsString),
		}
	}

	reportExecutionTime := "2020-06-30T03:04:05Z"

	previousDayStartFromExecutionTime := "2020-06-29T00:00:00Z"
	previousDayEndFromExecutionTime := "2020-06-29T23:59:59.999Z"

	When("on 24h report GetReportInterval", func() {

		It("has start of previous day as report start time", func() {
			startOfReport, _ := flakefinder.GetReportInterval(reportIntervalOptionsWithDurationTodayAndTill("24h", false, reportExecutionTime))

			Expect(startOfReport).To(BeEquivalentTo(parseAndFailTime(previousDayStartFromExecutionTime)))
		})

		It("has previous day end as report end time", func() {
			_, endOfReport := flakefinder.GetReportInterval(reportIntervalOptionsWithDurationTodayAndTill("24h", false, reportExecutionTime))

			Expect(endOfReport).To(BeEquivalentTo(parseAndFailTime(previousDayEndFromExecutionTime)))
		})

	})

	currentDayStartFromExecutionTime := "2020-06-30T00:00:00Z"

	When("on 24h report GetReportInterval if today is set", func() {

		It("it has start of current day as report start time", func() {
			startOfReport, _ := flakefinder.GetReportInterval(reportIntervalOptionsWithDurationTodayAndTill("24h", true, reportExecutionTime))

			Expect(startOfReport).To(BeEquivalentTo(parseAndFailTime(currentDayStartFromExecutionTime)))
		})

		It("has report execution time as report end time", func() {
			_, endOfReport := flakefinder.GetReportInterval(reportIntervalOptionsWithDurationTodayAndTill("24h", true, reportExecutionTime))

			Expect(endOfReport).To(BeEquivalentTo(parseAndFailTime(reportExecutionTime)))
		})

	})

	previousHourStartFromExecutionTime := "2020-06-30T02:00:00Z"
	previousHourEndFromExecutionTime := "2020-06-30T02:59:59.999Z"

	When("on 1h report GetReportInterval", func() {

		It("has previous hour as report time start", func() {
			startOfReport, _ := flakefinder.GetReportInterval(reportIntervalOptionsWithDurationTodayAndTill("1h", false, reportExecutionTime))

			Expect(startOfReport).To(BeEquivalentTo(parseAndFailTime(previousHourStartFromExecutionTime)))
		})

		It("has current hour as report time end", func() {
			_, endOfReport := flakefinder.GetReportInterval(reportIntervalOptionsWithDurationTodayAndTill("1h", false, reportExecutionTime))

			Expect(endOfReport).To(BeEquivalentTo(parseAndFailTime(previousHourEndFromExecutionTime)))
		})

	})

})
