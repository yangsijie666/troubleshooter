package pod

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/events"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/parallelize"
	frameworkruntime "k8s.io/kubernetes/pkg/scheduler/framework/runtime"
)

type frameworkOptions struct {
	clientSet            clientset.Interface
	kubeConfig           *rest.Config
	informerFactory      informers.SharedInformerFactory
	snapshotSharedLister framework.SharedLister
	extenders            []framework.Extender
	runAllFilters        bool
	parallelizer         parallelize.Parallelizer
}

type Option func(*frameworkOptions)

func NewFramework(
	r frameworkruntime.Registry,
	profile *config.KubeSchedulerProfile,
	opts ...Option,
) (framework.Framework, error) {
	options := defaultFrameworkOptions()
	for _, opt := range opts {
		opt(&options)
	}

	f := &TroubleShootPodScheduleFilterFramework{
		runAllFilters:         options.runAllFilters,
		clientSet:             options.clientSet,
		kubeConfig:            options.kubeConfig,
		sharedInformerFactory: options.informerFactory,
		snapshotSharedLister:  options.snapshotSharedLister,
		parallelizer:          options.parallelizer,
	}

	plugins, err := f.findNeededPlugins(r, profile)
	if err != nil {
		return nil, err
	}

	f.filterPlugins = make([]framework.FilterPlugin, 0)
	f.preFilterPlugins = make([]framework.PreFilterPlugin, 0)
	for _, pl := range plugins {
		if preFilterPl, ok := pl.(framework.PreFilterPlugin); ok {
			f.preFilterPlugins = append(f.preFilterPlugins, preFilterPl)
		}
		if filterPl, ok := pl.(framework.FilterPlugin); ok {
			f.filterPlugins = append(f.filterPlugins, filterPl)
		}
	}

	return f, nil
}

func (f *TroubleShootPodScheduleFilterFramework) findNeededPlugins(
	r frameworkruntime.Registry,
	profile *config.KubeSchedulerProfile,
) ([]framework.Plugin, error) {
	enabledPlugins := make(map[string]config.Plugin)
	for _, plugin := range profile.Plugins.MultiPoint.Enabled {
		enabledPlugins[plugin.Name] = plugin
	}

	pluginConfig := make(map[string]runtime.Object)
	for _, pc := range profile.PluginConfig {
		if _, ok := enabledPlugins[pc.Name]; ok {
			pluginConfig[pc.Name] = pc.Args
		}
	}

	plugins := make([]framework.Plugin, 0)

	for pluginName, factory := range r {
		args, _ := pluginConfig[pluginName]
		pl, err := factory(args, f)
		if err != nil {
			return nil, err
		}
		plugins = append(plugins, pl)
	}

	return plugins, nil
}

func WithRunAllFilters(runAllFilters bool) Option {
	return func(o *frameworkOptions) {
		o.runAllFilters = runAllFilters
	}
}

func WithClientSet(cs clientset.Interface) Option {
	return func(o *frameworkOptions) {
		o.clientSet = cs
	}
}

func WithKubeConfig(kubeConfig *rest.Config) Option {
	return func(o *frameworkOptions) {
		o.kubeConfig = kubeConfig
	}
}

func WithInformerFactory(informerFactory informers.SharedInformerFactory) Option {
	return func(o *frameworkOptions) {
		o.informerFactory = informerFactory
	}
}

func WithSnapshotSharedLister(sharedLister framework.SharedLister) Option {
	return func(o *frameworkOptions) {
		o.snapshotSharedLister = sharedLister
	}
}

func WithExtenders(extenders []framework.Extender) Option {
	return func(o *frameworkOptions) {
		o.extenders = extenders
	}
}

func WithParallelizer(parallelizer parallelize.Parallelizer) Option {
	return func(o *frameworkOptions) {
		o.parallelizer = parallelizer
	}
}

func defaultFrameworkOptions() frameworkOptions {
	return frameworkOptions{
		parallelizer: parallelize.NewParallelizer(parallelize.DefaultParallelism),
	}
}

type TroubleShootPodScheduleFilterFramework struct {
	framework.Handle
	preFilterPlugins []framework.PreFilterPlugin
	filterPlugins    []framework.FilterPlugin

	runAllFilters         bool
	clientSet             kubernetes.Interface
	kubeConfig            *rest.Config
	sharedInformerFactory informers.SharedInformerFactory
	snapshotSharedLister  framework.SharedLister

	parallelizer parallelize.Parallelizer
}

