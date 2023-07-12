package cmd

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	test_label_analyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"regexp"
)

var _ = Describe("root tests", func() {
	Context("getConfig", func() {
		DescribeTable("returns a config",
			func(options *configOptions, expectedConfig *test_label_analyzer.Config, expectedErr error) {
				config, err := options.getConfig()
				if err != nil {
					Expect(err).To(BeEquivalentTo(expectedErr))
				} else {
					Expect(config).To(BeEquivalentTo(expectedConfig))
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("returns err if no config selected",
				&configOptions{
					configFile:         "",
					configName:         "",
					ginkgoOutlinePaths: nil,
					testFilePath:       "",
					remoteURL:          "",
					testNameLabelRE:    "",
					outputHTML:         false,
				},
				nil,
				fmt.Errorf("no configuration found!"),
			),
			Entry("for simple RE",
				&configOptions{
					configFile:         "",
					configName:         "",
					ginkgoOutlinePaths: nil,
					testFilePath:       "",
					remoteURL:          "",
					testNameLabelRE:    "test regex",
					outputHTML:         false,
				},
				test_label_analyzer.NewTestNameDefaultConfig("test regex"),
				nil,
			),
			Entry("for quarantine config",
				&configOptions{
					configFile:         "",
					configName:         "quarantine",
					ginkgoOutlinePaths: nil,
					testFilePath:       "",
					remoteURL:          "",
					testNameLabelRE:    "",
					outputHTML:         false,
				},
				test_label_analyzer.NewQuarantineDefaultConfig(),
				nil,
			),
			Entry("for file with test names",
				&configOptions{
					configFile:          "",
					filterTestNamesFile: "testdata/filter-test-names.json",
					configName:          "",
					ginkgoOutlinePaths:  nil,
					testFilePath:        "",
					remoteURL:           "",
					testNameLabelRE:     "",
					outputHTML:          false,
				},
				&test_label_analyzer.Config{
					Categories: []test_label_analyzer.LabelCategory{
						{
							Name:            "flaky",
							TestNameLabelRE: test_label_analyzer.NewRegexp("test name 1"),
							GinkgoLabelRE:   nil,
						},
						{
							Name:            "also flaky",
							TestNameLabelRE: test_label_analyzer.NewRegexp("test name 2"),
							GinkgoLabelRE:   nil,
						},
						{
							Name:            "also flaky",
							TestNameLabelRE: test_label_analyzer.NewRegexp(regexp.QuoteMeta("[sig-compute]test name 3")),
							GinkgoLabelRE:   nil,
						},
					},
				},
				nil,
			),
		)
	})
})
