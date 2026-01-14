# test-label-analyzer

This tool has two main use cases
* generate stats about what tests are in a certain category
* given certain categories generate a string that can be used directly with [Ginkgo] `--filter` or `--skip` flags

Both use cases support input files that define a set of regular expressions for test names to match a certain category.

## generate stats about what tests are in a certain category

Say we want to know about how many tests of a given set (i.e. directory or file set) are in a certain category. We provide a configuration file to define what labels (either inside the test name or as an explicit [Ginkgo label]) match a certain category.

The tool then prints an overview of how many tests are in each category, additionally it prints out a list of all test names including their attributes as where to find each test inside the code base.

### Examples

#### Using outline files

```sh
$ # create an output directory
$ mkdir -p /tmp/ginkgo-outlines
$ # generate outline data files from the ginkgo test files (those that contain an import from ginkgo)
$ for test_file in $(cd $ginkgo_test_dir && grep -l 'github.com/onsi/ginkgo/v2' ./*.go); do; \
    ginkgo outline --format json $test_file \
        > /tmp/ginkgo-outlines/${test_file//[\/\.]/_}.ginkgooutline.json ; \
  done
$ # feed input files to test-label-analyzer to generate stats
$ test-label-analyzer stats --config-name quarantine \
    $(for outline_file in $(ls /tmp/ginkgo-outlines/); do; \
        echo " --test-outline-filepath /tmp/ginkgo-outlines/$outline_file" | \
        tr -d '\n'; done; echo "") \
        > /tmp/test-label-analyzer-output.json
$ # print the output
$ cat /tmp/test-label-analyzer-output.json
{"SpecsTotal":1483,"SpecsMatching":9,"MatchingSpecPaths":[[{"name":"Describe","text":...
$ # from the output we can generate the concatenated test names
$ jq '.MatchingSpecPaths[] | [ .[].text ] | join(" ")' /tmp/test-label-analyzer-output.json
```

#### Letting `test-labels-analyzer` call ginkgo to retrieve the outline data

```sh
# point test-label-analyzer to the directory containing the test source files
$ test-label-analyzer stats --config-name quarantine --test-file-path /path/to/tests \
        > /tmp/test-label-analyzer-output.json
$ # print the output
$ cat /tmp/test-label-analyzer-output.json
{"SpecsTotal":1483,"SpecsMatching":9,"MatchingSpecPaths":[[{"name":"Describe","text":...
$ # from the output we can generate the concatenated test names
$ jq '.MatchingSpecPaths[] | [ .[].text ] | join(" ")' /tmp/test-label-analyzer-output.json
```

#### Directly letting `test-labels-analyzer` filter tests with regular expression

```sh
# point test-label-analyzer to the directory containing the test source files
$ test-label-analyzer stats --test-name-label-re '.*Console Proxy Operand Resource.*' --test-file-path /home/dhiller/Projects/github.com/kubevirt.io/ssp-operator/tests \
        > /tmp/test-label-analyzer-output.json
$ # print the output
$ cat /tmp/test-label-analyzer-output.json
{"SpecsTotal":278,"SpecsMatching":40,"MatchingSpecPaths":[[{"name":"Describe", ...
$ # from the output we can generate the concatenated test names
$ jq '.MatchingSpecPaths[] | [ .[].text ] | join(" ")' /tmp/test-label-analyzer-output.json
"VM Console Proxy Operand Resource creation created cluster resource [test_id:TODO] cluster role"
"VM Console Proxy Operand Resource creation created cluster resource [test_id:TODO] cluster role binding"
...
```

#### Generating an html page from an existing profile

```shell
$ test-label-analyzer stats --output-html=true \
  --config-name quarantine \
  --test-file-path $(cd ../kubevirt && pwd)/tests \
  --remote-url 'https://github.com/kubevirt/kubevirt/tree/main/tests' > /tmp/test-output.html
```

#### Generating a json file with a rules input file

```shell
$ test-label-analyzer stats \
    --filter-test-names-file ./quarantined_tests.json \
    --test-file-path ../../kubevirt.io/kubevirt/tests/ \
    --remote-url 'https://github.com/kubevirt/kubevirt/tree/main/tests/' \
    > /tmp/ds-quarantined-tests.json
$ # we are going to select the files that had matching tests here
$ jq '.files_stats[] | select( .test_stats.matching_spec_paths != null ) | .path' /tmp/ds-quarantined-tests.json "https://github.com/kubevirt/kubevirt/tree/main/tests/vm_test.go"
"https://github.com/kubevirt/kubevirt/tree/main/tests/migration_test.go"
"https://github.com/kubevirt/kubevirt/tree/main/tests/virtctl/ssh.go"
"https://github.com/kubevirt/kubevirt/tree/main/tests/network/port_forward.go"
"https://github.com/kubevirt/kubevirt/tree/main/tests/network/vmi_multus.go"
"https://github.com/kubevirt/kubevirt/tree/main/tests/operator/operator.go"
"https://github.com/kubevirt/kubevirt/tree/main/tests/virtctl/scp.go"

```

## generate a string that can be used directly with [Ginkgo] `--filter` or `--skip` flags

_**NOT YET IMPLEMENTED**_

[Ginkgo]: https://onsi.github.io/ginkgo/
[Ginkgo label]: https://onsi.github.io/ginkgo/#spec-labels
