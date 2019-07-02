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

FROM golang:latest

ENV GOPATH /go
ENV PATH $PATH:/etc/gha2db:/${GOPATH}/bin

RUN apt-get update -y && \
				apt-get install -y apt-transport-https \
				git psmisc jsonlint yamllint gcc

RUN go get -u github.com/golang/lint/golint && \
  go get golang.org/x/tools/cmd/goimports

RUN go get github.com/jgautheron/goconst/cmd/goconst && \
	go get github.com/jgautheron/usedexports

RUN go get github.com/kisielk/errcheck && \
	go get github.com/lib/pq

RUN go get golang.org/x/text/transform && \
	go get golang.org/x/text/unicode/norm

RUN go get github.com/google/go-github/github && \
	go get golang.org/x/oauth2

RUN go get gopkg.in/yaml.v2 && \
	go get github.com/mattn/go-sqlite3

RUN apt-get install -y vim

#POSTGRES INSTALLATION
RUN apt-get install -y postgresql-client postgresql-9.6 sudo gosu
ENV PG_MAJOR 9.6
ENV PATH $PATH:/usr/lib/postgresql/$PG_MAJOR/bin
RUN adduser postgres sudo

#DEVSTATS INSTALLATION
RUN mkdir -p ${GOPATH}/src
WORKDIR ${GOPATH}/src
RUN git clone https://github.com/cncf/devstats
WORKDIR ${GOPATH}/src/devstats
RUN git checkout 56f581a2f03d6fd9f718faa9c2b1a885e1e9076f
RUN make
RUN make install
COPY ["scripts/setup_mount.sh", "scripts/setup_db.sh", "${GOPATH}/src/"]
