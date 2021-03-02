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

package example

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type LoremIpsum struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state.
	Spec LoremIpsumSpec `json:"spec,omitempty"`

	// Status represents the current state. This data may be out of date.
	// +optional
	Status LoremIpsumStatus `json:"status,omitempty"`
}

type LoremIpsumSpec struct {
	IpsumSpec `json:",inline"`

	// Maecenas tristique lobortis turpis, nec varius mauris vestibulum nec.
	// Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere
	// cubilia curae; Vivamus non dapibus magna.
	Maecenas string `json:"maecenas,omitempty"`

	// Aaa is the first way.
	Aaa LoremSpec `json:"aaa,omitempty"`

	// Bbb is the second way.
	Bbb LoremSpec `json:"bbb,omitempty"`

	// VerboseTypes shows an example of a ton of types.
	VerboseTypes VerboseTypes `json:"verboseTypes"`
}

type IpsumSpec struct {
	// Sed euismod nunc ac sollicitudin ornare.
	// +optional
	Sed string `json:"sed,omitempty"`
}

type LoremSpec struct {
	// Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nullam
	// pellentesque eget arcu eget porta. Morbi ex urna, tincidunt in odio
	// eget, hendrerit mattis odio. Sed vel augue rhoncus, rhoncus mi eget,
	// tempor nisi. Nullam eleifend scelerisque pellentesque. Fusce efficitur
	// urna mauris, sed suscipit sapien rhoncus et. Nunc viverra porta libero,
	// mattis venenatis orci. Pellentesque molestie egestas iaculis. Donec
	// sodales tristique ex, eget consectetur elit rutrum sed. Proin mollis,
	// tellus vitae lobortis pretium, lacus dolor rhoncus tellus, at ultrices
	// elit mauris vel enim. Suspendisse tempor ligula a est posuere, in
	// egestas eros vehicula. Nulla mi magna, cursus in ultrices eget,
	// porttitor eu odio. Nunc augue nisi, molestie at laoreet ut, sagittis a
	// libero. Ut ullamcorper leo lectus, vel placerat ipsum lacinia vitae.
	// Morbi commodo nibh neque, in ornare diam sodales ac.
	// Defaults to true.
	// +optional
	Lorem *bool `json:"lorem,omitempty"`

	// Praesent pulvinar consectetur enim. Aenean lobortis, eros quis molestie
	// euismod, nisl nunc mattis quam, et gravida risus diam at nulla. Donec
	// interdum, tortor a semper tincidunt, nibh odio euismod orci, rhoncus
	// rhoncus purus lacus pharetra mi. Suspendisse placerat dignissim magna
	// convallis dictum. Nulla facilisi. Vivamus sed tristique turpis.
	Praesent string `json:"praesent,omitempty"`

	// Ccc shows loop protection.
	Ccc *LoremSpec `json:"ccc,omitempty"`
}

type LoremIpsumStatus struct {
	// Luctus leo vitae ipsum fermentum, vitae pellentesque sapien finibus.
	Luctus int `json:"luctus"`

	// Suspendisse ipsum risus, porttitor a auctor vel, maximus eu mi.
	Suspendisse string `json:"suspendisse"`

	// Aliquam consequat placerat ante, eu ullamcorper purus consectetur quis.
	Aliquam []string `json:"aliquam,omitempty"`

	// Donec mollis purus id ipsum varius, sit amet ultricies elit laoreet.
	// +optional
	Donec *bool `json:"donec,omitempty"`
}

type VerboseTypes struct {
	// AInt8 is a field with the type int8.
	AInt8 int8 `json:"int8"`
	// AInt16 is a field with the type int16.
	AInt16 int16 `json:"int16"`
	// AInt32 is a field with the type int32.
	AInt32 int32 `json:"int32"`
	// AInt64 is a field with the type int64.
	AInt64 int64 `json:"int63"`
	// AUint is a field with the type uint.
	AUint uint `json:"uint"`
	// Uint8 is a field with the type uint8.
	Uint8 uint8 `json:"uint8"`
	// AUint16 is a field with the type uint16.
	AUint16 uint16 `json:"uint16"`
	// AUint32 is a field with the type uint32.
	AUint32 uint32 `json:"uint32"`
	// AUint64 is a field with the type uint64.
	AUint64 uint64 `json:"uint64"`
	// AFloat32 is a field with the type float32.
	AFloat32 float32 `json:"float32"`
	// AFloat64 is a field with the type float64.
	AFloat64 float64 `json:"float64"`
	// AMap is a field with the type map.
	AMap map[string]string `json:"map"`
}
