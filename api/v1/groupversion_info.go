// Package v1 contains API Schema definitions for the mcall v1 API group
// +kubebuilder:object:generate=true
// +groupName=mcall.tz.io
package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion is group version used to register these objects
	GroupVersion = schema.GroupVersion{Group: "mcall.tz.io", Version: "v1"}

	// SchemeBuilder initializes a scheme builder
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme is a global function that registers this API group & version to a scheme
	AddToScheme = SchemeBuilder.AddToScheme
)
