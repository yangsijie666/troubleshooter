package pkg

import (
	"context"
	"fmt"
	"github.com/briandowns/spinner"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"strings"
	"time"
)

type PodScheduleTroubleShooter struct {
	podName      *string
	podNamespace *string
	nodeName     *string

	kubeConfig *rest.Config
	client     *kubernetes.Clientset
}

func (s *PodScheduleTroubleShooter) Execute() {
	sp := spinner.New(spinner.CharSets[39], 10*time.Millisecond)
	sp.Start()

	conclusion, err := s.executeCore()
	sp.Stop()
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Printf("Conclusion is:\n%s\n", conclusion)
	}
}

func (s *PodScheduleTroubleShooter) executeCore() (string, error) {
	client, err := kubernetes.NewForConfig(s.kubeConfig)
	if err != nil {
		return "", err
	}
	s.client = client

	ctx := context.TODO()

	var pod *v1.Pod
	if *s.podNamespace == "" {
		pods, err := s.client.CoreV1().Pods(*s.podNamespace).List(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + *s.podName,
		})
		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Sprintf("Pod %s in all namespaces not found\n", *s.podName), nil
			} else {
				return "", err
			}
		}
		if len(pods.Items) == 0 {
			return fmt.Sprintf("Pod %s in all namespaces not found\n", *s.podName), nil
		}
		pod = pods.Items[0].DeepCopy()
	} else {
		pod, err = s.client.CoreV1().Pods(*s.podNamespace).Get(ctx, *s.podName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return fmt.Sprintf("Pod %s in namespace %s not found\n", *s.podName, *s.podNamespace), nil
			} else {
				return "", err
			}
		}
	}

	node, err := s.client.CoreV1().Nodes().Get(ctx, *s.nodeName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Sprintf("Node %s not found\n", *s.nodeName), nil
		} else {
			return "", err
		}
	}

	if pod.Spec.NodeName == node.Name {
		return fmt.Sprintf("Pod %s in namespace %s has already been scheduled to Node %s\n", pod.Name, pod.Namespace, node.Name), nil
	} else if pod.Spec.NodeName != "" {
		return fmt.Sprintf("Pod %s in namespace %s has been scheduled to other Node %s\n", pod.Name, pod.Namespace, pod.Spec.NodeName), nil
	}

	pods, err := s.client.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})
	if err != nil {
		return "", err
	}

	conclusion := ""
	singleConclusion, needContinue, err := s.troubleShootByResource(pod, node, pods)
	if err != nil {
		return "", err
	}
	conclusion += singleConclusion + "\n"
	if !needContinue {
		return conclusion, nil
	}

	//singleConclusion, err = s.troubleShootByPodAntiAffinity(pod, pods)
	//if err != nil {
	//	return "", err
	//}
	//conclusion += singleConclusion + "\n"

	singleConclusion, err = s.troubleShootByNodeTaint(pod, node)
	if err != nil {
		return "", err
	}
	conclusion += singleConclusion + "\n"

	singleConclusion, err = s.troubleShootByPodNodeSelector(pod, node)
	if err != nil {
		return "", err
	}
	conclusion += singleConclusion + "\n"

	singleConclusion, err = s.troubleShootByNodeUnschedule(pod, node)
	if err != nil {
		return "", err
	}
	conclusion += singleConclusion + "\n"

	return conclusion, nil
}

func (s *PodScheduleTroubleShooter) troubleShootByResource(pod *v1.Pod, node *v1.Node, pods *v1.PodList) (conclusion string, needContinue bool, err error) {
	podRequest := make(v1.ResourceList, 0)
	for _, podItem := range pods.Items {
		if podItem.Name == pod.Name && podItem.Namespace == pod.Namespace {
			return fmt.Sprintf("Pod %s in namespace %s has already been scheduled to Node %s.", pod.Name, pod.Namespace, node.Name), false, nil
		}

		for _, container := range podItem.Spec.Containers {
			for resName, quantity := range container.Resources.Requests {
				if q, ok := podRequest[resName]; ok {
					q.Add(quantity)
				} else {
					podRequest[resName] = quantity
				}
			}
		}
	}

	insufficientResourceList := make(v1.ResourceList)

	nodeAllocatable := node.Status.Allocatable
	for requestResource, requestQuantity := range podRequest {
		allocatableResource := nodeAllocatable.Name(requestResource, requestQuantity.Format)
		if allocatableResource.Cmp(requestQuantity) < 0 {
			insufficient := requestQuantity.DeepCopy()
			copyAlloc := allocatableResource.DeepCopy()
			insufficient.Sub(copyAlloc)
			insufficientResourceList[requestResource] = insufficient
		}
	}

	if len(insufficientResourceList) == 0 {
		return fmt.Sprintf("[Pass] Allocatable resource of node is sufficient."), true, nil
	} else {
		insufficientStrings := make([]string, len(insufficientResourceList))
		for resourceName, quantity := range insufficientResourceList {
			insufficientStrings = append(insufficientStrings, fmt.Sprintf("%s: %s", resourceName.String(), quantity.String()))
		}
		return fmt.Sprintf("[NoPass] Allocatable resource of node is insufficient: [%s]", strings.Join(insufficientStrings, ",")), true, nil
	}
}

