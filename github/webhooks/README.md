KubeVirt GitHub Webhooks
========================

We need two secret tokens, one provides the secret that is configured when
creating the webhook for a repository, while the other provides a personal
auth token used to submit results back to github. Set these tokens as local
env variables

  export SIG_TOKEN=...long random string...
  export AUTH_TOKEN=...some hash...

To create the project in OpenShift Online v3 then do

 # oc create -f openshift-template.json

 # oc process -p GITHUB_SIG_TOKEN_SECRET=$SIG_TOKEN -p GITHUB_AUTH_TOKEN_SECRET=$AUTH_TOKEN kubevirt-github-webhooks | oc create -f -


To later cleanup do

 # oc process kubevirt-github-webhooks | oc delete -f -

 # oc delete -f openshift-template.json
