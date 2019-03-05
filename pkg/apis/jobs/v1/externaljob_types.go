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
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ExternalJobSpec defines the desired state of ExternalJob
type ExternalJobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	RunAt string `json:"runAt"`
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

// since v1.ExternalJob is a spoke, it needs to be convertable.
// It needs to implement convert to/from storage version
// TODO(droot): evaluate advantages of taking `conversion.Hub` as input to
// converter methods. One downside of that approach is if we don't want to
// use Hub interface to indicate storage/hub type, then taking Hub as input
// is not a good idea. So keeping it open.

func (ej *ExternalJob) ConvertTo(dst conversion.Hub) error {
	switch t := dst.(type) {
	case *v2.ExternalJob:
		jobv2 := dst.(*v2.ExternalJob)
		jobv2.ObjectMeta = ej.ObjectMeta
		jobv2.Spec.ScheduleAt = ej.Spec.RunAt
		return nil
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
}

func (ej *ExternalJob) ConvertFrom(src conversion.Hub) error {
	switch t := src.(type) {
	case *v2.ExternalJob:
		jobv2 := src.(*v2.ExternalJob)
		ej.ObjectMeta = jobv2.ObjectMeta
		ej.Spec.RunAt = jobv2.Spec.ScheduleAt
		return nil
	default:
		return fmt.Errorf("unsupported type %v", t)
	}
}
