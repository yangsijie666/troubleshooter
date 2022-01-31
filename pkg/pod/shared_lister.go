package pod

import (
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type TroubleShootPodScheduleSnapshotSharedLister struct {
	TroubleShootPodScheduleSingleNodeInfoLister
}

func NewTroubleShootPodScheduleSnapshotSharedLister(nodeInfo *framework.NodeInfo) framework.SharedLister {
	return &TroubleShootPodScheduleSnapshotSharedLister{
		TroubleShootPodScheduleSingleNodeInfoLister{
			nodeInfo: nodeInfo,
		},
	}
}

func (l *TroubleShootPodScheduleSnapshotSharedLister) NodeInfos() framework.NodeInfoLister {
	return &l.TroubleShootPodScheduleSingleNodeInfoLister
}

type TroubleShootPodScheduleSingleNodeInfoLister struct {
	nodeInfo *framework.NodeInfo
}

func (l *TroubleShootPodScheduleSingleNodeInfoLister) List() ([]*framework.NodeInfo, error) {
	return []*framework.NodeInfo{l.nodeInfo}, nil
}

func (l *TroubleShootPodScheduleSingleNodeInfoLister) HavePodsWithAffinityList() ([]*framework.NodeInfo, error) {
	if len(l.nodeInfo.Pods) == 0 {
		return nil, nil
	}

	for _, pod := range l.nodeInfo.Pods {
		if len(pod.PreferredAffinityTerms) > 0 || len(pod.RequiredAffinityTerms) > 0 ||
			len(pod.PreferredAntiAffinityTerms) > 0 || len(pod.RequiredAntiAffinityTerms) > 0 {
			return []*framework.NodeInfo{l.nodeInfo}, nil
		}
	}

	return nil, nil
}

func (l *TroubleShootPodScheduleSingleNodeInfoLister) HavePodsWithRequiredAntiAffinityList() ([]*framework.NodeInfo, error) {
	if len(l.nodeInfo.Pods) == 0 {
		return nil, nil
	}

	for _, pod := range l.nodeInfo.Pods {
		if len(pod.RequiredAntiAffinityTerms) > 0 {
			return []*framework.NodeInfo{l.nodeInfo}, nil
		}
	}

	return nil, nil
}

func (l *TroubleShootPodScheduleSingleNodeInfoLister) Get(nodeName string) (*framework.NodeInfo, error) {
	if l.nodeInfo.Node() == nil {
		return nil, nil
	}

	if l.nodeInfo.Node().Name == nodeName {
		return l.nodeInfo, nil
	} else {
		return nil, nil
	}
}
