package controller

import (
	"context"
	"sync"
	"time"

	"github.com/champly/eventexporter/pkg/exporter"
	"github.com/champly/eventexporter/pkg/kube"
	"github.com/symcn/api"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/klog/v2"
	"k8s.io/utils/trace"
)

type Controller struct {
	ctx             context.Context
	metadataHandler map[string]*kube.MetadataHandler
	engine          *exporter.Engine

	api.MultiMingleClient
	sync.Mutex
}

func New(ctx context.Context, mc api.MultiMingleClient) (*Controller, error) {
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

func (ctrl *Controller) Reconcile(req api.WrapNamespacedName) (requeue api.NeedRequeue, after time.Duration, err error) {
	tr := trace.New("event-exporter-controller",
		trace.Field{Key: "cluster", Value: req.QName},
		trace.Field{Key: "namespace", Value: req.Namespace},
		trace.Field{Key: "name", Value: req.Name},
	)
	defer tr.LogIfLong(time.Millisecond * 100)

	tr.Step("GetClientWithName")
	cli, err := ctrl.GetWithName(req.QName)
	if err != nil {
		return api.Requeue, time.Second * 5, err
	}

	tr.Step("GetEventWithInformer")
	e := &corev1.Event{}
	err = cli.Get(req.NamespacedName, e)
	if err != nil {
		if apierrors.IsNotFound(err) {
			klog.Warningf("Cluster [%s] event %s not found", req.QName, req.NamespacedName.String())
			return api.Done, 0, nil
		}
		return api.Requeue, time.Second * 5, err
	}
	if time.Now().Sub(e.LastTimestamp.Time) > time.Second*5 {
		klog.V(3).Infof("Event %s/%s last time is %s skip.", e.Namespace, e.Name, e.LastTimestamp.Time.Format("2006-01-02 15:04:05"))
		return
	}

	// build enhanced event
	tr.Step("DeepCopy")
	ev := &kube.EnhancedEvent{Event: *e.DeepCopy()}
	ev.Event.ObjectMeta.ClusterName = req.QName

	tr.Step("GetLabels")
	ev.InvolvedObject.Labels = ctrl.getLabels(req, e)
	tr.Step("GetAnnotations")
	ev.InvolvedObject.Annotations = ctrl.getAnnotation(req, e)

	klog.V(4).Infof("Send enhanced event %s/%s to engine.", ev.Namespace, ev.Name)
	tr.Step("Send Event")
	ctrl.engine.OnEvent(ev)

	return api.Done, 0, nil
}

func (ctrl *Controller) getLabels(req api.WrapNamespacedName, evt *corev1.Event) map[string]string {
	handler, ok := ctrl.metadataHandler[req.QName]
	if !ok {
		klog.Warningf("Cluster [%s] labels cache not found", req.QName)
		return nil
	}

	labels, err := handler.GetLabels(&evt.InvolvedObject)
	if err != nil {
		// ignoring error, but log it anyways
		klog.Errorf("Cannot list cluster [%s] labels of the objects: %s", req.QName, err)
		return nil
	}
	return labels
}

func (ctrl *Controller) getAnnotation(req api.WrapNamespacedName, evt *corev1.Event) map[string]string {
	handler, ok := ctrl.metadataHandler[req.QName]
	if !ok {
		klog.Warningf("Cluster [%s] annotations cache not found", req.QName)
		return nil
	}

	annotations, err := handler.GetAnnotations(&evt.InvolvedObject)
	if err != nil {
		// ignoring error, but log it anyways
		klog.Errorf("Cannot list cluster [%s] annotations of the objects: %s", req.QName, err)
		return nil
	}
	return annotations
}