func (s *PodScheduleTroubleShooter) troubleShootByNodeTaint(pod *v1.Pod, node *v1.Node) (string, error) {
	filterPredicate := func(t *v1.Taint) bool {
		return t.Effect == v1.TaintEffectNoSchedule || t.Effect == v1.TaintEffectNoExecute
	}

	untoleratedTaints := findMatchingUntoleratedTaints(node.Spec.Taints, pod.Spec.Tolerations, filterPredicate)
	if len(untoleratedTaints) == 0 {
		return fmt.Sprintf("[Pass] Found no taints that are not tolerated."), nil
	}
	untoleratedTaintStrList := make([]string, len(untoleratedTaints))
	for _, taint := range untoleratedTaints {
		untoleratedTaintStrList = append(untoleratedTaintStrList, taint.String())
	}
	return fmt.Sprintf("[NoPass] Found taints that untolerated by pod: %s", strings.Join(untoleratedTaintStrList, ",")), nil
}

func (s *PodScheduleTroubleShooter) troubleShootByNodeUnschedule(pod *v1.Pod, node *v1.Node) (string, error) {
	tolerations := pod.Spec.Tolerations
	if tolerations == nil {
		tolerations = make([]v1.Toleration, 0)
	}

	canTolerate := false
	for _, toleration := range tolerations {
		canTolerate = canTolerate || toleration.ToleratesTaint(&v1.Taint{
			Key:    v1.TaintNodeUnschedulable,
			Effect: v1.TaintEffectNoSchedule,
		})
		if canTolerate {
			break
		}
	}

	if !canTolerate && node.Spec.Unschedulable {
		return fmt.Sprintf("[NoPass] Node is unschedulable now."), nil
	}
	return fmt.Sprintf("[Pass] Node is schedulable."), nil
}

type taintsFilterFunc func(*v1.Taint) bool

// FindMatchingUntoleratedTaints checks if the given tolerations tolerates
// all the filtered taints, and returns the first taint without a toleration
func findMatchingUntoleratedTaints(taints []v1.Taint, tolerations []v1.Toleration, inclusionFilter taintsFilterFunc) []v1.Taint {
	untoleratedTaints := make([]v1.Taint, 0)

	filteredTaints := getFilteredTaints(taints, inclusionFilter)
	for _, taint := range filteredTaints {
		if !tolerationsTolerateTaint(tolerations, &taint) {
			untoleratedTaints = append(untoleratedTaints, taint)
		}
	}
	return untoleratedTaints
}

// tolerationsTolerateTaint checks if taint is tolerated by any of the tolerations.
func tolerationsTolerateTaint(tolerations []v1.Toleration, taint *v1.Taint) bool {
	for i := range tolerations {
		if tolerations[i].ToleratesTaint(taint) {
			return true
		}
	}
	return false
}

// getFilteredTaints returns a list of taints satisfying the filter predicate
func getFilteredTaints(taints []v1.Taint, inclusionFilter taintsFilterFunc) []v1.Taint {
	if inclusionFilter == nil {
		return taints
	}
	filteredTaints := []v1.Taint{}
	for _, taint := range taints {
		if !inclusionFilter(&taint) {
			continue
		}
		filteredTaints = append(filteredTaints, taint)
	}
	return filteredTaints
}

func (s *PodScheduleTroubleShooter) troubleShootByPodNodeSelector(pod *v1.Pod, node *v1.Node) (string, error) {
	if len(pod.Spec.NodeSelector) > 0 {
		mismatchNodeSelectors := troubleShootByNodeSelector(pod.Spec.NodeSelector, node.Labels)
		if mismatchNodeSelectors == nil || len(mismatchNodeSelectors) > 0 {
			return fmt.Sprintf("[NoPass] The nodeSelector of pod contains value(s) that node labels do not have: %s", mismatchNodeSelectors), nil
		}
	}

	affinity := pod.Spec.Affinity
	if affinity != nil && affinity.NodeAffinity != nil {
		if !troubleShootByNodeAffinityTerms(affinity.NodeAffinity, node.Labels, node.Name) {
			return fmt.Sprintf("[NoPass] Pod has node affinity that node labels do not have."), nil
		}
	}

	return fmt.Sprintf("[Pass] NodeSelectorAndAffinity of pod match labels of node."), nil
}

