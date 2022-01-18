project-infra images
=================

Contains configurations for container images that are used by KubeVirt CI Prow jobs. Most of the images have corresponding build jobs that are executed after the configuration of an image in one of the subdirectories is changed. The job then will build and push the image to `quay.io/kubevirtci` image registry.

`publish_image.sh`
------------------

You can use the above script to build and publish images to `quay.io` (provided you have the quay credentials)

Example:

```bash
./publish_image.sh -b golang quay.io kubevirtci
```

builds the image in folder `images/golang` and tags it with `quay.io/kubevirtci/golang:v20211232-abcdefgh` where the tag is calculated from date and current git commit id when building 


Updating published images
-------------------------

See
* [./hack/update-jobs-with-latest-image.sh](../hack/update-jobs-with-latest-image.sh) for doing a manual update for a selected directory containing job definitions.
* [./hack/bump-prow-job-images.sh](../hack/bump-prow-job-images.sh) for updating all references to images hosted here in all the [jobs configured to run with prow](../github/ci/prow-deploy/files/jobs/)

`update-jobs-with-latest-image.sh`
----------------------------------

if you want to update an image to the latest version, you can use the script `update-jobs-with-latest-image.sh like this:

```bash
./hack/update-jobs-with-latest-image.sh gcr.io/k8s-testimages/bootstrap ./github/ci/prow-deploy/files/jobs/kubevirt/cluster-network-addons-operator
```

which then would produce this

```
diff --git a/github/ci/prow-deploy/files/jobs/kubevirt/cluster-network-addons-operator/cluster-network-addons-operator-presubmits.yaml b/github/ci/prow-deploy/files/jobs/kubevirt/cluster-network-addons-operator/cluster-network-addons-operator-presubmits.yaml
index 89e2b8b2..43a5b7f9 100644
--- a/github/ci/prow-deploy/files/jobs/kubevirt/cluster-network-addons-operator/cluster-network-addons-operator-presubmits.yaml
+++ b/github/ci/prow-deploy/files/jobs/kubevirt/cluster-network-addons-operator/cluster-network-addons-operator-presubmits.yaml
@@ -54,7 +54,7 @@ presubmits:
nodeSelector:
type: bare-metal-external
containers:
-          - image: gcr.io/k8s-testimages/bootstrap:v20190516-c6832d9
+          - image: gcr.io/k8s-testimages/bootstrap:v20210913-fc7c4e8
             securityContext:
               privileged: true
             resources:
@@ -86,7 +86,7 @@ presubmits:
nodeSelector:
type: bare-metal-external
containers:
-          - image: gcr.io/k8s-testimages/bootstrap:v20190516-c6832d9
+          - image: gcr.io/k8s-testimages/bootstrap:v20210913-fc7c4e8
             securityContext:
               privileged: true
             resources:
@@ -118,7 +118,7 @@ presubmits:
nodeSelector:
type: bare-metal-external
containers:
-          - image: gcr.io/k8s-testimages/bootstrap:v20190516-c6832d9
+          - image: gcr.io/k8s-testimages/bootstrap:v20210913-fc7c4e8
             securityContext:
               privileged: true
             resources:
```

`bump-prow-job-images.sh`
-------------------------

Used by the prow job [`periodic-project-infra-prow-job-image-bump`](../github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-periodics.yaml) to regularly update the image references in the prow job configurations.
