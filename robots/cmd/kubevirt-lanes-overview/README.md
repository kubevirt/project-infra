# KubeVirt e2e test lanes overview

Creates a [CSV] file that contains all e2e test lanes and the env settings per job, so that we can see what settings the lanes actually have. 

## How to run
```shell
go run ./robots/cmd/kubevirt-lanes-overview/...
...
2022/09/19 15:45:53 skipping presubmit "pull-kubevirt-code-lint"
2022/09/19 15:45:53 Output written to "/tmp/kubevirt-lanes-overview-1190285516.csv"
```


[CSV]: https://www.rfc-editor.org/rfc/rfc4180.html