func troubleShootByNodeAffinityTerms(nodeAffinity *v1.NodeAffinity, nodeLabels map[string]string, nodeName string) bool {
	if nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return true
	}

	nodeSelectorTerms := nodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(nodeSelectorTerms) == 0 {
		return true
	}

	return matchNodeSelectorTerms(nodeSelectorTerms, nodeLabels, fields.Set{
		"metadata.name": nodeName,
	})
}

// matchNodeSelectorTerms checks whether the node labels and fields match node selector terms in ORed;
// nil or empty term matches no objects.
func matchNodeSelectorTerms(
	nodeSelectorTerms []v1.NodeSelectorTerm,
	nodeLabels labels.Set,
	nodeFields fields.Set,
) bool {
	for _, req := range nodeSelectorTerms {
		// nil or empty term selects no objects
		if len(req.MatchExpressions) == 0 && len(req.MatchFields) == 0 {
			continue
		}

		if len(req.MatchExpressions) != 0 {
			labelSelector, err := nodeSelectorRequirementsAsSelector(req.MatchExpressions)
			if err != nil || !labelSelector.Matches(nodeLabels) {
				continue
			}
		}

		if len(req.MatchFields) != 0 {
			fieldSelector, err := nodeSelectorRequirementsAsFieldSelector(req.MatchFields)
			if err != nil || !fieldSelector.Matches(nodeFields) {
				continue
			}
		}

		return true
	}

	return false
}

// nodeSelectorRequirementsAsSelector converts the []NodeSelectorRequirement api type into a struct that implements
// labels.Selector.
func nodeSelectorRequirementsAsSelector(nsm []v1.NodeSelectorRequirement) (labels.Selector, error) {
	if len(nsm) == 0 {
		return labels.Nothing(), nil
	}
	selector := labels.NewSelector()
	for _, expr := range nsm {
		var op selection.Operator
		switch expr.Operator {
		case v1.NodeSelectorOpIn:
			op = selection.In
		case v1.NodeSelectorOpNotIn:
			op = selection.NotIn
		case v1.NodeSelectorOpExists:
			op = selection.Exists
		case v1.NodeSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		case v1.NodeSelectorOpGt:
			op = selection.GreaterThan
		case v1.NodeSelectorOpLt:
			op = selection.LessThan
		default:
			return nil, fmt.Errorf("%q is not a valid node selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*r)
	}
	return selector, nil
}

// nodeSelectorRequirementsAsFieldSelector converts the []NodeSelectorRequirement core type into a struct that implements
// fields.Selector.
func nodeSelectorRequirementsAsFieldSelector(nsm []v1.NodeSelectorRequirement) (fields.Selector, error) {
	if len(nsm) == 0 {
		return fields.Nothing(), nil
	}

	selectors := []fields.Selector{}
	for _, expr := range nsm {
		switch expr.Operator {
		case v1.NodeSelectorOpIn:
			if len(expr.Values) != 1 {
				return nil, fmt.Errorf("unexpected number of value (%d) for node field selector operator %q",
					len(expr.Values), expr.Operator)
			}
			selectors = append(selectors, fields.OneTermEqualSelector(expr.Key, expr.Values[0]))

		case v1.NodeSelectorOpNotIn:
			if len(expr.Values) != 1 {
				return nil, fmt.Errorf("unexpected number of value (%d) for node field selector operator %q",
					len(expr.Values), expr.Operator)
			}
			selectors = append(selectors, fields.OneTermNotEqualSelector(expr.Key, expr.Values[0]))

		default:
			return nil, fmt.Errorf("%q is not a valid node field selector operator", expr.Operator)
		}
	}

	return fields.AndSelectors(selectors...), nil
}

func troubleShootByNodeSelector(nodeSelector, nodeLabels map[string]string) map[string]string {
	if nodeLabels == nil {
		nodeLabels = map[string]string{}
	}
	if nodeSelector == nil {
		nodeSelector = map[string]string{}
	}

	mismatchNodeSelectors := make(map[string]string)

	for k, v := range nodeSelector {
		if nv, ok := nodeLabels[k]; ok {
			if v != nv {
				mismatchNodeSelectors[k] = v
			}
		} else {
			mismatchNodeSelectors[k] = v
		}
	}
	return mismatchNodeSelectors
}

func NewPodScheduleTroubleShooter(kubeConfigPath, podName, podNamespace, nodeName string) *PodScheduleTroubleShooter {
	if kubeConfigPath == "" {
		panic("path to kubeConfig should not be null")
	}

	if podName == "" {
		panic("pod should not be null")
	}

	if nodeName == "" {
		panic("node should not be null")
	}

	kubeConfig, err := loadKubeConfigByPath(kubeConfigPath)
	if err != nil {
		panic(err)
	}

	return &PodScheduleTroubleShooter{
		podName:      &podName,
		podNamespace: &podNamespace,
		nodeName:     &nodeName,
		kubeConfig:   kubeConfig,
	}
}
