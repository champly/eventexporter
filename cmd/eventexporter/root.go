package eventexporter

import (
	"flag"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Long: "Multi kubernetes cluster event exporter",
	}

	klog.InitFlags(nil)

	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	rootCmd.AddCommand(NewEventExporter())

	return rootCmd
}

func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		klog.Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}
