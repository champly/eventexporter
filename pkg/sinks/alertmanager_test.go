package sinks

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/cli"
)

func TestSendToAlertManager(t *testing.T) {
	// r, err := client.NewHTTPClientWithConfig(nil, &client.TransportConfig{Host: "127.0.0.1:9093", BasePath: "/api/v2"}).Alert.PostAlerts(&alert.PostAlertsParams{
	// Alerts: []*models.PostableAlert{
	// {
	// Annotations: models.LabelSet(map[string]string{"k": "v"}),
	// },
	// },
	// HTTPClient: http.DefaultClient,
	// })
	// // models.PostableAlerts
	// if err != nil {
	// t.Error(err)
	// return
	// }

	// t.Log(r.Error())

	labels, _ := parseLabels(nil, map[string]string{"alertname": "123", "l1": "v1", "l2": "v2"})
	annotations, _ := parseLabels(nil, map[string]string{"a1": "v1", "a2": "v2"})

	pa := &models.PostableAlert{
		Alert: models.Alert{
			GeneratorURL: strfmt.URI("http://xxxx.com"),
			Labels:       labels,
		},
		Annotations: annotations,
		StartsAt:    strfmt.DateTime(time.Now()),
		EndsAt:      strfmt.DateTime(time.Now().Add(time.Minute * 5)),
	}
	alertParams := alert.NewPostAlertsParams().WithContext(context.TODO()).
		WithAlerts(models.PostableAlerts{pa})

	amclient := cli.NewAlertmanagerClient(&url.URL{Host: "127.0.0.1:9093"})
	r, err := amclient.Alert.PostAlerts(alertParams)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(r.Error())
}
