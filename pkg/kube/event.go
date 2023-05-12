package kube

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type EnhancedEvent struct {
	corev1.Event   `json:",inline"`
	InvolvedObject EnhancedObjectReference `json:"involvedObject"`
}

type EnhancedObjectReference struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ClusterName string            `json:"clusterName,omitempty"`
}

// ToJSON does not return an error because we are %99 confident it is JSON serializable.
// TODO(makin) Is it a bad practice? It's open to discussion.
func (e *EnhancedEvent) ToJSON() []byte {
	b, _ := json.Marshal(e)
	return b
}

func (e *EnhancedEvent) GetTimestampMs() int64 {
	return e.FirstTimestamp.UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}
