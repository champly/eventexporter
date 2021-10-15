package exporter

import (
	"fmt"
	"io/ioutil"

	"github.com/champly/eventexporter/pkg/kube"
	"github.com/champly/eventexporter/pkg/sinks"
	"gopkg.in/yaml.v2"
	"k8s.io/klog/v2"
)

var (
	ConfigPath = "/etc/eventexporter/exporter.yaml"
)

type Config struct {
	Route          Route                  `yaml:"route"`
	ReceiverConfig []sinks.ReceiverConfig `yaml:"receiverConfigs"`
}

type Engine struct {
	Route Route
}

func NewEngine() (*Engine, error) {
	b, err := ioutil.ReadFile(ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("Read exporter config %s failed: %+v", ConfigPath, err)
	}
	cfg := &Config{}
	err = yaml.Unmarshal(b, cfg)
	if err != nil {
		return nil, fmt.Errorf("YAML unmarshal %s failed: %+v", string(b), err)
	}
	for _, rcfg := range cfg.ReceiverConfig {
		err = sinks.InitReceiver(rcfg)
		if err != nil {
			return nil, err
		}
	}

	return &Engine{Route: cfg.Route}, nil
}

// OnEvent does not care whether event is add or update. Prior filtering should be done int the controller/watcher
func (e *Engine) OnEvent(ev *kube.EnhancedEvent) {
	e.Route.ProcessEvent(ev)
}

func (e *Engine) Stop() {
	klog.Info("Closing sinks")
	sinks.Close()
	klog.Info("All sinks closed")
}
