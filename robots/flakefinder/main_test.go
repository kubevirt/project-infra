package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("main.go", func() {

	When("Setting up output path", func() {

		BeforeEach(func() {
			ReportOutputPath = ReportsPath
		})

		It("has default path", func() {
			options := options{}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder"))
		})

		It("has preview if option enabled", func() {
			options := options{isPreview: true}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/preview"))
		})

		It("has child branch", func() {
			options := options{reportOutputChildPath: "master"}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/master"))
		})

		It("has preview and child branch", func() {
			options := options{isPreview: true, reportOutputChildPath: "master"}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/preview/master"))
		})

	})

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

	optionsWithDurationAndToday := func(durationAsString string, today bool) options {
		return options{
			merged: parseAndFailDuration(durationAsString),
			today:  today,
		}
	}

	reportExecutionTime := "2020-06-30T03:04:05Z"

	previousDayStartFromExecutionTime := "2020-06-29T00:00:00Z"
	previousDayEndFromExecutionTime := "2020-06-29T23:59:59.999Z"

	When("on 24h report GetReportInterval", func() {

		It("has start of previous day as report start time", func() {
			startOfReport, _ := GetReportInterval(optionsWithDurationAndToday("24h", false), parseAndFailTime(reportExecutionTime))

			Expect(startOfReport).To(BeEquivalentTo(parseAndFailTime(previousDayStartFromExecutionTime)))
		})

		It("has previous day end as report end time", func() {
			_, endOfReport := GetReportInterval(optionsWithDurationAndToday("24h", false), parseAndFailTime(reportExecutionTime))

			Expect(endOfReport).To(BeEquivalentTo(parseAndFailTime(previousDayEndFromExecutionTime)))
		})

	})

	currentDayStartFromExecutionTime := "2020-06-30T00:00:00Z"

	When("on 24h report GetReportInterval if today is set", func() {

		It("it has start of current day as report start time", func() {
			startOfReport, _ := GetReportInterval(optionsWithDurationAndToday("24h", true), parseAndFailTime(reportExecutionTime))

			Expect(startOfReport).To(BeEquivalentTo(parseAndFailTime(currentDayStartFromExecutionTime)))
		})

		It("has report execution time as report end time", func() {
			_, endOfReport := GetReportInterval(optionsWithDurationAndToday("24h", true), parseAndFailTime(reportExecutionTime))

			Expect(endOfReport).To(BeEquivalentTo(parseAndFailTime(reportExecutionTime)))
		})

	})

	previousHourStartFromExecutionTime := "2020-06-30T02:00:00Z"
	previousHourEndFromExecutionTime := "2020-06-30T02:59:59.999Z"

	When("on 1h report GetReportInterval", func() {

		It("has previous hour as report time start", func() {
			startOfReport, _ := GetReportInterval(optionsWithDurationAndToday("1h", false), parseAndFailTime(reportExecutionTime))

			Expect(startOfReport).To(BeEquivalentTo(parseAndFailTime(previousHourStartFromExecutionTime)))
		})

		It("has current hour as report time end", func() {
			_, endOfReport := GetReportInterval(optionsWithDurationAndToday("1h", false), parseAndFailTime(reportExecutionTime))

			Expect(endOfReport).To(BeEquivalentTo(parseAndFailTime(previousHourEndFromExecutionTime)))
		})

	})

})
