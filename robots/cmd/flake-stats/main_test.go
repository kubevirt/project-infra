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

	DescribeTable("TopXTests less comparison",
		func(t TopXTests, expected bool) {
			Expect(t.Less(0, 1)).To(BeEquivalentTo(expected))
		},
		Entry("Sum i > j => i Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresSum(42)),
				NewTopXTestWithOptions("j", WithAllFailuresSum(17)),
			},
			true,
		),
		Entry("Sum i < j =>  i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresSum(17)),
				NewTopXTestWithOptions("j", WithAllFailuresSum(42)),
			},
			false,
		),
		Entry("Sum i = j => i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresSum(17)),
				NewTopXTestWithOptions("j", WithAllFailuresSum(17)),
			},
			false,
		),
		Entry("Max i > j => i Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresMax(42)),
				NewTopXTestWithOptions("j", WithAllFailuresMax(17)),
			},
			true,
		),
		Entry("Max i < j =>  i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresMax(17)),
				NewTopXTestWithOptions("j", WithAllFailuresMax(42)),
			},
			false,
		),
		Entry("Max i = j => i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresMax(17)),
				NewTopXTestWithOptions("j", WithAllFailuresMax(17)),
			},
			false,
		),
		Entry("Avg i > j => i Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresAvg(42)),
				NewTopXTestWithOptions("j", WithAllFailuresAvg(17)),
			},
			true,
		),
		Entry("Avg i < j =>  i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresAvg(17)),
				NewTopXTestWithOptions("j", WithAllFailuresAvg(42)),
			},
			false,
		),
		Entry("Avg i = j => i not Less j",
			TopXTests{
				NewTopXTestWithOptions("i", WithAllFailuresAvg(17)),
				NewTopXTestWithOptions("j", WithAllFailuresAvg(17)),
			},
			false,
		),
	)
})

type TopXTestOption func(*TopXTest)

func WithAllFailuresSum(sum int) TopXTestOption {
	return func(t *TopXTest) {
		t.AllFailures.Sum = sum
	}
}

func WithAllFailuresMax(max int) TopXTestOption {
	return func(t *TopXTest) {
		t.AllFailures.Max = max
	}
}

func WithAllFailuresAvg(avg float64) TopXTestOption {
	return func(t *TopXTest) {
		t.AllFailures.Avg = avg
	}
}

func NewTopXTestWithOptions(name string, options ...TopXTestOption) *TopXTest {
	test := NewTopXTest(name)
	for _, o := range options {
		o(test)
	}
	return test
}
