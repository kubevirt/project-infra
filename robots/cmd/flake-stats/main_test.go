package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("main", func() {
	DescribeTable("normalizeTestName",
		func(input, expected string) {
			Expect(normalizeTestName(input)).To(BeEquivalentTo(expected))
		},
		Entry("basic", "test[QUARANTINE]test", "testtest"),
		Entry("one space", "test [QUARANTINE]test", "test test"),
		Entry("two spaces", "test [QUARANTINE] test", "test test"),
	)
})
