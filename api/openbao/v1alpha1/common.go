/*
Copyright 2026 openbao-entity-operator contributors.

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

package v1alpha1

// CreationPolicy controls how an external entity is acquired.
type CreationPolicy string

const (
	CreationPolicyCreate        CreationPolicy = "Create"
	CreationPolicyAdopt         CreationPolicy = "Adopt"
	CreationPolicyCreateOrAdopt CreationPolicy = "CreateOrAdopt"
)

func (p CreationPolicy) AllowsCreation() bool {
	return p == "" || p == CreationPolicyCreate || p == CreationPolicyCreateOrAdopt
}

func (p CreationPolicy) AllowsAdoption() bool {
	return p == CreationPolicyAdopt || p == CreationPolicyCreateOrAdopt
}

// DeletionPolicy controls what happens to an external entity on deletion.
type DeletionPolicy string

const (
	DeletionPolicyDelete DeletionPolicy = "Delete"
	DeletionPolicyOrphan DeletionPolicy = "Orphan"
)

func (p DeletionPolicy) IsDelete() bool {
	return p == DeletionPolicyDelete
}

// OpenBaoConnectionReference identifies a same-namespace OpenBaoConnection.
type OpenBaoConnectionReference struct {
	// Name is the OpenBaoConnection resource name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// OpenBaoEntityReference identifies a same-namespace OpenBaoEntity.
type OpenBaoEntityReference struct {
	// Name is the OpenBaoEntity resource name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// OpenBaoGroupReference identifies a same-namespace OpenBaoGroup.
type OpenBaoGroupReference struct {
	// Name is the OpenBaoGroup resource name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
}

// SecretKeyReference identifies a key in a same-namespace Secret.
type SecretKeyReference struct {
	// Name is the Secret resource name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Key is the Secret data key. It defaults to token for token references.
	// +optional
	Key string `json:"key,omitempty"`
}
