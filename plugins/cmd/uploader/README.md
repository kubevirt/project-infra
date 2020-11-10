# Uploader

The binary checks the specified WORKSPACE file for unmirrored dependencies,
uploads them to the specified GCS bucket and finally adds the upload location as
a fallback download location to the WORKSPACE file.

```
export GOOGLE_APPLICATION_CREDENTIALS=credential-path-to.json
bazelisk run //plugins/cmd/uploader:uploader -- -workspace <kubevirt-source-path>/WORKSPACE # you will see a dry-run
bazelisk run //plugins/cmd/uploader:uploader -- -workspace <kubevirt-source-path>/WORKSPACE -dry-run=false # will do the upload and modify WORKSPACE
```
