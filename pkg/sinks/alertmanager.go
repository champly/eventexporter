package sinks

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

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

var (
	defaultLayoutLabelMap = map[string]string{
		"alertname": "eventexporter",
		"type":      "{{ .Event.Type }}",
		"cluster":   "{{ .Event.ClusterName }}",
		"kind":      "{{ .Event.InvolvedObject.Kind }}",
		"reason":    "{{ .Event.Reason }}",
		"name":      "{{ .Event.Name }}",
		"namespace": "{{ .Event.Namespace }}",
		"count":     "{{ .Event.Count }}",
		"message":   "{{ .Event.Message }}",
		"component": "{{ .Event.Source.Component }}",
		"host":      "{{ .Event.Source.Host }}",
	}

	defaultLayoutAnnotationMap = map[string]string{
		"message": "{{ .Event.Message }}",
	}
)

func init() {
	factory[AlertmanagerSinkName] = NewAlertmanagerSink
}

type alertmanagerConfig struct {
	Host             string            `yaml:"host"`
	LabelLayout      map[string]string `yaml:"laybelLayout"`
	AnnotationLayout map[string]string `yaml:"annotationLayout"`
}

type alertmanager struct {
	*alertmanagerConfig
	amclient *client.AlertmanagerAPI
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
	pa, err := am.buildPostableAlertData(ev)
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

	klog.Infof("Send %s -> %s/%s event success.", ev.InvolvedObject.ClusterName, ev.Namespace, ev.Name)
	return nil
}

func (alert *alertmanager) Close() {
	// TODO: close connected.
}

func (am *alertmanager) buildPostableAlertData(ev *kube.EnhancedEvent) (*models.PostableAlert, error) {
	labels, err := parseLabels(ev, defaultLayoutLabelMap, am.LabelLayout)
	if err != nil {
		return nil, err
	}

	annotations, err := parseLabels(ev, defaultLayoutAnnotationMap, am.AnnotationLayout)
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
		EndsAt:      strfmt.DateTime(ev.Event.LastTimestamp.Add(time.Minute * 5)),
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
		// https://github.com/helm/charts/blob/f4f301ae450101b981805bd045451f08c0d74afa/stable/prometheus-operator/templates/alertmanager/service.yaml?_pjax=%23js-repo-pjax-container%2C%20div%5Bitemtype%3D%22http%3A%2F%2Fschema.org%2FSoftwareSourceCode%22%5D%20main%2C%20%5Bdata-pjax-container%5D#L44
		if len(svc.Spec.Selector) > 0 && svc.Spec.Selector["app"] == "alertmanager" {
			return fmt.Sprintf("%s.%s.svc:9093", svc.Name, svc.Namespace), nil
		}
	}

	return "", errors.New("not found alertmanager in manager plane cluster with svc spec.selector.app == alertmanager")
}

func buildInputLabelsField(name, value string) string {
	return name + "=" + value
}

// parseLabels parses a list of labels (cli arguments).
func parseLabels(ev *kube.EnhancedEvent, inputLabels ...map[string]string) (models.LabelSet, error) {
	labelSet := make(models.LabelSet)

	for _, inputLabel := range inputLabels {
		for key, value := range inputLabel {
			m, _ := getLayoutString(ev, value)
			if len(m) > 0 {
				labelSet[key] = m
			}
		}
	}

	return labelSet, nil
}

/*
 * {
 *     "annotations": {
 *         "message": "HPA default/mid-cloud-test-provider-gz01b-green has been running at max replicas for longer than 15 minutes.",
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
 *     "generatorURL": "http://prometheus.default.com/graph?g0.expr=kube_hpa_status_current_replicas%7Bjob%3D%22kube-state-metrics%22%2Cnamespace%3D~%22.%2A%22%7D+%3D%3D+kube_hpa_spec_max_replicas%7Bjob%3D%22kube-state-metrics%22%2Cnamespace%3D~%22.%2A%22%7D&g0.tab=1",
 *     "labels": {
 *         "alertname": "KubeHpaMaxedOut",
 *         "cluster": "dev-tke-cd-mid",
 *         "endpoint": "http",
 *         "env": "prod",
 *         "hpa": "mid-cloud-test-provider-gz01b-green",
 *         "instance": "10.48.224.79:8080",
 *         "job": "kube-state-metrics",
 *         "namespace": "default",
 *         "pod": "monitor-dev-tke-cd-mid-kube-state-metrics-7f4586c4d9-wzssg",
 *         "service": "monitor-dev-tke-cd-mid-kube-state-metrics",
 *         "severity": "warning"
 *     }
 * }
 */
