#!/bin/bash

# Copyright 2019 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

mkdir -p ./grafana/dashboards/knative
dashboards="companies-stats.json companies-summary.json contributing-companies.json dashboards.json developers-summary.json first-non-author-activity.json github-events.json issues-age.json issues-repository-group.json new-and-episodic-issue-creators.json new-and-episodic-pr-contributors.json new-contributors-table.json new-prs.json opened-to-merged.json pr-comments.json project-statistics.json prs-age.json prs-authors-companies-histogram.json prs-authors-histogram.json prs-authors.json prs-merged-repository-groups.json repository-comments.json timezones-stats.json top-commenters.json user-reviews.json"
for f in $dashboards
do
  cp $GOPATH/src/devstats/grafana/dashboards/knative/$f ./grafana/dashboards/knative || exit 1
done
echo "Dashboards copied."
