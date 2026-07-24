# The directory containing this Makefile
PLUGINS_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
PROJECT_INFRA_ROOT := $(abspath $(dir $(PLUGINS_ROOT)))

# Try to determine the plugin to build from the current directory
ifneq ($(CURDIR),$(PLUGINS_ROOT))
PROW_PLUGIN := $(shell \
    path="$(CURDIR)"; tail=$${path#$(PLUGINS_ROOT)/}; \
    [ "$${path}" != "$${tail}" ] && echo "$${tail%%/*}" \
)
endif

ifeq ($(PROW_PLUGIN),)
define err_no_plugin
[ERROR] Could not determine which plugin to process.
It can be specified from the command line `make PROW_PLUGIN=<plugin>`
endef
$(error $(err_no_plugin))
endif

CRI_BIN         := podman
IMAGE_REGISTRY  := quay.io
IMAGE_NAMESPACE := kubevirtci
IMAGE_NAME      := $(IMAGE_REGISTRY)/$(IMAGE_NAMESPACE)/$(PROW_PLUGIN)
IMAGE_TAG       := $(shell date +%Y%m%d%H%M)-$(shell git rev-parse --short=7 HEAD)
PLUGIN_DIR      := $(PLUGINS_ROOT)/$(PROW_PLUGIN)

.PHONY: Makefile build format image test push
.SUFFIXES:

build:
	@cd $(PLUGIN_DIR) && set -eux; go build ./...

format:
	@cd $(PLUGIN_DIR) && set -eux; gofmt -w ./

test:
	@cd $(PLUGIN_DIR) && set -eux; go test ./...

image:
	@cd $(PROJECT_INFRA_ROOT) && set -eux; \
	$(CRI_BIN) image build . \
	    --build-arg=PROW_PLUGIN=$(PROW_PLUGIN) \
	    --file=external-plugins/Containerfile \
	    --volume=$(PROJECT_INFRA_ROOT):/workspace \
	    --security-opt=label=disable \
	    --squash \
	    --tag=$(IMAGE_NAME):$(IMAGE_TAG)

push:: image
	@cd $(PROJECT_INFRA_ROOT) && set -eux; \
	$(CRI_BIN) image push $(IMAGE_NAME):$(IMAGE_TAG); \
	./hack/update-deployments-with-latest-image.sh $(IMAGE_NAME) $(IMAGE_TAG)

ifeq ($(PROW_PLUGIN),coverage)
push:: COVERAGE_JOB_IMAGE = covreport
push::
	@cd $(PROJECT_INFRA_ROOT)/images && set -eux; \
	./publish_image.sh $(COVERAGE_JOB_IMAGE) $(IMAGE_REGISTRY) $(IMAGE_NAMESPACE); \
	./update-latest-image-tag.sh $(COVERAGE_JOB_IMAGE) $(IMAGE_REGISTRY) $(IMAGE_NAMESPACE)
endif
