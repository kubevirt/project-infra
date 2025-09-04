# Example PR Description with docs-pr Field

This document shows examples of how PR descriptions should be formatted with the new `docs-pr` field.

## Example 1: PR with Documentation Update

```markdown
**What this PR does / why we need it**:
This PR adds support for new VM migration features, allowing users to migrate VMs with persistent volumes attached.

**Which issue(s) this PR fixes**:
Fixes #1234

**Documentation update**:
<!-- Add your Docs PR number
1. Enter the related docs PR number in the field below. This is expected to be but not required to be in the kubevirt/user-guide repo. 
2. If you have multiple docs PRs, just include one in this field and include the others as comments in the PR. You do not need to add the docs PR to create the PR but it will be required to merge. You can add it at a later time by editing the PR description. The docs PR does not need to be merged for this PR to merge.
3. If a docs update is not required, just write "NONE".-->
```docs-pr
kubevirt/user-guide#567
```

**Special notes for your reviewer**:
This change requires careful testing with different storage backends.
```

**Result**: Plugin will add `docs-pr` label and allow merge.

## Example 2: PR with No Documentation Needed

```markdown
**What this PR does / why we need it**:
This PR fixes a minor typo in error message logging.

**Which issue(s) this PR fixes**:
Fixes #5678

**Documentation update**:
<!-- Add your Docs PR number -->
```docs-pr
NONE
```

**Special notes for your reviewer**:
Simple typo fix, no user-facing changes.
```

**Result**: Plugin will add `docs-pr-none` label and allow merge.

## Example 3: PR Missing docs-pr Field

```markdown
**What this PR does / why we need it**:
This PR adds new API endpoints for VM management.

**Which issue(s) this PR fixes**:
Fixes #9999

**Special notes for your reviewer**:
This introduces new APIs that need documentation.
```

**Result**: Plugin will add `do-not-merge/docs-pr-required` label and block merge.

## Example 4: PR with Empty docs-pr Field

```markdown
**What this PR does / why we need it**:
This PR changes VM behavior significantly.

**Documentation update**:
```docs-pr

```

**Special notes for your reviewer**:
Major behavior change.
```

**Result**: Plugin will add `do-not-merge/docs-pr-required` label and block merge.

## Using the /docs-pr Command

Users can also update the docs-pr status using comments:

- `/docs-pr #1234` - Set docs PR to #1234
- `/docs-pr NONE` - Mark as not needing docs
- `/docs-pr kubevirt/user-guide#567` - Set docs PR with full repo reference

The plugin will update the PR description and labels accordingly.