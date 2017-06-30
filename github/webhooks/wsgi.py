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

import os
from hooks import signoff
from flask import Flask

application = Flask(__name__)  # Standard Flask app

sig_token = os.environ["GITHUB_SIG_TOKEN"]
auth_token = os.environ["GITHUB_AUTH_TOKEN"]

signoff.SignoffHook(sig_token, auth_token).register(application, "/signoff")

@application.route("/healthz")
def healthz():
    return "OK"

if __name__ == "__main__":
    application.run()
