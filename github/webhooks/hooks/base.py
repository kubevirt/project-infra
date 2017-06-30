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

import hashlib
import hmac
from flask import abort, request


class Hook(object):

    def __init__(self, sig_token, auth_token):
        self.sig_token = sig_token
        self.auth_token = auth_token

    def authorize(self):
        sig = request.headers.get("X-Hub-Signature", None)
        if sig is None:
            abort(401, "Missing signature")

        parts = sig.split("=", 1)
        if len(parts) != 2:
            abort(401, "Invalid signature")

        if parts[0] != "sha1":
            abort(401, "Invalid signature")

        actual = str(parts[1])
        digest = hmac.new(self.sig_token, request.data, hashlib.sha1)
        expected = digest.hexdigest()

        if not hmac.compare_digest(expected, actual):
            abort(401, "Invalid signature")

    def post(self):
        self.authorize()

        payload = request.get_json()

        if payload is None:
            abort(400, "Missing request payload")

        return self.run(payload)

    def run(self, payload):
        raise NotImplementedError()

    def register(self, app, path):
        app.add_url_rule(path, view_func=self.post, methods=["POST"])
