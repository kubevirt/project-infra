# shellcheck shell=bash
# Sourced by runner.sh as a setup mixin. A Go version not cached in the image
# is downloaded here, and gimme discards curl errors and does not retry, so a
# single network failure would otherwise leave the job running without go.
for gimme_attempt in 1 2 3; do
    if gimme_env="$(
        if [[ ${gimme_attempt} -gt 1 ]]; then
            sleep $((gimme_attempt * 5))
            # tracing only on retries: it is the one place the failed URL shows up
            export GIMME_DEBUG=true
        fi
        gimme "${GIMME_GO_VERSION}"
    )"; then
        eval "${gimme_env}"
        unset gimme_attempt gimme_env
        return 0
    fi
    echo "gimme could not install go ${GIMME_GO_VERSION} (attempt ${gimme_attempt} of 3)" >&2
done
echo "giving up: go ${GIMME_GO_VERSION} is not installed, aborting the job before it runs" >&2
# exit, not return: runner.sh sources this with errexit off, so a returned
# failure would let the job run on without go. That skips its podman cleanup,
# which the pod teardown makes moot.
exit 1
