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
// management of multiple election strategies using Kubernetes leader election
// primitives or StatefulSet pod ordinals.
//
// To run every replica in StatefulSet ordinal mode, use zero-based pod ordinals
// and keep the StatefulSet replica count equal to the configured bucket count
// when deploying or scaling. If there are fewer replicas than buckets, some
// buckets have no owner and reconciliation for objects assigned to them stalls.
// A replica whose ordinal is outside the configured bucket range cannot use
// ordinal mode. WithDynamicLeaderElectorBuilder falls back to Lease-based
// leader election for that replica, which can lead to duplicate reconciliation
// alongside replicas in ordinal mode. This package does not read the StatefulSet
// replica count. Controllers can check the desired replica count against the
// bucket count at startup.
//
// For more details, see the original design document:
// https://docs.google.com/document/d/e/2PACX-1vTh40N-Kk6EPNzYpITiLg8YJk0qZyZv7KgMpcQS72T9Lv_F2PQeGybx4TtH0E1N1aUgLQer7b8u3lDc/pub
package leaderelection
