# Adding or moving a repository to the KubeVirt organization

In certain occasions it makes sense to create or move a repository to the KubeVirt GitHub organization. 

However, since we don't want this to go unnoticed for several reasons, i.e. the possible advertisement that we have a new and helpful project for the KubeVirt community, we are advocating a process that every repository needs to go through before it is added to the KubeVirt GitHub org.

## Questions to ask

* why is the repo a good fit for the org?
* does the licensing fit the overall licensing in KubeVirt?
* are all maintainers ok with transitioning?
* is CI support required and if so do we have the CI capacity left? Ask the [CI maintainers](../OWNERS_ALIASES) to make sure of this
* are minimal parameters for repo and maintainer team defined (description, maintainers)?

## Repository and team configuration

Basically you need to create a PR against `orgs.yaml`, where you add a new entry to the `repos` and another to the `teams` sections. These then define the repository properties and the team that maintains and has access to the repository.

Here's the PR adding the `monitoring` repository as an example: https://github.com/kubevirt/project-infra/pull/990/files

Consider looking into enabling [merge automation](https://github.com/kubevirt/community/blob/main/docs/add-merge-automation-to-your-repository.md) for the new repository.

## Pull request template

Before you create the pull request, you should be notified with a pull request creation url similar to this one:

```
https://github.com/{your-github-handle}/project-infra/pull/new/{your-github-branch}
```

Use the template [here](../.github/PULL_REQUEST_TEMPLATE/add_repo.md) by adding the template parameter to your pull request creation url:

```
https://github.com/{your-github-handle}/project-infra/pull/new/{your-github-branch}?template=add_repo.md
```

After that PR has been merged, the automation will kick in within around two hours.

## Automation

Regarding how the automation works, you can find further details in [Automating GitHub org management].

[Automating GitHub org management]: https://github.com/kubevirt/community/blob/main/docs/automating-github-org-management.md 
