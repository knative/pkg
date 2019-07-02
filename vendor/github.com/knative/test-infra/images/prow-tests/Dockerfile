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

FROM gcr.io/k8s-testimages/kubekins-e2e:v20190103-edc7696cc-master
LABEL maintainer "Adriano Cunha <adrcunha@google.com>"

# Install extras on top of base image

ENV DEBIAN_FRONTEND noninteractive
RUN apt-get update
RUN gcloud components update

# Docker
RUN gcloud components install docker-credential-gcr
RUN docker-credential-gcr configure-docker

# Extra tools through apt-get
RUN apt-get install -y uuid-runtime  # for uuidgen
RUN apt-get install -y rubygems  # for mdl

# Install go at 1.12 same as how it's installed in kubekins-e2e image, reference code here:
# https://github.com/kubernetes/test-infra/blob/1e9b5dc3de4b268aab57c9c48120aa3dcf096bc6/images/kubekins-e2e/Dockerfile#L64
ENV GO_TARBALL "go1.12.linux-amd64.tar.gz"
RUN rm -rf /usr/local/go && \
    wget -q "https://storage.googleapis.com/golang/${GO_TARBALL}" && \
    tar xzf "${GO_TARBALL}" -C /usr/local && \
    rm "${GO_TARBALL}"

# Extra tools through go get
RUN go get -u github.com/google/ko/cmd/ko
RUN go get -u github.com/golang/dep/cmd/dep
RUN go get -u github.com/jstemmer/go-junit-report
RUN go get -u github.com/raviqqe/liche

# Extract bazel version
RUN bazel version > /bazel_version
RUN bazel shutdown

# Extract ko version
RUN git ls-remote https://github.com/google/ko HEAD | cut -f1 > /ko_version

# Extra tools through gem
RUN gem install mixlib-config -v 2.2.4  # required because ruby is 2.1
RUN gem install mdl

# Temporarily add test-infra to the image to build custom tools
ADD . /go/src/github.com/knative/test-infra

# Build custom tools in the container
RUN make -C /go/src/github.com/knative/test-infra/tools/githubhelper
RUN cp /go/src/github.com/knative/test-infra/tools/githubhelper/githubhelper .

# Use a local dep-collector that's a wrapper for building the tool
# on-the-fly.
# TODO(adrcunha): Remove once https://github.com/knative/test-infra/pull/1019 is in all repos.
RUN cp /go/src/github.com/knative/test-infra/images/prow-tests/dep-collector /usr/bin/dep-collector

# Remove test-infra from the container
RUN rm -fr /go/src/github.com/knative/test-infra
