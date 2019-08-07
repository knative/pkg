/*
Copyright 2019 The Knative Authors

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

package profiling

import (
	"strconv"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

const profilingKey = "profiling.enable"

// UpdateProfilingFromConfigMap modifies the Enabled flag in the Handler that is passed
// as an argument, according to the value in the given ConfigMap
func UpdateProfilingFromConfigMap(profilingHandler *Handler, logger *zap.SugaredLogger) func(configMap *corev1.ConfigMap) {
	return func(configMap *corev1.ConfigMap) {
		if profiling, ok := configMap.Data[profilingKey]; ok {
			if enabled, err := strconv.ParseBool(profiling); err == nil {
				logger.Infof("Profiling enabled: %t", enabled)
				profilingHandler.Enabled = enabled
			} else {
				logger.Errorw("Failed to update profiling", zap.Error(err))
			}
		}
	}
}
