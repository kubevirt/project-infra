# repo.gitbundle

Contains the git repository that the tests in this folder operate upon.

The bundle file is extracted to a full git repository under `repo/` in the suite tests.
This is done since we don't want to fiddle with git submodules and stuff and git
complains whenever we want to add the git repo residing in `repo/`.

## updating the repo

whenever changes in the repo are necessary we need to update the bundle file
from within the `repo/` folder like so:

```bash
git bundle create ../repo.gitbundle --all
```
