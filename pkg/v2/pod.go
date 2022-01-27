package v2

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	frameworkplugins "k8s.io/kubernetes/pkg/scheduler/framework/plugins"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

//
func buildFramework() {
	p := &config.KubeSchedulerProfile{
		SchedulerName: v1.DefaultSchedulerName,
	}
	fw, err := frameworkruntime.NewFramework(
		frameworkplugins.NewInTreeRegistry(),
		p,
	)

	if err != nil {
		panic(err)
	}

	fw.RunFilterPlugins()
}
