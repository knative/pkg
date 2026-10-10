/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// LeaderElection provides an interface for controllers implementing using
// controller injection:
// https://github.com/knative/pkg/blob/main/injection/README.md
//
// Leaderelection uses the context-stuffing mechanism to provide config-driven
// management of multiple election strategies (currently, using Kubernetes
// etcd-based election primitives or StatefulSet indexes and counts).
//
// StatefulSet ordinal mode is 1:1: ordinal N owns bucket N, whose identity is
// the DNS name of pod N. It is selected when STATEFUL_CONTROLLER_ORDINAL is
// set. STATEFUL_REPLICA_COUNT must equal the configured bucket count; extra
// buckets are not redistributed. An explicitly configured but invalid
// StatefulSet topology fails rather than falling back to standard election.
//
// For more details, see the original design document:
// https://docs.google.com/document/d/e/2PACX-1vTh40N-Kk6EPNzYpITiLg8YJk0qZyZv7KgMpcQS72T9Lv_F2PQeGybx4TtH0E1N1aUgLQer7b8u3lDc/pub
package leaderelection
