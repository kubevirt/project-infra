badbuilds
=========

Tool to fetch three sigma ratings over a set of builds for a jenkins job.

How to run
----------

```shell
go run ./robots/cmd/badbuilds/... --help
Usage of /tmp/go-build1320403461/b001/exe/badbuilds:
  -endpoint string
        jenkins base url (default "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/")
  -job-name string
        jenkins job name
  -start-from duration
        jenkins job name (default 336h0m0s)
```

Simple example:

```shell
go run ./robots/cmd/badbuilds/... --job-name=test-kubevirt-cnv-4.11-storage-ocs
```
