/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"troubleshooter/pkg/pod"
)

// scheduleCmd represents the schedule command
var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Troubleshoot problems related to pod schedule",
	Long: `Troubleshoot problems related to pod schedule.

Examples:
# Troubleshoot pod schedule
troubleshoot pod schedule -p xxxx -n yyyy

# Troubleshoot pod schedule with specified kubeconfig
troubleshoot pod --kube-config /path/to/kubeconfig schedule -p xxxx -n yyyy`,
	Run: run,
}

var (
	podName      string
	podNamespace string
	nodeName     string
)

func init() {
	podCmd.AddCommand(scheduleCmd)
	scheduleCmd.Flags().StringVarP(&podName, "pod", "p", "", "pod name in k8s")
	scheduleCmd.Flags().StringVarP(&nodeName, "node", "n", "", "node name in k8s")
	scheduleCmd.Flags().StringVar(&podNamespace, "namespace", "", "namespace of pod in k8s")

	scheduleCmd.MarkFlagRequired("pod")
	scheduleCmd.MarkFlagRequired("node")
}

func run(cmd *cobra.Command, args []string) {
	defer func() {
		if r := recover(); r != nil {
			if err, ok := r.(error); ok {
				fmt.Println("[NoPass] " + err.Error())
			}
		}
	}()

	ts := pod.NewScheduleTroubleShooter(
		kubeConfigPath,
		podName,
		podNamespace,
		nodeName,
	)

	ts.Execute()
}
