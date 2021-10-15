package sinks

import (
	"testing"
	"time"

	"github.com/champly/eventexporter/pkg/kube"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLayoutConvert(t *testing.T) {
	ev := &kube.EnhancedEvent{}
	ev.Namespace = "default"
	ev.Type = "Warning"
	ev.Kind = "Pod"
	ev.Name = "nginx-server-123abc-456def"
	ev.Message = "Successfully pulled image \"nginx:latest\""
	ev.FirstTimestamp = metav1.Time{Time: time.Now()}

	// Because Go, when parsing yaml, its []interface, not []string
	var tagz interface{}
	tagz = make([]interface{}, 2)
	tagz.([]interface{})[0] = "sre"
	tagz.([]interface{})[1] = "ops"

	layout := map[string]interface{}{
		"details": map[interface{}]interface{}{
			"message":   "{{ .Message }}",
			"kind":      "{{ .Kind }}",
			"name":      "{{ .Name }}",
			"namespace": "{{ .Namespace }}",
			"type":      "{{ .Type }}",
			"tags":      tagz,
		},
		"eventType": "kube-event",
		"region":    "us-west-2",
		"createdAt": "{{ .GetTimestampMs }}", // TODO: Test Int casts
	}

	res, err := convertLayoutTemplate(layout, ev)
	require.NoError(t, err)
	require.Equal(t, res["eventType"], "kube-event")

	val, ok := res["details"].(map[string]interface{})
	require.True(t, ok, "cannot cast to event")

	val2, ok2 := val["message"].(string)
	require.True(t, ok2, "cannot cast message to string")

	require.Equal(t, val2, ev.Message)
}
