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

import base
import urllib2
import json

class PullRequestResult(object):

    def __init__(self, context, status, message=None):
        self.context = context
        self.status = status
        self.message = message

class CommitCheckHook(base.Hook):

    def _request(self, url, data=None):
        req = urllib2.Request(url, data=data,
                              headers={
                                  "Content-Type": "application/json",
                                  "Authorization": "token " + self.auth_token,
                              })

        opener = urllib2.build_opener()
        return opener.open(req)

    def _set_status(self, repo, commit, result):
        sha = commit["sha"]
        url = "https://api.github.com" + "/repos/" + repo + "/statuses/" + sha

        status = {
            "context": result.context,
            "state": result.status
        }
        if result.message != None:
            status["description"] = result.message

        res = self._request(url, json.dumps(status))

    def check_commit(self, commit):
        raise NotImplementedError()

    def run(self, data):
        if "pull_request" not in data:
            return "OK"

        url = data["pull_request"]["commits_url"]

        repo = data["repository"]["full_name"]

        resp = urllib2.urlopen(url)
        commits = json.load(resp)

        for commit in commits:
            result = self.check_commit(commit)

            self._set_status(repo, commit, result)

        return "OK"