func (f *TroubleShootPodScheduleFilterFramework) QueueSortFunc() framework.LessFunc {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunPostFilterPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunPreBindPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunPostBindPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunReservePluginsReserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunReservePluginsUnreserve(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunPermitPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) WaitOnPermit(ctx context.Context, pod *v1.Pod) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunBindPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) HasFilterPlugins() bool {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) HasPostFilterPlugins() bool {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) HasScorePlugins() bool {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) ListPlugins() *config.Plugins {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) ProfileName() string {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) AddNominatedPod(pod *framework.PodInfo, nodeName string) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) DeleteNominatedPodIfExists(pod *v1.Pod) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) UpdateNominatedPod(oldPod *v1.Pod, newPodInfo *framework.PodInfo) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) NominatedPodsForNode(nodeName string) []*framework.PodInfo {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunPreScorePlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunScorePlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) (framework.PluginToNodeScores, *framework.Status) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunFilterPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) framework.PluginToStatus {
	statuses := make(framework.PluginToStatus)
	for _, pl := range f.filterPlugins {
		pluginStatus := f.runFilterPlugin(ctx, pl, state, pod, nodeInfo)
		if !pluginStatus.IsSuccess() {
			if !pluginStatus.IsUnschedulable() {
				// Filter plugins are not supposed to return any status other than
				// Success or Unschedulable.
				errStatus := framework.AsStatus(fmt.Errorf("running %q filter plugin: %w", pl.Name(), pluginStatus.AsError())).WithFailedPlugin(pl.Name())
				return map[string]*framework.Status{pl.Name(): errStatus}
			}
			pluginStatus.SetFailedPlugin(pl.Name())
			statuses[pl.Name()] = pluginStatus
			if !f.runAllFilters {
				// Exit early if we don't need to run all filters.
				return statuses
			}
		}
	}

	return statuses
}

func (f *TroubleShootPodScheduleFilterFramework) runFilterPlugin(ctx context.Context, pl framework.FilterPlugin, state *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	return pl.Filter(ctx, state, pod, nodeInfo)
}

func (f *TroubleShootPodScheduleFilterFramework) RunPreFilterExtensionAddPod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToAdd *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RunPreFilterExtensionRemovePod(ctx context.Context, state *framework.CycleState, podToSchedule *v1.Pod, podInfoToRemove *framework.PodInfo, nodeInfo *framework.NodeInfo) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) SnapshotSharedLister() framework.SharedLister {
	return f.snapshotSharedLister
}

func (f *TroubleShootPodScheduleFilterFramework) IterateOverWaitingPods(callback func(framework.WaitingPod)) {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) GetWaitingPod(uid types.UID) framework.WaitingPod {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) RejectWaitingPod(uid types.UID) bool {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) ClientSet() kubernetes.Interface {
	return f.clientSet
}

func (f *TroubleShootPodScheduleFilterFramework) KubeConfig() *rest.Config {
	return f.kubeConfig
}

func (*TroubleShootPodScheduleFilterFramework) EventRecorder() events.EventRecorder {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) SharedInformerFactory() informers.SharedInformerFactory {
	return f.sharedInformerFactory
}

func (f *TroubleShootPodScheduleFilterFramework) RunFilterPluginsWithNominatedPods(ctx context.Context, state *framework.CycleState, pod *v1.Pod, info *framework.NodeInfo) *framework.Status {
	//TODO implement me
	panic("implement me")
}

func (f *TroubleShootPodScheduleFilterFramework) Extenders() []framework.Extender {
	return []framework.Extender{}
}

func (f *TroubleShootPodScheduleFilterFramework) Parallelizer() parallelize.Parallelizer {
	return f.parallelizer
}

func (f *TroubleShootPodScheduleFilterFramework) RunPreFilterPlugins(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (status *framework.Status) {
	for _, pl := range f.preFilterPlugins {
		status = f.runPreFilterPlugin(ctx, pl, state, pod)
		if !status.IsSuccess() {
			status.SetFailedPlugin(pl.Name())
			if status.IsUnschedulable() {
				return status
			}
			return framework.AsStatus(fmt.Errorf("running PreFilter plugin %q: %w", pl.Name(), status.AsError())).WithFailedPlugin(pl.Name())
		}
	}

	return nil
}

func (f *TroubleShootPodScheduleFilterFramework) runPreFilterPlugin(ctx context.Context, pl framework.PreFilterPlugin, state *framework.CycleState, pod *v1.Pod) *framework.Status {
	return pl.PreFilter(ctx, state, pod)
}
