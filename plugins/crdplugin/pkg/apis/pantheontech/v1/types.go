/*
 * Copyright (c) 2020 PANTHEON.tech s.r.o. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1

import (
	contiv_crd "github.com/contiv/vpp/plugins/crd/pkg/apis/contivppio/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// StatusSuccess is returned in Status.Status when controller successfully creates/deletes/updates CRD.
	StatusSuccess = "Success"
	// StatusFailure is returned in Status.Status when controller fails to create/delete/update CRD.
	StatusFailure = "Failure"
)

// CNFConfiguration defines (arbitrary) configuration to be applied for Pantheon CNFs.
// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CNFConfiguration struct {
	// TypeMeta is the metadata for the resource, like kind and apiversion
	meta_v1.TypeMeta `json:",inline"`
	// ObjectMeta contains the metadata for the particular object
	meta_v1.ObjectMeta `json:"metadata,omitempty"`
	// Spec is the specification for the CNF configuration.
	Spec CNFConfigurationSpec `json:"spec"`
	// Status informs about the status of the resource.
	Status meta_v1.Status `json:"status,omitempty"`
}

// CNFConfigurationSpec is the spec for CNF configuration resource.
// It is largely based on CustomConfiguration resource from Contiv/VPP.
type CNFConfigurationSpec struct {
	// Microservice label determines where the configuration item should be applied.
	// Use label as defined in the environment variable MICROSERVICE_LABEL of the destination CNF.
	// This microservice label will be used for all items in the list below which do not have microservice defined.
	Microservice string `json:"microservice"`
	// Items is a list of configuration items.
	ConfigItems []contiv_crd.ConfigurationItem `json:"configItems"`
}

// CNFConfigurationList is a list of CNFConfiguration resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CNFConfigurationList struct {
	meta_v1.TypeMeta `json:",inline"`
	meta_v1.ListMeta `json:"metadata"`

	Items []CNFConfiguration `json:"items"`
}
