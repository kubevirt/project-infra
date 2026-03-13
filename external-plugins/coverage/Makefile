CONTAINER_IMAGE := coverage
CONTAINER_REPO  := quay.io/kubevirtci/$(CONTAINER_IMAGE)
JOB_IMAGE       := covreport

.PHONY: all clean format test push push-plugin push-job
all: format test push

build:
	go build ./...

format:
	gofmt -w .

test:
	go test ./...

push: push-plugin push-job

push-plugin:
	cd ../../images && ./publish_image.sh $(CONTAINER_IMAGE) quay.io kubevirtci
	bash -x ../../hack/update-deployments-with-latest-image.sh $(CONTAINER_REPO)

push-job:
	cd ../../images && ./publish_image.sh $(JOB_IMAGE) quay.io kubevirtci
	cd ../../images && ./update-latest-image-tag.sh $(JOB_IMAGE) quay.io kubevirtci


