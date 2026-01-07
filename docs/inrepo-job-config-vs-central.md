# Prow - whether to use central or InRepo job configuration

For the KubeVirt GitHub organization we currently are using the central job configuration, job configuration files are
located in folder `github/ci/prow-deploy/files/jobs`, where global configuration files for Prow are symlinked into
`github/ci/prow-deploy/files`, but original files are located in sub-folders below
`github/ci/prow-deploy/kustom/base/configs/current`.

## Central Job Configuration

This is the classic and original method for managing jobs in Prow. All job definitions are stored in a single, dedicated
repository (named test-infra or project-infra in our case), separate from the source code repositories they test.

### How it works

* A central `config.yaml` file in the project-infra repository defines the overall Prow configuration, including which
  plugins are enabled.
* Job configurations (presubmits, postsubmits, periodics) are typically stored in a structured directory within that
  same repository, for example,                    
  jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml.
* To add or modify a CI job for any repository, a developer must create a pull request against this central
  project-infra repository.

### Advantages

* Centralized Control & Security: A dedicated team can review all CI/CD changes, ensuring consistency, security, and
  adherence to best practices across the entire organization.
* Global Visibility: It's easy to get a complete overview of all CI jobs running for the organization by looking
  in one place.
* Enforced Standards: Simplifies the process of enforcing standards, such as using specific base images, resource
  limits, or required command-line arguments for all jobs.
* Bulk Changes: Making a change across many jobs or repositories (e.g., updating a Go version or a critical tool) is
  straightforward because all definitions are in one location.

### Disadvantages

* Review Bottleneck: The team maintaining the central repository can become a bottleneck. Every team needing a
  CI change has to go through them, which can slow down development velocity.
  > [!NOTE] since we are using autoowners to sync OWNERS from source repositories repo users can review their changes
  > themselves
* High Barrier to Entry: Developers must understand the structure and conventions of a separate infrastructure
  repository just to add a simple test job.
* Configuration is Decoupled from Code: A code change that requires a new test job must be coordinated across two
  separate pull requests (one in the application repo, one in the infra repo), which can be cumbersome.
* Scalability Issues: In very large organizations, the central job configuration files can become massive and difficult
  to manage.
  > [!NOTE] since we are using the folder structure approach we don't suffer that much, even for release branches

## InRepoConfig

This is a newer, opt-in feature in Prow that allows repositories to define their own Prow jobs within their own
source tree.

### How it works

* The feature must first be enabled in the central config.yaml for specific organizations or repositories.
* Once enabled, a repository can define its jobs in a file named .prow.yaml (or in a .prow/ directory) in the root
  of that repository.
* Prow's config-bootstrapper component will then dynamically discover and load these jobs.
* To add or modify a CI job, a developer includes the changes to .prow.yaml in the same pull request as their code
  changes.

### Advantages

* Developer Autonomy & Velocity: Teams can manage their own CI configurations without waiting for a central team.
  This significantly speeds up the development cycle.
* Atomic Changes: Code changes and the CI jobs that test them are committed and reviewed in the same pull request.
  If the PR is reverted, the CI changes are reverted with it.
* Improved Scalability: Distributes the ownership and maintenance of CI configuration, preventing the central
  repository from becoming unmanageable.
* Lower Barrier to Entry: Developers can define jobs in the context of the repository they are already working in,
  using a familiar workflow.

### Disadvantages

* Reduced Central Control: It becomes harder to enforce organization-wide standards, as teams are free to define
  their own jobs. This can lead to inconsistencies.
* Configuration Sprawl: Job definitions are scattered across hundreds or thousands of repositories, making it difficult
  to get a global overview or perform bulk
  updates.
* Potential Security Risks: If not properly managed, teams could potentially define jobs with excessive permissions or
  insecure practices. Prow mitigates this by
  allowing centrally-defined JobBase configurations that InRepoConfig jobs must inherit from, but the risk is still
  higher.
* Discovery Complexity: Understanding the full CI picture requires checking many different repositories.

## Similarities of InRepoConfig and project-infra job config

Despite their different approaches, both methods share common ground:

* Core Job Schema: The actual YAML syntax for defining a presubmit, postsubmit, or periodic job is identical in both
  models.
* Underlying Execution: Prow's core components (like Plank, Hook, Sinker) handle the scheduling and execution of jobs in
  the exact same way, regardless of where the
  job was defined.
* Central `config.yaml` is Still Required: InRepoConfig is not fully decentralized. A central `config.yaml` is still the
  source of truth for global settings, plugin
  configurations, Tide (merge automation), and enabling InRepoConfig for specific repos.
* Status Reporting: Both configuration styles report pass/fail statuses back to GitHub pull requests in the same way.

## Differences of InRepoConfig and project-infra job config

| Aspect            | Central Job Configuration                      | InRepoConfig                                                      |
|-------------------|------------------------------------------------|-------------------------------------------------------------------|
| Config Location   | Single, dedicated project-infra repository.    | `.prow.yaml` file or `.prow/` folder inside each code repository. |
| Workflow          | PR to the central repo to change CI jobs.      | CI changes are included in the same PR as code changes.           |
| Control Model     | Centralized, strict control by an infra team.  | Decentralized, delegated control to development teams.            |
| Main Advantage    | Consistency and security.                      | Developer velocity and autonomy.                                  |
| Main Disadvantage | Review bottleneck and slower workflow.         | Configuration sprawl and potential inconsistency.                 |
| Coupling          | CI config is decoupled from the code it tests. | CI config is tightly coupled with the code it tests.              |
