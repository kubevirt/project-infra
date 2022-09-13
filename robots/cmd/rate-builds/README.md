rate-builds
===========

Fetches sigma ratings over number of test failures for a set of builds of a jenkins job.

The [three sigma rule] roughly states that nearly all (99.7%) significant values lie within three standard deviations of the mean. Therefore we can exclude builds with number of test failures beyond three times the mean number of test failures, since they have a low probability of being a normal build considering the number of failures.

How to run
----------

```shell
go run ./robots/cmd/rate-builds/... --help
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
$ go run ./robots/cmd/rate-builds/... --job-name=test-kubevirt-cnv-4.12-compute-ocs
{"level":"info","msg":"Creating client for ...","robot":"badbuilds","time":"2022-09-02T15:05:44+02:00"}
{"job":"test-kubevirt-cnv-4.12-compute-ocs","level":"info","msg":"Fetching completed builds, starting at 57","robot":"badbuilds","time":"2022-09-02T15:05:45+02:00"}
{"build":57,"job":"test-kubevirt-cnv-4.12-compute-ocs","level":"info","msg":"Fetching build no 57","robot":"badbuilds","time":"2022-09-02T15:05:45+02:00"}
...
{"build":33,"job":"test-kubevirt-cnv-4.12-compute-ocs","level":"info","msg":"Build 33 ran at 2022-08-22T00:35:00+02:00","robot":"badbuilds","time":"2022-09-02T15:08:16+02:00"}
{"job":"test-kubevirt-cnv-4.12-compute-ocs","level":"info","msg":"Fetched 20 completed builds","robot":"badbuilds","time":"2022-09-02T15:08:16+02:00"}
{
        "name": "test-kubevirt-cnv-4.12-compute-ocs",
        "source": "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/",
        "startFrom": 1209600000000000,
        "buildNumbers": [
                57,
                55,
                53,
                52,
                50,
                49,
                48,
                47,
                46,
                45,
                44,
                42,
                41,
                40,
                39,
                36,
                35,
                34,
                33,
                32
        ],
        "buildNumbersToData": {
                "32": {
                        "number": 32,
                        "failures": 7,
                        "sigma": 1
                },
                "33": {
                        "number": 33,
                        "failures": 7,
                        "sigma": 1
                },
                "34": {
                        "number": 34,
                        "failures": 7,
                        "sigma": 1
                },
                "35": {
                        "number": 35,
                        "failures": 9,
                        "sigma": 2
                },
                "36": {
                        "number": 36,
                        "failures": 8,
                        "sigma": 1
                },
                "39": {
                        "number": 39,
                        "failures": 6,
                        "sigma": 1
                },
                "40": {
                        "number": 40,
                        "failures": 5,
                        "sigma": 1
                },
                "41": {                                                                                                                                                                                                                                                                        [0/284]
                        "number": 41,
                        "failures": 5,
                        "sigma": 1
                },
                "42": {
                        "number": 42,
                        "failures": 5,
                        "sigma": 1
                },
                "44": {
                        "number": 44,
                        "failures": 6,
                        "sigma": 1
                },
                "45": {
                        "number": 45,
                        "failures": 5,
                        "sigma": 1
                },
                "46": {
                        "number": 46,
                        "failures": 7,
                        "sigma": 1
                },
                "47": {
                        "number": 47,
                        "failures": 10,
                        "sigma": 2
                },
                "48": {
                        "number": 48,
                        "failures": 3,
                        "sigma": 2
                },
                "49": {
                        "number": 49,
                        "failures": 3,
                        "sigma": 2
                },
                "50": {
                        "number": 50,
                        "failures": 3,
                        "sigma": 2
                },
                "52": {
                        "number": 52,
                        "failures": 8,
                        "sigma": 1
                },
                "53": {
                        "number": 53,
                        "failures": 9,
                        "sigma": 2
                },
                "55": {
                        "number": 55,
                        "failures": 7,
                        "sigma": 1
                },
                "57": {
                        "number": 57,
                        "failures": 5,
                        "sigma": 1
                }
        },
        "totalCompletedBuilds": 20,
        "totalFailures": 125,
        "mean": 6.25,
        "variance": 3.8875,
        "standardDeviation": 1.9716744153130354
}
```

[three sigma rule]: https://en.wikipedia.org/wiki/68%E2%80%9395%E2%80%9399.7_rule
