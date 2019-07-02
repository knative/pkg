# Copyright 2019 The Knative Authors
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

FROM postgres:9.6

COPY postgre-docker-entrypoint.sh /usr/local/bin/
RUN chmod a+x /usr/local/bin/postgre-docker-entrypoint.sh
COPY postgresql.conf /tmp/postgresql.conf
COPY pg_hba.conf /tmp/pg_hba.conf
COPY updateConfig.sh      /docker-entrypoint-initdb.d/_updateConfig.sh
