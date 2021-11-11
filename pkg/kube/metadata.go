package kube

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/symcn/api"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/restmapper"
	"k8s.io/klog/v2"
)

type MetadataHandler struct {
	ctx               context.Context
	cli               api.MingleClient
	informers         dynamicinformer.DynamicSharedInformerFactory
	rm                meta.RESTMapper
	sharedInformerMap map[schema.GroupVersionResource]informers.GenericInformer
	sync.RWMutex
}

func NewMetadataHandler(ctx context.Context, cli api.MingleClient) *MetadataHandler {
	a := &MetadataHandler{
		ctx:               ctx,
		cli:               cli,
		sharedInformerMap: map[schema.GroupVersionResource]informers.GenericInformer{},
	}

	// build dynamicinformer factory
	informersFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamic.NewForConfigOrDie(cli.GetKubeRestConfig()), time.Hour*2)
	a.informers = informersFactory
	go a.informers.Start(a.ctx.Done())

	// build resource mapping
	groupResources, err := restmapper.GetAPIGroupResources(cli.GetKubeInterface().Discovery())
	if err != nil {
		panic(err)
	}
	a.rm = restmapper.NewDiscoveryRESTMapper(groupResources)
	return a
}

func (m *MetadataHandler) GetAnnotations(reference *corev1.ObjectReference) (map[string]string, error) {
	obj, err := m.getUnstructedWithObjectReference(reference)
	if err != nil {
		return nil, err
	}

	// filter annotations
	annotations := obj.GetAnnotations()
	for key := range annotations {
		if strings.Contains(key, "kubernetes.io/") || strings.Contains(key, "k8s.io/") {
			delete(annotations, key)
		}
	}
	return annotations, nil
}

func (m *MetadataHandler) GetLabels(reference *corev1.ObjectReference) (map[string]string, error) {
	obj, err := m.getUnstructedWithObjectReference(reference)
	if err != nil {
		return nil, err
	}

	return obj.GetLabels(), nil
}

func (m *MetadataHandler) GetlabelsAndAnnotations(reference *corev1.ObjectReference) (labels map[string]string, annotations map[string]string, err error) {
	obj, err := m.getUnstructedWithObjectReference(reference)
	if err != nil {
		return nil, nil, err
	}

	labels = obj.GetLabels()
	// filter annotations
	annotations = obj.GetAnnotations()
	for key := range annotations {
		if strings.Contains(key, "kubernetes.io/") || strings.Contains(key, "k8s.io/") {
			delete(annotations, key)
		}
	}
	return labels, annotations, nil
}

func (m *MetadataHandler) getUnstructedWithObjectReference(reference *corev1.ObjectReference) (*unstructured.Unstructured, error) {
	// build generic informer
	informer, err := m.getGenericInfomer(reference)
	if err != nil {
		return nil, err
	}

	// get resource from informer
	o, err := informer.Lister().ByNamespace(reference.Namespace).Get(reference.Name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, err
		}
		// maybe resource not define with namespace, should re-get without namespace.
		// such as: node
		o, err = informer.Lister().Get(reference.Name)
		if err != nil {
			return nil, err
		}
	}
	return TransformRuntimeObjToUnstructured(o)
}

func (m *MetadataHandler) getGenericInfomer(reference *corev1.ObjectReference) (informers.GenericInformer, error) {
	gk, v := GetGKindVersion(reference)
	mapping, err := m.rm.RESTMapping(gk, v)
	if err != nil {
		return nil, err
	}

	var (
		informer informers.GenericInformer
		ok       bool
	)

	m.Lock()
	defer m.Unlock()
	informer, ok = m.sharedInformerMap[mapping.Resource]
	if ok {
		return informer, nil
	}

	klog.Infof("Build cluster [%s] new dynamic informer for -> %s", m.cli.GetClusterCfgInfo().GetName(), mapping.Resource.String())
	informer = m.informers.ForResource(mapping.Resource)

	go informer.Informer().Run(m.ctx.Done())

	for !informer.Informer().HasSynced() {
		klog.V(5).Infof("Wait %s informer cache sync.", mapping.Resource.String())
		time.Sleep(time.Millisecond * 100)
	}
	m.sharedInformerMap[mapping.Resource] = informer
	return informer, nil
}
