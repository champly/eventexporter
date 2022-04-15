package kube

import (
	"context"
	"fmt"

	"github.com/symcn/api"
	"github.com/symcn/pkg/clustermanager/client"
	"github.com/symcn/pkg/clustermanager/configuration"
)

var (
	ManagerPlaneName          = "eventexporter-manager-plane"
	ManagerPlaneClusterClient api.MingleClient
)

// InitManagerPlaneClusterClient build manager-plane cluster client
// default use current env kubeconfig
// TODO: support kubeconfig configuration
func InitManagerPlaneClusterClient(ctx context.Context) (err error) {
	ManagerPlaneClusterClient, err = client.NewMingleClient(
		configuration.BuildDefaultClusterCfgInfo(ManagerPlaneName),
		client.DefaultOptions(),
	)
	if err != nil {
		return fmt.Errorf("init manager-plane cluster client failed: %s", err.Error())
	}

	go ManagerPlaneClusterClient.Start(ctx)

	return nil
}
