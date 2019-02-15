/*

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

package v1

import (
	"fmt"

	"github.com/droot/crd-conversion-example/pkg/apis/jobs/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExternalJobSpec defines the desired state of ExternalJob
type ExternalJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// ExternalJobStatus defines the observed state of ExternalJob
type ExternalJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalJob is the Schema for the externaljobs API
// +k8s:openapi-gen=true
type ExternalJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalJobSpec   `json:"spec,omitempty"`
	Status ExternalJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalJobList contains a list of ExternalJob
type ExternalJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExternalJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ExternalJob{}, &ExternalJobList{})
}

func (ej *ExternalJob) convertToV2ExternalJob(jobV2 *v2.ExternalJob) error {
	//TODO(droot): figure out how to make it easy
	jobV2.ObjectMeta = ej.ObjectMeta
	return nil
}

func (ej *ExternalJob) convertFromV2ExternalJob(jobv2 *v2.ExternalJob) error {
	ej.ObjectMeta = jobv2.ObjectMeta
	return nil
}

func (ej *ExternalJob) ConvertTo(dst runtime.Object) error {
	switch t := dst.(type) {
	case *v2.ExternalJob:
		return ej.convertToV2ExternalJob(dst.(*v2.ExternalJob))
	case *ExternalJob:
		ej.DeepCopyInto(dst.(*ExternalJob))
		return nil
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
}

func (ej *ExternalJob) ConvertFrom(src runtime.Object) error {
	switch t := src.(type) {
	case *v2.ExternalJob:
		return ej.convertFromV2ExternalJob(src.(*v2.ExternalJob))
	case *ExternalJob:
		src.(*ExternalJob).DeepCopyInto(ej)
		return nil
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
}

// Make it a Hub ?
// func (ej *ExternalJob) Hub() {}
