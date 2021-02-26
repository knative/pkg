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

type LaremIpsum struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired state.
	Spec LaremIpsumSpec `json:"spec,omitempty"`

	// Status represents the current state. This data may be out of date.
	// +optional
	Status LaremIpsumStatus `json:"status,omitempty"`
}

type LaremIpsumSpec struct {
	IpsumSpec `json:",inline"`

	// Maecenas tristique lobortis turpis, nec varius mauris vestibulum nec.
	// Vestibulum ante ipsum primis in faucibus orci luctus et ultrices posuere
	// cubilia curae; Vivamus non dapibus magna.
	Maecenas string `json:"maecenas,omitempty"`

	// Aaa is the first way.
	Aaa LaremSpec `json:"aaa,omitempty"`

	// Bbb is the second way.
	Bbb LaremSpec `json:"bbb,omitempty"`
}

type IpsumSpec struct {
	// Sed euismod nunc ac sollicitudin ornare.
	// +optional
	Sed string `json:"sed,omitempty"`
}

type LaremSpec struct {
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
	Larem *bool `json:"lelarem,omitempty"`

	// Praesent pulvinar consectetur enim. Aenean lobortis, eros quis molestie
	// euismod, nisl nunc mattis quam, et gravida risus diam at nulla. Donec
	// interdum, tortor a semper tincidunt, nibh odio euismod orci, rhoncus
	// rhoncus purus lacus pharetra mi. Suspendisse placerat dignissim magna
	// convallis dictum. Nulla facilisi. Vivamus sed tristique turpis.
	Praesent string `json:"praesent,omitempty"`

	// Ccc shows loop protection.
	Ccc *LaremSpec `json:"ccc,omitempty"`
}

type LaremIpsumStatus struct {
	// Luctus leo vitae ipsum fermentum, vitae pellentesque sapien finibus.
	Luctus int `json:"luctus"`

	// Suspendisse ipsum risus, porttitor a auctor vel, maximus eu mi.
	Suspendisse string `json:"suspendisse"`

	// Aliquam consequat placerat ante, eu ullamcorper purus consectetur quis.
	Aliquam []string `json:"aliquam,omitempty"`

	// Donec mollis purus id ipsum varius, sit amet ultricies elit laoreet.
	// +optional
	Larem *bool `json:"lelarem,omitempty"`
}
