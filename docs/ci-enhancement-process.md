# KubeVirt CI Enhancement Process

This repository is used to manage KubeVirt CI Enhancement Proposals (CEPs), emphasizing centralized prioritization and enhanced SIG involvement and collaboration. This process is based on the [KubeVirt Enhancement Proposal Process](https://github.com/kubevirt/enhancements/blob/main/README.md).

## WHY

The process aims to focus the community's efforts on prioritized pull requests that enhance the KubeVirt CI system, increase the review bandwidth, and ensure clear visibility of feature progress and associated issues.

**Glossary**

- CEP: CI Enhancement Proposal
- EF: Enhancement Freeze
- CF: Code Freeze
- RC: Release Candidate

## Process

1.  **Visibility and Tracking**: The Author of a CEP will open an issue in the `kubevirt/project-infra` repository to track their progress, maturity stages, list the associated bugs, and user feedback.

2.  **CEP Creation**: CEP authors will initiate proposals via PRs to the `kubevirt/project-infra` repository, adding a new document under the `docs/ci-enhancements` directory. The template for CEPs should be used.

3.  **SIG Review and Collaboration**: Each CEP will be owned by SIG-CI. The SIG will assign dedicated reviewers to oversee the proposal, collaborate with other SIGs as needed, and provide feedback. The author and reviewers work towards a merged CEP.

4.  **Centralized Prioritization**: At the start of each release cycle, all accepted CEPs will be designated as a priority for SIG-CI, focusing community efforts on the associated pull requests. Acceptance will be based on community support for the CEP and a commitment from the CEP owner to implement the work before Code Freeze.

    - On the EF, approved CEPs will be announced to the kubevirt-dev mailing list in order to draw attention to them.
    - Each week the release team and SIG-CI will check tracked CEPs in order to assure activity.

5.  **Single source of truth**: Each CEP will be the authoritative reference for the associated feature. It will ensure that each enhancement includes all the relevant information, including the design and the state.

### Responsibilities

#### CEP Owner

The CEP owner is responsible to update it as its development progresses, until it is fully mature (or deprecated).
In addition, they are encouraged to do the following:

- Talk about the design in the SIG-CI meetings to bring everyone on the same page about the problem, use-cases/design etc...
- Join the relevant SIG calls for further discussions, and when seeking reviews if it impacts other SIGs.

#### SIG-CI

The responsibility of SIG-CI is to do their best to help ensure sure the CEP is implemented, not diverging, and that the
implementation is not lacking behind the CEP, following this non-exhaustive list meant as checklist:

1.  After Code Freeze, SIG-CI needs to go over a tracking issue and perform the following checklist:
    1.  All PRs are merged into release branch
    2.  Docs PR is merged
    3.  Verify that the Enhancement was implemented and doesn't need any update or exception
    4.  Track any bugs
    5.  Make sure the CEP's issue is tracking required PRs and Issues
2.  Weekly check-in on progress of the Enhancement and its implementation
3.  Coordinate with other SIGs, reviewers and approvers in order to progress the Enhancement

### Release check-ins

Both the release team and approvers of the CEPs are responsible for weekly check-ins, the outcomes of which will be posted on
the CEP's tracking issue. The following are the goals of the
check-ins:

1.  Re-targeting of CEP - In case of implementation not converging, new blockers being discovered, pushback of community
    or withdrawal of an approver, the CEP may need to be re-targeted to a different release. In this case, the CEP needs
    to be updated with the new target and SIG-CI should shift the focus on tracked CEPs.
    Re-targeting could also be rejection of the CEP completely in case it is not implementable.

2.  Coordination - SIG-CI is responsible to ensure reviews are not lagging behind by more than a week.
    The release team makes sure there is always an active SIG representative.

### Labels

For easier management of the release and CEPs the following labels will be used:
1.  `sig/ci` label will be used for all CEPs.
2.  Target labels - There will be a label in order to target the CEP for release.

> [!NOTE]
> Acceptance of an enhancement doesn't guarantee that the feature will land in the current or later release.
> The process is collaborative effort between contributors and approvers.
> Features not landing in the release branch prior to CF will need to file for an exception,
> see [Exceptions](#Exceptions)

## Deadlines

The particular deadlines are always changing based on the release and are published here: [kubevirt/sig-release](https://github.com/kubevirt/sig-release).
The following deadlines are important for the CEP:

1.  CEP planning - at the beginning of every release cycle, SIG-CI would prioritize CEPs and decide which ones are being tracked for the upcoming release.
2.  Enhancement Freeze - The deadline for this milestone is Alpha release of KubeVirt. See [kubevirt/sig-release/releases](https://github.com/kubevirt/sig-release/releases)
3.  Code Freeze - This is tracked by each release [kubevirt/sig-release/releases](https://github.com/kubevirt/sig-release/releases)

## Exceptions

Exceptions are served for any edge case that is not specified in this document, by the release team/repository or within
the KubeVirt repository.
Typically, an exception would be asked to allow contributors to continue to working on CEP/PRs/code after the EF or CF
respectively.
Exceptions can be asked before the actual EF/CF.

**How to ask for exception?**
A request for exception must be sent to the [kubevirt-dev](https://groups.google.com/forum/#!forum/kubevirt-dev)
mailing list, the following should not be missing:

1.  Justification for exception
2.  Additional time period that is required
3.  In case of exception not being granted, what is the impact? (Think about graduation, maturity of the feature, user
    impact, etc.)

## Common Questions

**Do PRs need to be approved by CEP approvers?**
No, it is the whole SIG responsibility to be approving their code. The approver should be aware of the CEP and approve
based on it. There is a process to ensure this happens.

**What to do in case all PRs didn't make it before CF?**
The author of the CEP needs to file for the exception [Exceptions](#Exceptions). The outcome will be determined
individually based on context by maintainers.

**How to raise attention for my CEP?**
SIG-CI has a recurring meeting. The CEP owner is encouraged to join the meeting and introduce the CEP to the community.
