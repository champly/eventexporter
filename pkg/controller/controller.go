package controller

import (
	"context"
	"sync"
	"time"

	"github.com/champly/eventexporter/pkg/exporter"
	"github.com/champly/eventexporter/pkg/kube"
	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/client"
	"github.com/symcn/pkg/clustermanager/configuration"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/trace"
)

var (
	ClusterCfgManagerCMNamespace = "default"
	ClusterCfgManagerCMLabels    = []string{"clusterowner=eventexporter"}
	ClusterCfgManagerCMDataKey   = "kubeconfig.yaml"
	ClusterCfgManagerCMStatusKey = "status"
)

type Controller struct {
	ctx             context.Context
	metadataHandler map[string]*kube.MetadataHandler
	engine          *exporter.Engine

	api.MultiMingleClient
	sync.Mutex
}

func New(ctx context.Context, mcc *client.MultiClientConfig) (*Controller, error) {
	mcc.ClusterCfgManager = configuration.NewClusterCfgManagerWithCM(
		kube.ManagerPlaneClusterClient.GetKubeInterface(),
		ClusterCfgManagerCMNamespace,
		transformLabelsArrayToMap(ClusterCfgManagerCMLabels),
		ClusterCfgManagerCMDataKey,
		ClusterCfgManagerCMStatusKey,
	)
	cc, err := client.Complete(mcc)
	if err != nil {
		return nil, err
	}
	mc, err := cc.New()
	if err != nil {
		return nil, err
	}

	// build exporter engine
	engine, err := exporter.NewEngine()
	if err != nil {
		return nil, err
	}

	ctrl := &Controller{
		ctx:               ctx,
		metadataHandler:   map[string]*kube.MetadataHandler{},
		engine:            engine,
		MultiMingleClient: mc,
	}

	ctrl.registryBeforAfterHandler()

	return ctrl, nil
}

func (ctrl *Controller) Start() error {
	return ctrl.MultiMingleClient.Start(ctrl.ctx)
}

func (ctrl *Controller) Reconcile(ctx context.Context, req api.WrapNamespacedName) (requeue api.NeedRequeue, after time.Duration, err error) {
	tr := trace.New("event-exporter-controller",
		trace.Field{Key: "cluster", Value: req.QName},
		trace.Field{Key: "namespace", Value: req.Namespace},
		trace.Field{Key: "name", Value: req.Name},
	)
	defer tr.LogIfLong(time.Millisecond * 100)
	// tr.Log()

	cli, err := ctrl.GetWithName(req.QName)
	if err != nil {
		return api.Requeue, time.Second * 5, err
	}
	tr.Step("GetClientWithName")

	e := &corev1.Event{}
	err = cli.Get(req.NamespacedName, e)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// maybe event deleted.
			klog.Warningf("Cluster [%s] event %s not found", req.QName, req.NamespacedName.String())
			return api.Done, 0, nil
		}
		return api.Requeue, time.Second * 5, err
	}
	tr.Step("GetEventWithInformer")
	if time.Since(e.LastTimestamp.Time) > time.Second*5 {
		klog.Infof("Event %s/%s last time is %s skip.", e.Namespace, e.Name, e.LastTimestamp.Time.Format("2006-01-02 15:04:05"))
		return
	}

	// build enhanced event
	ev := &kube.EnhancedEvent{Event: *e.DeepCopy()}
	ev.InvolvedObject.ClusterName = req.QName
	tr.Step("DeepCopy")

	ev.InvolvedObject.Labels, ev.InvolvedObject.Annotations = ctrl.getLabelsAndAnnotations(req, e)
	tr.Step("GetLabelsAndAnnotations")

	klog.V(4).Infof("Send enhanced event %s/%s to engine.", ev.Namespace, ev.Name)
	ctrl.engine.OnEvent(ev)
	tr.Step("Send Event")

	return api.Done, 0, nil
}

func (ctrl *Controller) getLabelsAndAnnotations(req api.WrapNamespacedName, evt *corev1.Event) (map[string]string, map[string]string) {
	handler, ok := ctrl.metadataHandler[req.QName]
	if !ok {
		klog.Warningf("Cluster [%s] metadata handler not found", req.QName)
		return nil, nil
	}

	labels, annotations, err := handler.GetlabelsAndAnnotations(&evt.InvolvedObject)
	if err != nil {
		// ignoring error, but log it anyways
		klog.Errorf("Cannot list cluster [%s] labels of the objects: %s", req.QName, err)
		return nil, nil
	}
	return labels, annotations
}
