release-querier
===============

`release-querier` is a tool to find latest releases on github projects which use semver.

`release-querier` does the following:

1. Finding the latest release of a github project
2. Finding the latest patch release of a minor release
3. Finding the last three minor releases of a major release (what k8s supports)
4. Format the output via a configurable template for better script integration

Example
=======

```
$ # Latest k8s release
$ release-querier --github-token-path="" -org=kubernetes -repo=kubernetes -latest
v1.17.3
$
$ # Current latest supported k8s versions
$ release-querier --github-token-path="" -org=kubernetes -repo=kubernetes -last-three-minor-of v1
v1.17.3
v1.16.7
v1.15.9
$
$ # Latest 1.17 patch release
$ release-querier --github-token-path="" -org=kubernetes -repo=kubernetes -last-patch-of v1.17
v1.17.3
$ # Change the output format
$ release-querier --github-token-path="" -org=kubernetes -repo=kubernetes -last-three-minor-of v1 --template="{{.Major}}.{{.Minor}}"
1.17
1.16
1.15
```
