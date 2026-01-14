# retests-to-merge

This tool fetches the most recent set of pull requests from kubevirt/kubevirt that have `lgtm` and `approved` labels.

For each of these it logs to stdout the pull request url, the number of retests since last commit and the set of labels present.
 Output is sorted from high to low.

## how to run

```bash
go run ./robots/retests-to-merge/main.go \
    --github-token-path=/path/to/your/token
```
