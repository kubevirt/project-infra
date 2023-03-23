# test-label-analyzer

This tool has two main use cases
* generate status about what tests are in a certain category
* given certain categories generate a string that can be used directly with [Ginkgo] `--filter` or `--skip` flags

Both use cases support input files that define a set of regular expressions for test names to match a certain category.

## generate status about what tests are in a certain category

Say we want to know about how many tests of a given set (i.e. directory or file set) are in a certain category. We provide a configuration file to define what labels (either inside the test name or as an explicit [Ginkgo label]) match a certain category.

The tool then prints an overview of how many tests are in each category, additionally it prints out a list of all test names including their attributes as where to find each test inside the code base.

## generate a string that can be used directly with [Ginkgo] `--filter` or `--skip` flags

[Ginkgo]: https://onsi.github.io/ginkgo/
[Ginkgo label]: https://onsi.github.io/ginkgo/#spec-labels
