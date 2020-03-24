/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Simple abstraction for storing state on a k8s ConfigMap. Very very simple
// and uses a single entry in the ConfigMap.data for storing serialized
// JSON of the generic data that Load/Save uses. Handy for things like sources
// that need to persist some state (checkpointing for example).
package configstore

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"knative.dev/pkg/logging"
)

const (
	configdatakey = "configdata"
)

type ConfigStore struct {
	cmClient  v1.ConfigMapInterface
	name      string
	namespace string
	data      string
}

func NewConfigStore(name string, namespace string, cmClient v1.ConfigMapInterface) *ConfigStore {
	return &ConfigStore{name: name, namespace: namespace, cmClient: cmClient}
}

// Initialize ConfigStore. Either fetches an existing store or creates one
// from the provided value.
func (cs *ConfigStore) Init(ctx context.Context, value interface{}) error {
	l := logging.FromContext(ctx)
	l.Info("Initializing ConfigStore...")

	err := cs.loadConfigMapData()
	if apierrors.IsNotFound(err) {
		l.Info("No config found, creating empty")
		err = cs.createConfigMapData(value)
		if err != nil {
			l.Info("Failed to create empty configmap: %s\n", err)
		} else {
			l.Info("Empty config created successfully")
			err = cs.loadConfigMapData()
			if err == nil {
				l.Info("Config loaded succsesfully")
				return nil
			} else {
				l.Error("Failed to load configmap data: %s\n", err)
			}
		}
	}
	return err
}

// Load fetches the ConfigMap from k8s and unmarshals the data found
// in the configdatakey type as specified by value.
func (cs *ConfigStore) Load(value interface{}) error {
	err := cs.loadConfigMapData()
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(cs.data), value)
	if err != nil {
		return fmt.Errorf("Failed to Unmarshal: %v", err)
	}
	return nil
}

// Save takes the value given in, and marshals it into a string
// and saves it into the k8s ConfigMap under the configdatakey.
func (cs *ConfigStore) Save(value interface{}) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("Failed to Marshal: %v", err)
	}
	cs.data = string(bytes)
	return cs.saveConfigMapData()
}

func (cs *ConfigStore) createConfigMapData(value interface{}) error {
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("Failed to Marshal: %v", err)
	}
	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cs.name,
			Namespace: cs.namespace,
		},
		Data: map[string]string{configdatakey: string(bytes)},
	}
	_, err = cs.cmClient.Create(cm)
	return err
}

// loadConfigMapData loads the ConfigMap and grabs the configmapkey value from
// the map that contains our state.
func (cs *ConfigStore) loadConfigMapData() error {
	cm, err := cs.cmClient.Get(cs.name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	cs.data = cm.Data[configdatakey]
	return nil
}

// saveConfigMapData saves the ConfigMap with the data from ConfigStore.data
// stored in the configmapkey value of that map.
func (cs *ConfigStore) saveConfigMapData() error {
	cm, err := cs.cmClient.Get(cs.name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	cm.Data[configdatakey] = cs.data
	_, err = cs.cmClient.Update(cm)
	return err

}
