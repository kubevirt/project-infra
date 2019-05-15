deps-update:
	bazel run //:gazelle -- update-repos -from_file=limiter/go.mod
	bazel run //:gazelle

test:
	bazel test //limiter:*

deploy:
	gcloud functions deploy limiter --entry-point CutBucketConnections --set-env-vars 'GOOGLE_CLOUD_BUCKETS=builddeps;kubevirt-prow' --trigger-topic billing --runtime go111 --memory=128MB
