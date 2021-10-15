package sinks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/champly/eventexporter/pkg/kube"
	"github.com/go-openapi/strfmt"
	"github.com/prometheus/alertmanager/api/v2/client"
	"github.com/prometheus/alertmanager/api/v2/client/alert"
	"github.com/prometheus/alertmanager/api/v2/models"
	"github.com/prometheus/alertmanager/cli"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

const AlertmanagerSinkName = "alertmanager"

func init() {
	factory[AlertmanagerSinkName] = NewAlertmanagerSink
}

type alertmanagerConfig struct {
	Host    string            `yaml:"host"`
	Headers map[string]string `yaml:"headers"`
}

type alertmanager struct {
	*alertmanagerConfig
	amclient *client.Alertmanager
}

func NewAlertmanagerSink(cfg interface{}) (Sink, error) {
	alertCfg, err := parse(cfg)
	if err != nil {
		return nil, err
	}
	amclient := cli.NewAlertmanagerClient(&url.URL{Host: alertCfg.Host})

	return &alertmanager{
		alertmanagerConfig: alertCfg,
		amclient:           amclient,
	}, nil
}

func (am *alertmanager) Send(ctx context.Context, ev *kube.EnhancedEvent) error {
	pa, err := buildPostableAlertData(ev)
	if err != nil {
		return err
	}
	alertParams := alert.NewPostAlertsParams().WithContext(context.TODO()).
		WithAlerts(models.PostableAlerts{pa})
	r, err := am.amclient.Alert.PostAlerts(alertParams)
	if err != nil {
		return err
	}
	if !strings.Contains(r.Error(), "postAlertsOK") {
		return errors.New(r.Error())
	}

	klog.Infof("Send %s -> %s/%s event success.", ev.Event.ObjectMeta.ClusterName, ev.Namespace, ev.Name)
	return nil
}

func (alert *alertmanager) Close() {
	// TODO: close connected.
}

func buildPostableAlertData(ev *kube.EnhancedEvent) (*models.PostableAlert, error) {
	labelsSlice := []string{
		buildInputLabelsField("alertname", "eventexporter"),
		buildInputLabelsField("type", ev.Event.Type),
		buildInputLabelsField("cluster", ev.Event.ClusterName),
		buildInputLabelsField("kind", ev.Event.Kind),
		buildInputLabelsField("reason", ev.Event.Reason),
		buildInputLabelsField("name", ev.Event.Name),
		buildInputLabelsField("namespace", ev.Namespace),
		buildInputLabelsField("count", strconv.Itoa(int(ev.Event.Count))),
		buildInputLabelsField("message", ev.Event.Message),
		buildInputLabelsField("component", ev.Event.Source.Component),
		buildInputLabelsField("host", ev.Event.Source.Host),
		buildInputLabelsField("ape", ev.Labels["app"]),
		buildInputLabelsField("group", ev.Labels["sym-group"]),
	}
	labels, err := parseLabels(labelsSlice)
	if err != nil {
		return nil, err
	}

	annotationsSlice := []string{
		// buildInputLabelsField("message", ev.Event.Message),
	}
	annotations, err := parseLabels(annotationsSlice)
	if err != nil {
		return nil, err
	}

	pa := &models.PostableAlert{
		Alert: models.Alert{
			// GeneratorURL: strfmt.URI("http://xxxx.com"),
			Labels: labels,
		},
		Annotations: annotations,
		StartsAt:    strfmt.DateTime(ev.Event.CreationTimestamp.Time),
		EndsAt:      strfmt.DateTime(ev.Event.LastTimestamp.Time),
	}

	return pa, nil
}

