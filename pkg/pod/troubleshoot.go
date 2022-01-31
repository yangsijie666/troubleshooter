package pod

import (
	"context"
	"fmt"
	"github.com/briandowns/spinner"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kube-scheduler/config/v1beta3"
	"k8s.io/kubernetes/pkg/scheduler/apis/config"
	"k8s.io/kubernetes/pkg/scheduler/apis/config/scheme"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	frameworkplugins "k8s.io/kubernetes/pkg/scheduler/framework/plugins"
	"strings"
	"time"
	"troubleshooter/pkg"
)

type ScheduleTroubleShooter struct {
	pod      *v1.Pod
	nodeInfo *framework.NodeInfo

	kubeConfig *rest.Config
	client     *kubernetes.Clientset
}

func NewScheduleTroubleShooter(
	kubeConfigPath,
	podName,
	podNamespace,
	nodeName string,
) *ScheduleTroubleShooter {
	ctx := context.Background()

	kubeConfig, err := pkg.LoadKubeConfigByPath(kubeConfigPath)
	if err != nil {
		panic(err)
	}

	if len(podName) == 0 {
		panic(fmt.Errorf("podName should not be empty"))
	}

	if len(podNamespace) == 0 {
		podNamespace = ""
	}

	if len(nodeName) == 0 {
		panic(fmt.Errorf("nodeName should not be empty"))
	}

	clientSet, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		panic(err)
	}

	pod, err := findPod(ctx, clientSet, podName, podNamespace)
	if err != nil {
		panic(err)
	}

	node, err := findNode(ctx, clientSet, nodeName)
	if err != nil {
		panic(err)
	}

	nodeInfo, err := buildNodeInfo(ctx, clientSet, node)
	if err != nil {
		panic(err)
	}

	return &ScheduleTroubleShooter{
		pod:        pod,
		nodeInfo:   nodeInfo,
		kubeConfig: kubeConfig,
		client:     clientSet,
	}
}

func buildNodeInfo(ctx context.Context, cs *clientset.Clientset, node *v1.Node) (*framework.NodeInfo, error) {
	pods := make([]*v1.Pod, 0)
	podList, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})
	if err != nil {
		if !errors.IsNotFound(err) {
			return nil, err
		}
	} else {
		for _, p := range podList.Items {
			pods = append(pods, p.DeepCopy())
		}
	}

	ni := framework.NewNodeInfo(pods...)
	ni.SetNode(node)
	return ni, nil
}

func findPod(ctx context.Context, cs *kubernetes.Clientset, name, namespace string) (*v1.Pod, error) {
	var pod *v1.Pod
	if len(namespace) == 0 {
		pods, err := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + name,
		})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, fmt.Errorf("Pod %s in all namespaces not found\n", name)
			} else {
				return nil, err
			}
		}
		if len(pods.Items) == 0 {
			return nil, fmt.Errorf("Pod %s in all namespaces not found\n", name)
		}
		pod = pods.Items[0].DeepCopy()
	} else {
		var err error
		pod, err = cs.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, fmt.Errorf("Pod %s in namespace %s not found\n", name, namespace)
			} else {
				return nil, err
			}
		}
	}
	return pod, nil
}

func findNode(ctx context.Context, cs *kubernetes.Clientset, name string) (*v1.Node, error) {
	node, err := cs.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, fmt.Errorf("Node %s not found\n", name)
		} else {
			return nil, err
		}
	}
	return node, nil
}

func (s *ScheduleTroubleShooter) Execute() {
	sp := spinner.New(spinner.CharSets[21], 100*time.Millisecond)
	sp.Start()

	ctx := context.Background()
	conclusion, err := s.executeCore(ctx)
	sp.Stop()
	if err != nil {
		panic(err)
	}
	fmt.Println(conclusion)
}

func (s *ScheduleTroubleShooter) executeCore(ctx context.Context) (string, error) {
	for _, pi := range s.nodeInfo.Pods {
		if ((s.pod.Namespace != "" && s.pod.Namespace == pi.Pod.Namespace) || (s.pod.Namespace == "")) && s.pod.Name == pi.Pod.Name {
			return fmt.Sprintf("Pod %s already on node %s", s.pod.Name, s.nodeInfo.Node().Name), nil
		}
	}

	status := framework.NewCycleState()
	fw, err := s.buildScheduleFramework()
	if err != nil {
		return "", err
	}

	preFilterPluginStatuses := fw.RunPreFilterPlugins(ctx, status, s.pod)
	if !preFilterPluginStatuses.IsSuccess() {
		return "", preFilterPluginStatuses.AsError()
	}

	filterPluginStatuses := fw.RunFilterPlugins(ctx, status, s.pod, s.nodeInfo)
	if len(filterPluginStatuses) == 0 {
		return fmt.Sprintf("[Success] Pod can be scheduled to nodes, please wait..."), nil
	} else {
		plgStatusList := make([]string, 0, len(filterPluginStatuses))
		for plg, status := range filterPluginStatuses {
			plgStatusList = append(plgStatusList, fmt.Sprintf("%s: %s", plg, strings.Join(status.Reasons(), ",")))
		}
		return fmt.Sprintf("[Fail] Reasons are:\n%s", strings.Join(plgStatusList, "\n")), nil
	}
}

func (s *ScheduleTroubleShooter) buildScheduleFramework() (framework.Framework, error) {
	var versionedCfg v1beta3.KubeSchedulerConfiguration
	scheme.Scheme.Default(&versionedCfg)
	cfg := config.KubeSchedulerConfiguration{}
	if err := scheme.Scheme.Convert(&versionedCfg, &cfg, nil); err != nil {
		return nil, err
	}

	profile := cfg.Profiles[0]
	registry := frameworkplugins.NewInTreeRegistry()
	informFactory := NewInformerFactory(s.client, 0)

	return NewFramework(
		registry,
		&profile,
		WithClientSet(s.client),
		WithKubeConfig(s.kubeConfig),
		WithInformerFactory(informFactory),
		WithSnapshotSharedLister(NewTroubleShootPodScheduleSnapshotSharedLister(s.nodeInfo)),
	)
}

func NewInformerFactory(cs clientset.Interface, resyncPeriod time.Duration) informers.SharedInformerFactory {
	informerFactory := informers.NewSharedInformerFactory(cs, resyncPeriod)
	informerFactory.InformerFor(&v1.Pod{}, newPodInformer)
	return informerFactory
}

func newPodInformer(cs clientset.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	selector := fmt.Sprintf("status.phase!=%v,status.phase!=%v", v1.PodSucceeded, v1.PodFailed)
	tweakListOptions := func(options *metav1.ListOptions) {
		options.FieldSelector = selector
	}
	return coreinformers.NewFilteredPodInformer(cs, metav1.NamespaceAll, resyncPeriod, nil, tweakListOptions)
}
