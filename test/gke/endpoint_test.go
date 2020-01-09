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

package gke

import "testing"

func TestServiceEndpoint(t *testing.T) {
    datas := []struct {
        env  string
        want string
    }{
        {"", ""},
        {testEnv, testEndpoint},
        {stagingEnv, stagingEndpoint},
        {staging2Env, staging2Endpoint},
        {prodEnv, prodEndpoint},
        {"invalid_url", ""},
        {"https://custom.container.googleapis.com/", "https://custom.container.googleapis.com/"},
    }
    for _, data := range datas {
        if got := ServiceEndpoint(data.env); got != data.want {
            t.Errorf("Service endpoint for %q = %q, want: %q",
                data.env, got, data.want)
        }
    }
}
