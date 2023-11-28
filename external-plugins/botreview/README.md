botreview plugin
================

Automates "simple reviews", meaning reviews that have been created by automation and where a review process can be easily put into code.

Motivation
----------
Most of the time `ci-maintainers` are looking at PRs that have been created by some kind of automation, i.e. the prow update mechanism, the prowjob image update, the kubevirtci bump, etc.
Updates in these PRs are mostly tedious to review for a human, since they contain lengthy repeated updates to some URLs or some image reference. A human could only look at these changes and try to manually spot errors in the references, which first of all is hard and second is already covered by the prow-deploy-presubmit.

What `botreview` can at least do is automate what a human would do anyway, like applying an expected change pattern to the changes. And this is what botreview does.

`botreview` has of course room for improval, i.e. it might generate a list of the images and check whether these are pull-able, or even perform further checks on the images. **Note: the latter is not implemented (yet)**
