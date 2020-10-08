/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package highestprioritylabel

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	testprovider "k8s.io/autoscaler/cluster-autoscaler/cloudprovider/test"
	"k8s.io/autoscaler/cluster-autoscaler/expander"
	. "k8s.io/autoscaler/cluster-autoscaler/utils/test"
	schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)

func TestPriorityBased(t *testing.T) {
	provider := testprovider.NewTestCloudProvider(nil, nil)

	groupOptions := make(map[string]expander.Option)
	nodeInfos := make(map[string]*schedulernodeinfo.NodeInfo)

	for ngId, priority := range map[string]*int64{
		"highPriority":  intPtr(1000),
		"highPriority2": intPtr(1000),
		"lowPriority":   intPtr(100),
		"noPriority":    nil,
		"zeroPriority":  intPtr(0),
	} {
		provider.AddNodeGroup(ngId, 1, 10, 1)
		node := BuildTestNode(ngId, 1000, 1000)
		if priority != nil {
			node.Labels[highestPriorityLabel] = strconv.FormatInt(*priority, 10)
		}
		provider.AddNode(ngId, node)

		nodeGroup, _ := provider.NodeGroupForNode(node)
		nodeInfo := schedulernodeinfo.NewNodeInfo()
		_ = nodeInfo.SetNode(node)

		groupOptions[ngId] = expander.Option{
			NodeGroup: nodeGroup,
			NodeCount: 1,
			Debug:     ngId,
		}
		nodeInfos[ngId] = nodeInfo
	}

	var (
		zeroPriorityGroup  = groupOptions["zeroPriority"]
		noPriorityGroup    = groupOptions["noPriority"]
		lowPriorityGroup   = groupOptions["lowPriority"]
		highPriorityGroup  = groupOptions["highPriority"]
		highPriorityGroup2 = groupOptions["highPriority2"]
	)

	e := NewStrategy()

	// if there's no available options we return nil
	ret := e.BestOption([]expander.Option{}, nodeInfos)
	assert.Nil(t, ret)

	// if there's only one group we return that group
	ret = e.BestOption([]expander.Option{highPriorityGroup}, nodeInfos)
	require.NotNil(t, ret)
	assert.True(t, assert.ObjectsAreEqual(*ret, highPriorityGroup))

	// if there's two groups we return the one with higher priority
	ret = e.BestOption([]expander.Option{highPriorityGroup, lowPriorityGroup}, nodeInfos)
	require.NotNil(t, ret)
	assert.True(t, assert.ObjectsAreEqual(*ret, highPriorityGroup))

	// if there's the same two groups in different order the result is the same
	ret = e.BestOption([]expander.Option{lowPriorityGroup, highPriorityGroup}, nodeInfos)
	require.NotNil(t, ret)
	assert.True(t, assert.ObjectsAreEqual(*ret, highPriorityGroup))

	// if there's many different priorities we return the pool with the highest
	ret = e.BestOption([]expander.Option{highPriorityGroup, lowPriorityGroup}, nodeInfos)
	require.NotNil(t, ret)
	assert.True(t, assert.ObjectsAreEqual(*ret, highPriorityGroup2))

	// if there's multiple groups with same priority we return either one
	ret = e.BestOption([]expander.Option{highPriorityGroup, highPriorityGroup2, lowPriorityGroup}, nodeInfos)
	require.NotNil(t, ret)
	assert.True(t, assert.ObjectsAreEqual(*ret, highPriorityGroup) || assert.ObjectsAreEqual(*ret, highPriorityGroup2))

	// if there's a group with no priority it's assumed to be zero and therefore less than low priority.
	ret = e.BestOption([]expander.Option{lowPriorityGroup, noPriorityGroup}, nodeInfos)
	require.NotNil(t, ret)
	assert.True(t, assert.ObjectsAreEqual(*ret, lowPriorityGroup))

	// if there's a group with zero priority it's the same as no priority.
	ret = e.BestOption([]expander.Option{zeroPriorityGroup, noPriorityGroup}, nodeInfos)
	require.NotNil(t, ret)
	assert.True(t, assert.ObjectsAreEqual(*ret, zeroPriorityGroup) || assert.ObjectsAreEqual(*ret, noPriorityGroup))
}

func intPtr(v int64) *int64 {
	return &v
}
