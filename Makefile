deps-update:
	go mod tidy
	go mod vendor
	bazel run //:gazelle

test:
	bazel test //limiter:*

deploy-limiter:
	cd limiter
	gcloud functions deploy limiter --entry-point CutBucketConnections --set-env-vars 'GOOGLE_CLOUD_BUCKETS=builddeps;kubevirt-prow' --trigger-topic billing --runtime go111 --memory=128MB
