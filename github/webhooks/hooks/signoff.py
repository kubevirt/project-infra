# Copyright 2017 Red Hat, Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import pullrequest
import re

class SignoffHook(pullrequest.CommitCheckHook):

    SOB = re.compile("^\s*Signed-off-by:.*")
    
    def check_commit(self, commit):
        message = commit["commit"]["message"]
        lines = message.split("\n")
        ok = False

        for line in lines:
            if self.SOB.match(line):
                ok = True

        if ok:
            return pullrequest.PullRequestResult(
                "Signed-off-by checker",
                "success")
        else:
            return pullrequest.PullRequestResult(
                "Signed-off-by checker",
                "failure",
                "Commit message must be signed off")
