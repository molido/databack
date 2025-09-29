/*
Copyright 2025.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type Origin struct {
	//数据库访问地址
	Host string `json:"host"`
	//数据库端口
	Port     int32  `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// 目标地址
type Destination struct {
	Endpoint     string `json:"endpoint"`
	AccessKey    string `json:"accessKey"`
	AccessSecret string `json:"accessSecret"`
	BuketName    string `json:"buketName"`
}

// DatabackSpec defines the desired state of Databack
type DatabackSpec struct {
	//是否开启备份任务
	Enable bool `json:"enable"`
	//数据备份开始备份时间 12:00
	StartTime string `json:"startTime"`
	//数据源
	Origin Origin `json:"origin"`
	//备份目标地址
	Destination Destination `json:"destination"`
	//间隔周期（分钟）
	Period int `json:"period"`
}

// DatabackStatus defines the observed state of Databack.
type DatabackStatus struct {
	Active           bool   `json:"active"`
	NextTime         int64  `json:"nextTime"`
	LastBackupResult string `json:"lastBackupResult"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Databack is the Schema for the databacks API
type Databack struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Databack
	// +required
	Spec DatabackSpec `json:"spec"`

	// status defines the observed state of Databack
	// +optional
	Status DatabackStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// DatabackList contains a list of Databack
type DatabackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Databack `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Databack{}, &DatabackList{})
}
