package main

import (
	"math/rand"
	"time"

	"github.com/champly/eventexporter/cmd/eventexporter"
	"k8s.io/klog/v2"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	cmd := eventexporter.NewRootCmd()
	if err := cmd.Execute(); err != nil {
		klog.Errorf("Execute event exporter failed.")
	}
}
