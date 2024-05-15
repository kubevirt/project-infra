/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright the KubeVirt Authors.
 *
 */

/*
Package metrics contains the relevant metrics that the referee plugin tracks.

# Retest metrics

Retest rate shows the stability of the current test situation. We have observed that whenever there is a surge in retests, a stability problem has occurred. Examples recalled:

  - sudden rise in retests shown after a test got flaky
  - sudden rise in retests after an infrastructure issue occurred (i.e. quay outage)
  - sudden rise in retests after an instability inside a PR surfaced, which was not properly investigated

The latter is distinct from the former since the issuing of the retest
occurs on one specific PR, thus the rate increase in general is not that high

# Goals

  - Be alerted if the rate of retests suddenly rises
  - Have an overview of the current state per PR

# Approach

To achieve the goals it makes sense to monitor two kinds of metrics:

  - the rate of observed retest commands will give an overview of stability, an alert on a sudden surge will make us aware that a problem is present
  - the current state per PR will give an overview if there’s a more specific issue with using retests, i.e. a dedicated instability caused by changes from a PR - an alert beyond the magic boundary will make us aware that there’s a problem
*/
package metrics
