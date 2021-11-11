package kube

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TransformRuntimeObjToUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	unstructObj := &unstructured.Unstructured{}
	o, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("converts an object into map[string]interface{} representation failed: %+v", err)
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(o, unstructObj)
	if err != nil {
		return nil, fmt.Errorf("converts an object from map[string]interface{} representation into a concrete typefailed: %+v", err)
	}
	return unstructObj, nil
}

func GetGKindVersion(reference *corev1.ObjectReference) (schema.GroupKind, string) {
	var group, version string
	s := strings.Split(reference.APIVersion, "/")
	if len(s) == 1 {
		group = ""
		version = s[0]
	} else {
		group = s[0]
		version = s[1]
	}

	return schema.GroupKind{Group: group, Kind: reference.Kind}, version
}
