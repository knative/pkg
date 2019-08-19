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

package resolver

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	resolverFileName  = "/etc/resolv.conf"
	defaultDomainName = "cluster.local"
)

var (
	domainName     string
	domainNameOnce sync.Once
)

// ClusterDomainName fetches the cluster's domain name.
func ClusterDomainName() string {
	domainNameOnce.Do(func() {
		f, err := os.Open(resolverFileName)
		if err != nil {
			domainName = defaultDomainName
			return
		}
		defer f.Close()
		domainName = readClusterDomainName(f)
	})

	return domainName
}

func readClusterDomainName(r io.Reader) string {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		elements := strings.Split(scanner.Text(), " ")
		if elements[0] != "search" {
			continue
		}
		for i := 1; i < len(elements)-1; i++ {
			if strings.HasPrefix(elements[i], "svc.") {
				return elements[i][4:]
			}
		}
	}
	// For all abnormal cases return default domain name
	return defaultDomainName
}

// ServiceHostName resolves the hostname for a Kubernetes Service.
func ServiceHostName(serviceName, namespace string) string {
	return fmt.Sprintf("%s.%s.svc.%s", serviceName, namespace, ClusterDomainName())
}
