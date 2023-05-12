package eventexporter

import (
	"github.com/champly/eventexporter/pkg/controller"
	"github.com/champly/eventexporter/pkg/exporter"
	"github.com/champly/eventexporter/pkg/kube"
	"github.com/spf13/cobra"
	"github.com/symcn/pkg/clustermanager/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func NewEventExporter() *cobra.Command {
	mcc := client.NewMultiClientConfig()
	cmd := &cobra.Command{
		Use:          "event",
		Short:        "Event exporter",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			PrintFlags(cmd.Flags())

			ctx := signals.SetupSignalHandler()

			// !import build manager plane client first
			if err := kube.InitManagerPlaneClusterClient(ctx); err != nil {
				return err
			}

			// build controller
			ctrl, err := controller.New(ctx, mcc)
			if err != nil {
				return err
			}
			return ctrl.Start()
		},
	}

	// manager-plane
	cmd.PersistentFlags().StringVarP(&kube.ManagerPlaneName, "manager_plane_name", "", kube.ManagerPlaneName, "manager plane client-go user-agent name")

	// exporter
	cmd.PersistentFlags().StringVarP(&exporter.ConfigPath, "exporter_config_path", "", exporter.ConfigPath, "Exported config path which can define multi receiver and filter rule with yaml format.")

	// controller
	cmd.PersistentFlags().IntVarP(&controller.HttpPort, "http_port", "", controller.HttpPort, "Controller http port witch provide metrics, health, ready and debug.")

	// cluster configuration manager config
	cmd.PersistentFlags().StringVarP(&controller.ClusterCfgManagerCMNamespace, "ccm_namespace", "", controller.ClusterCfgManagerCMNamespace, "Multi cluster manager connect info, filter configmap with namespace.")
	cmd.PersistentFlags().StringArrayVar(&controller.ClusterCfgManagerCMLabels, "ccm_labels", controller.ClusterCfgManagerCMLabels, "Multi cluster manager connect info get form configmap with labels.")
	cmd.PersistentFlags().StringVarP(&controller.ClusterCfgManagerCMDataKey, "ccm_data_key", "", controller.ClusterCfgManagerCMDataKey, "Multi cluster manager connect info get form configmap with data_key.")
	cmd.PersistentFlags().StringVarP(&controller.ClusterCfgManagerCMStatusKey, "ccm_stats_key", "", controller.ClusterCfgManagerCMStatusKey, "Multi cluster manager connect info form configmap with status.")

	// multi client
	cmd.PersistentFlags().DurationVarP(&mcc.FetchInterval, "fetch_interval", "", mcc.FetchInterval, "Auto invoke multiclusterconfiguration find new cluster or delete old cluster time interval.")
	cmd.PersistentFlags().DurationVarP(&mcc.Options.ExecTimeout, "exec_timeout", "", mcc.Options.ExecTimeout, "Set mingle client exec timeout, if less than default timeout, use default.")
	cmd.PersistentFlags().DurationVarP(&mcc.Options.HealthCheckInterval, "health_check_interval", "", mcc.Options.HealthCheckInterval, "Set mingle clinet check kubernetes connected interval.")
	cmd.PersistentFlags().DurationVarP(&mcc.Options.SyncPeriod, "sync_period", "", mcc.Options.SyncPeriod, "Set informer sync period time interval.")
	cmd.PersistentFlags().StringVarP(&mcc.Options.UserAgent, "user_agent", "", mcc.Options.UserAgent, "client-go connected user-agent.")
	cmd.PersistentFlags().IntVarP(&mcc.Options.QPS, "qps", "", mcc.Options.QPS, "Set mingle client qps for each cluster")
	cmd.PersistentFlags().IntVarP(&mcc.Options.Burst, "burst", "", mcc.Options.Burst, "Set mingle client burst for each cluster")
	cmd.PersistentFlags().BoolVarP(&mcc.Options.LeaderElection, "leader_election", "", mcc.Options.LeaderElection, "Enabled leader election, if true, should set --leader_election_id both.")
	cmd.PersistentFlags().StringVarP(&mcc.Options.LeaderElectionID, "leader_election_id", "", mcc.Options.LeaderElectionID, "Set leader election id.")
	cmd.PersistentFlags().StringVarP(&mcc.Options.LeaderElectionNamespace, "leader_election_ns", "", mcc.Options.LeaderElectionNamespace, "Set leader election namespace.")

	return cmd
}
