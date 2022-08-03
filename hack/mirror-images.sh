#!/bin/bash

set -euo pipefail

if ! command -V skopeo; then
    echo "skopeo required!"
    exit 1
fi

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
IMAGE_LIST_FILE="${BASEDIR}/images_to_mirror.csv"
IMAGE_LIST=($(cat "${IMAGE_LIST_FILE}"))

for image in "${IMAGE_LIST[@]}";
do
  source_registry="$(echo "$image" |cut -d',' -f1)"
  image_in_source="$(echo "$image" |cut -d',' -f2)"
  target_registry="$(echo "$image" |cut -d',' -f3)"
  # since we are mirroring all images into a specific organization
  # we cannot use full path of original image which contain user+image name.
  # example: docker.io/user/image:v1 -> quay.io/kubevirtci/user-image:v1
  image_in_target="${image_in_source//\//-}" # replace / with -
  if [[ $image_in_target =~ istio- ]]; then
    # Istio is deployed with Istio operator, which can only be configured to use different registry repository
    # but will always use container image names: pilot, proxyv2, etc.
    # Therefore the images should not have the "istio-" prefix
    # example: istio-pilot:v1 -> pilot:v1
    # this way the istio operator will use quay.io/kubevirtci/pilot:v1 instead of docker.io/istio/pilot:v1
    image_in_target="${image_in_target//istio-/}"
  fi
  echo "Mirroring from $source_registry/$image_in_source to $target_registry/$image_in_target"
  skopeo copy --multi-arch all "docker://$source_registry/$image_in_source" "docker://$target_registry/$image_in_target"
done

echo "DONE!"
