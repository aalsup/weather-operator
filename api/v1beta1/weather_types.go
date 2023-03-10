/*
Copyright 2022.

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

type SecretRefSpec struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

// WeatherSpec defines the desired state of Weather
type WeatherSpec struct {
	Lon           string        `json:"lon"`
	Lat           string        `json:"lat"`
	SecretRef     SecretRefSpec `json:"secretRef"`
	RefreshPeriod string        `json:"refreshPeriod"`
}

// WeatherStatus defines the observed state of Weather
type WeatherStatus struct {
	RefreshTime  string `json:"refresh_time"`
	CountryCode  string `json:"country_code"`
	LocationName string `json:"location_name"`
	Temp         string `json:"temp"`
	Pressure     int64  `json:"pressure"`
	Humidity     int64  `json:"humidity"`
	WindSpeed    string `json:"wind_speed"`
	WindGust     string `json:"wind_gust"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Lat",type="string",JSONPath=".spec.lat",description="Latitude"
//+kubebuilder:printcolumn:name="Lon",type="string",JSONPath=".spec.lon",description="Longitude"
//+kubebuilder:printcolumn:name="Location",type="string",JSONPath=".status.location_name",description="Location"
//+kubebuilder:printcolumn:name="Temp",type="string",JSONPath=".status.temp",description="Temp"
//+kubebuilder:printcolumn:name="Refreshed",type="string",JSONPath=".status.refresh_time",description="Refreshed"

// Weather is the Schema for the weathers API
type Weather struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WeatherSpec   `json:"spec,omitempty"`
	Status WeatherStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WeatherList contains a list of Weather
type WeatherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Weather `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Weather{}, &WeatherList{})
}
