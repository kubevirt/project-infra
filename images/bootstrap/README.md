Bootstrap image
===============

KubeVirt CI general purpose image for prow jobs.

**Note: this image doesn't have `golang` builtin, use the golang image in this case.**

Docker debug logs
-----------------

To enable docker debug logs, add this to the prowjob container configuration

```yaml
env:
...
- name: DOCKER_DEBUG
  value: "true"
```