func parse(cfg interface{}) (*alertmanagerConfig, error) {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("init receiver, marshal interface{} to yaml %s config {%+v} failed: %v", AlertmanagerSinkName, cfg, err)
	}

	alertCfg := &alertmanagerConfig{}
	err = yaml.Unmarshal(b, alertCfg)
	if err != nil {
		return nil, fmt.Errorf("init receiver, unmarshal yaml to alertmanagerConfig %s failed config -> %s", AlertmanagerSinkName, cfg)
	}

	if alertCfg.Host == "" {
		host, err := getAlertManagerHost()
		if err != nil {
			return nil, err
		}
		klog.Infof("Not config endpoint, use manager plane cluster \"%s\" as endpoint", host)
		alertCfg.Host = host
	}
	klog.Infof("Alertmanager url: %s", alertCfg.Host)

	return alertCfg, nil
}

func getAlertManagerHost() (host string, err error) {
	svcList := &corev1.ServiceList{}
	err = kube.ManagerPlaneClusterClient.List(
		svcList,
		// &rtclient.ListOptions{
		// FieldSelector: fields.SelectorFromSet(fields.Set{
		// "spec.selector.app": "alertmanager",
		// }),
		// },
	)
	if err != nil {
		return "", fmt.Errorf("get manager plane cluster svc failed: %s", err.Error())
	}

	for _, svc := range svcList.Items {
		if len(svc.Spec.Selector) > 0 && svc.Spec.Selector["app"] == "alertmanager" {
			return fmt.Sprintf("%s.%s.svc", svc.Name, svc.Namespace), nil
		}
	}

	return "", fmt.Errorf("not found alertmanager in manager plane cluster with svc spec.selector.app == alertmanager.")
}

func buildInputLabelsField(name, value string) string {
	return name + "=" + value
}

// parseLabels parses a list of labels (cli arguments).
func parseLabels(inputLabels []string) (models.LabelSet, error) {
	labelSet := make(models.LabelSet, len(inputLabels))

	for _, l := range inputLabels {
		tmp := strings.Split(l, "=")
		if len(tmp) != 2 {
			klog.Warningf("inputLabels %s is not key=value and both not empty, skip it.", l)
			continue
		}
		labelSet[tmp[0]] = tmp[1]
	}

	return labelSet, nil
}

/*
 * {
 *     "annotations": {
 *         "message": "HPA dmall-inner/mid-cloud-test-provider-gz01b-green has been running at max replicas for longer than 15 minutes.",
 *         "runbook_url": "https://github.com/kubernetes-monitoring/kubernetes-mixin/tree/master/runbook.md#alert-name-kubehpamaxedout"
 *     },
 *     "endsAt": "2021-10-15T08:21:10.214Z",
 *     "fingerprint": "48c00a4033bdca56",
 *     "receivers": [
 *         {
 *             "name": "null"
 *         }
 *     ],
 *     "startsAt": "2021-10-08T07:10:40.214Z",
 *     "status": {
 *         "inhibitedBy": [],
 *         "silencedBy": [],
 *         "state": "active"
 *     },
 *     "updatedAt": "2021-10-15T08:18:10.260Z",
 *     "generatorURL": "http://mid.prometheus.dmall.com/graph?g0.expr=kube_hpa_status_current_replicas%7Bjob%3D%22kube-state-metrics%22%2Cnamespace%3D~%22.%2A%22%7D+%3D%3D+kube_hpa_spec_max_replicas%7Bjob%3D%22kube-state-metrics%22%2Cnamespace%3D~%22.%2A%22%7D&g0.tab=1",
 *     "labels": {
 *         "alertname": "KubeHpaMaxedOut",
 *         "cluster": "dev-tke-cd-mid",
 *         "endpoint": "http",
 *         "env": "prod",
 *         "hpa": "mid-cloud-test-provider-gz01b-green",
 *         "instance": "10.49.18.241:8080",
 *         "job": "kube-state-metrics",
 *         "namespace": "dmall-inner",
 *         "pod": "monitor-dev-tke-cd-mid-kube-state-metrics-7f4586c4d9-wzssg",
 *         "service": "monitor-dev-tke-cd-mid-kube-state-metrics",
 *         "severity": "warning"
 *     }
 * }
 */
