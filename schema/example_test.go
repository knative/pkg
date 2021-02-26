/*
Copyright 2021 The Knative Authors

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

package main

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"

	"knative.dev/pkg/schema/docs"
	"knative.dev/pkg/schema/example"
	"knative.dev/pkg/schema/schema"
)

func Example_kindLaremIpsum() {
	docs.SetRoot("knative.dev/pkg/schema")

	t := reflect.TypeOf(example.LaremIpsum{})
	s := schema.GenerateForType(t)
	b, _ := yaml.Marshal(s)
	fmt.Print(string(b))

	// Output:
	// type: object
	// properties:
	//     spec:
	//         description: Spec defines the desired state.
	//         type: object
	//         properties:
	//             aaa:
	//                 description: Aaa is the first way.
	//                 type: object
	//                 required:
	//                   - ccc
	//                 properties:
	//                     ccc:
	//                         description: Ccc shows loop protection.
	//                         type: object
	//                     lelarem:
	//                         description: Lorem ipsum dolor sit amet, consectetur adipiscing
	//                             elit. Nullam pellentesque eget arcu eget porta. Morbi
	//                             ex urna, tincidunt in odio eget, hendrerit mattis odio.
	//                             Sed vel augue rhoncus, rhoncus mi eget, tempor nisi. Nullam
	//                             eleifend scelerisque pellentesque. Fusce efficitur urna
	//                             mauris, sed suscipit sapien rhoncus et. Nunc viverra porta
	//                             libero, mattis venenatis orci. Pellentesque molestie egestas
	//                             iaculis. Donec sodales tristique ex, eget consectetur
	//                             elit rutrum sed. Proin mollis, tellus vitae lobortis pretium,
	//                             lacus dolor rhoncus tellus, at ultrices elit mauris vel
	//                             enim. Suspendisse tempor ligula a est posuere, in egestas
	//                             eros vehicula. Nulla mi magna, cursus in ultrices eget,
	//                             porttitor eu odio. Nunc augue nisi, molestie at laoreet
	//                             ut, sagittis a libero. Ut ullamcorper leo lectus, vel
	//                             placerat ipsum lacinia vitae. Morbi commodo nibh neque,
	//                             in ornare diam sodales ac. Defaults to true.
	//                         type: boolean
	//                     praesent:
	//                         description: Praesent pulvinar consectetur enim. Aenean lobortis,
	//                             eros quis molestie euismod, nisl nunc mattis quam, et
	//                             gravida risus diam at nulla. Donec interdum, tortor a
	//                             semper tincidunt, nibh odio euismod orci, rhoncus rhoncus
	//                             purus lacus pharetra mi. Suspendisse placerat dignissim
	//                             magna convallis dictum. Nulla facilisi. Vivamus sed tristique
	//                             turpis.
	//                         type: string
	//             bbb:
	//                 description: Bbb is the second way.
	//                 type: object
	//                 required:
	//                   - ccc
	//                 properties:
	//                     ccc:
	//                         description: Ccc shows loop protection.
	//                         type: object
	//                     lelarem:
	//                         description: Lorem ipsum dolor sit amet, consectetur adipiscing
	//                             elit. Nullam pellentesque eget arcu eget porta. Morbi
	//                             ex urna, tincidunt in odio eget, hendrerit mattis odio.
	//                             Sed vel augue rhoncus, rhoncus mi eget, tempor nisi. Nullam
	//                             eleifend scelerisque pellentesque. Fusce efficitur urna
	//                             mauris, sed suscipit sapien rhoncus et. Nunc viverra porta
	//                             libero, mattis venenatis orci. Pellentesque molestie egestas
	//                             iaculis. Donec sodales tristique ex, eget consectetur
	//                             elit rutrum sed. Proin mollis, tellus vitae lobortis pretium,
	//                             lacus dolor rhoncus tellus, at ultrices elit mauris vel
	//                             enim. Suspendisse tempor ligula a est posuere, in egestas
	//                             eros vehicula. Nulla mi magna, cursus in ultrices eget,
	//                             porttitor eu odio. Nunc augue nisi, molestie at laoreet
	//                             ut, sagittis a libero. Ut ullamcorper leo lectus, vel
	//                             placerat ipsum lacinia vitae. Morbi commodo nibh neque,
	//                             in ornare diam sodales ac. Defaults to true.
	//                         type: boolean
	//                     praesent:
	//                         description: Praesent pulvinar consectetur enim. Aenean lobortis,
	//                             eros quis molestie euismod, nisl nunc mattis quam, et
	//                             gravida risus diam at nulla. Donec interdum, tortor a
	//                             semper tincidunt, nibh odio euismod orci, rhoncus rhoncus
	//                             purus lacus pharetra mi. Suspendisse placerat dignissim
	//                             magna convallis dictum. Nulla facilisi. Vivamus sed tristique
	//                             turpis.
	//                         type: string
	//             maecenas:
	//                 description: Maecenas tristique lobortis turpis, nec varius mauris
	//                     vestibulum nec. Vestibulum ante ipsum primis in faucibus orci
	//                     luctus et ultrices posuere cubilia curae; Vivamus non dapibus
	//                     magna.
	//                 type: string
	//             sed:
	//                 description: Sed euismod nunc ac sollicitudin ornare.
	//                 type: string
	//     status:
	//         description: Status represents the current state. This data may be out of
	//             date.
	//         type: object
	//         properties:
	//             aliquam:
	//                 description: Aliquam consequat placerat ante, eu ullamcorper purus
	//                     consectetur quis.
	//                 type: array
	//                 items:
	//                     type: string
	//             lelarem:
	//                 description: Donec mollis purus id ipsum varius, sit amet ultricies
	//                     elit laoreet.
	//                 type: boolean
	//             luctus:
	//                 description: Luctus leo vitae ipsum fermentum, vitae pellentesque
	//                     sapien finibus.
	//                 type: integer
	//                 format: int32
	//             suspendisse:
	//                 description: Suspendisse ipsum risus, porttitor a auctor vel, maximus
	//                     eu mi.
	//                 type: string
}

// TODO: there is a bug where ROOT can't go up a level without looking in vendor, but it really is just a problem with Examples and because the file finder needs to be go mod aware.
//func Example_kindKResource() {
//	docs.SetRoot("knative.dev/pkg/schema")
//
//	t := reflect.TypeOf(duckv1.KResource{})
//	s := schema.GenerateForType(t)
//	b, _ := yaml.Marshal(s)
//	fmt.Print(string(b))
//
//	// Output:
//	// type: object
//	// properties:
//	//     status:
//	//         description: 'not found: unable to parse dir: error parse dir "vendor/knative.dev/pkg/apis/duck/v1":
//	//             open vendor/knative.dev/pkg/apis/duck/v1: no such file or directory'
//	//         type: object
//	//         properties:
//	//             annotations:
//	//                 description: 'not found: unable to parse dir: error parse dir "vendor/knative.dev/pkg/apis/duck/v1":
//	//                     open vendor/knative.dev/pkg/apis/duck/v1: no such file or directory'
//	//                 type: object
//	//                 xpreserveunknownfields: true
//	//             conditions:
//	//                 description: 'not found: unable to parse dir: error parse dir "vendor/knative.dev/pkg/apis/duck/v1":
//	//                     open vendor/knative.dev/pkg/apis/duck/v1: no such file or directory'
//	//                 type: array
//	//                 items:
//	//                     type: object
//	//                     properties:
//	//                                 dir "vendor/knative.dev/pkg/apis": open vendor/knative.dev/pkg/apis:
//	//                                 no such file or directory'
//	//                             type: string
//	//             observedGeneration:
//	//                 description: 'not found: unable to parse dir: error parse dir "vendor/knative.dev/pkg/apis/duck/v1":
//	//                     open vendor/knative.dev/pkg/apis/duck/v1: no such file or directory'
//	//                 type: integer
//	//                 format: int64
//}
