/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
)

// podCmd represents the pod command
var podCmd = &cobra.Command{
	Use:   "pod",
	Short: "Troubleshoot problems related to pod",
	Long: `Troubleshoot problems related to pod.

Examples:
# Troubleshoot pod schedule
troubleshoot pod schedule -p xxxx -n yyyy

# Troubleshoot pod schedule with specified kubeconfig
troubleshoot pod --kube-config /path/to/kubeconfig schedule -p xxxx -n yyyy`,
}

var kubeConfigPath string

func init() {
	rootCmd.AddCommand(podCmd)
	podCmd.PersistentFlags().StringVar(&kubeConfigPath, "kube-config", defaultKubeConfigPath(), "kubeconfig to access k8s")
}

func defaultKubeConfigPath() string {
	return filepath.Join(homedir.HomeDir(), ".kube", "config")
}
