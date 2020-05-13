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

package configmap

import (
	"strconv"
	"strings"
	"time"
)

func AsBool(key string, target *bool) func(map[string]string) error {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			*target = strings.EqualFold(raw, "true")
		}
		return nil
	}
}

func AsInt32(key string, target *int32) func(map[string]string) error {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return err
			}
			*target = int32(val)
		}
		return nil
	}
}

func AsInt64(key string, target *int64) func(map[string]string) error {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return err
			}
			*target = val
		}
		return nil
	}
}

func AsFloat64(key string, target *float64) func(map[string]string) error {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return err
			}
			*target = val
		}
		return nil
	}
}

func AsDuration(key string, target *time.Duration) func(map[string]string) error {
	return func(data map[string]string) error {
		if raw, ok := data[key]; ok {
			val, err := time.ParseDuration(raw)
			if err != nil {
				return err
			}
			*target = val
		}
		return nil
	}
}

func Parse(data map[string]string, parsers ...func(map[string]string) error) error {
	for _, parse := range parsers {
		if err := parse(data); err != nil {
			return err
		}
	}
	return nil
}
