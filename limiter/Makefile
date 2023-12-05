test:
	bazel test //limiter:*

deploy:
	gcloud functions deploy limiter --entry-point CutBucketConnections --set-env-vars 'GOOGLE_CLOUD_BUCKETS=builddeps;kubevirt-prow' --trigger-topic billing --runtime go121 --memory=128MB
