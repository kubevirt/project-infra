# Uploader

The binary checks the specified `WORKSPACE` file (or Bzlmod `MODULE.bazel` file
and any referenced `bazeldnf` JSON lockfiles such as `rpms.json`) for
unmirrored dependencies, uploads them to the specified GCS bucket, prunes stale
`404` upstream URLs, and adds the GCS object URL as a fallback download
location.

If `-workspace` points to a missing `WORKSPACE` file, `uploader` automatically
falls back to `MODULE.bazel` in the same directory so repositories and branches
can transition from `WORKSPACE` to Bzlmod without breaking periodic jobs.

```bash
export GOOGLE_APPLICATION_CREDENTIALS=credential-path-to.json

# Legacy WORKSPACE (falls back to MODULE.bazel + rpms.json if WORKSPACE is absent)
go run ./robots/uploader -workspace <kubevirt-source-path>/WORKSPACE # dry-run
go run ./robots/uploader -workspace <kubevirt-source-path>/WORKSPACE -dry-run=false

# Explicit Bzlmod MODULE.bazel mode (auto-discovers bazeldnf lock_file attributes)
go run ./robots/uploader -module <kubevirt-source-path>/MODULE.bazel # dry-run
go run ./robots/uploader -module <kubevirt-source-path>/MODULE.bazel -dry-run=false
```
