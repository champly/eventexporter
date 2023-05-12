package sinks

import (
	"context"
	"fmt"

	"github.com/champly/eventexporter/pkg/kube"
	"k8s.io/klog/v2"
)

type ReceiverConfig struct {
	Name   string
	Config interface{}
}

var (
	factory        = map[string]NewSinkFunc{}
	initedReceiver = map[string]Sink{}
)

func InitReceiver(cfg ReceiverConfig) error {
	f, ok := factory[cfg.Name]
	if !ok {
		return fmt.Errorf("not found %s receiver init function", cfg.Name)
	}
	if _, ok := initedReceiver[cfg.Name]; ok {
		return fmt.Errorf("receiver %s repeat initialization", cfg.Name)
	}

	sink, err := f(cfg.Config)
	if err != nil {
		return err
	}
	initedReceiver[cfg.Name] = sink
	return nil
}

func SendEvent(name string, ev *kube.EnhancedEvent) {
	sink, ok := initedReceiver[name]
	if !ok {
		klog.Errorf("Not config %s receiver", name)
		return
	}
	if err := sink.Send(context.TODO(), ev); err != nil {
		klog.Errorf("Receiver %s cannot send event: %+v", name, err)
	}
}

func Close() {
	for _, sink := range initedReceiver {
		sink.Close()
	}
}
