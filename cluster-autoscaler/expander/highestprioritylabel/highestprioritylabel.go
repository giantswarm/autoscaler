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
	"math"
	"strconv"

	"k8s.io/autoscaler/cluster-autoscaler/expander"
	"k8s.io/autoscaler/cluster-autoscaler/expander/random"
	"k8s.io/klog"
	schedulernodeinfo "k8s.io/kubernetes/pkg/scheduler/nodeinfo"
)

const (
	highestPriorityLabel = "giantswarm.io/autoscaling-priority"
)

type priority struct {
	fallback expander.Strategy
}

// NewStrategy returns a scale up strategy (expander) that picks the node group with the highest priority.
func NewStrategy() expander.Strategy {
	return &priority{
		fallback: random.NewStrategy(),
	}
}

// BestOption selects the expansion option based on the highest priority label
func (ps *priority) BestOption(expansionOptions []expander.Option, nodeInfo map[string]*schedulernodeinfo.NodeInfo) *expander.Option {
	priorityOptions := []expander.Option{}
	highestPriority := math.MinInt64

	for _, option := range expansionOptions {
		info, found := nodeInfo[option.NodeGroup.Id()]
		if !found {
			klog.Warningf("No node info for %s", option.NodeGroup.Id())
			continue
		}

		// get priority from label and default to 0 if missing
		scalingPriority, err := strconv.Atoi(info.Node().Labels[highestPriorityLabel])
		if err != nil {
			scalingPriority = 0
			klog.Warningf("Priority not set, using 0 for %s", option.NodeGroup.Id())
		}

		klog.Infof("ASG %s found with priority %d", option.NodeGroup.Id(), scalingPriority)

		// disregard node group if it has a lower priority than the current highest priority.
		if scalingPriority < highestPriority {
			continue
		}

		// keep track of the highest priority and reset options if we find a node
		// group with higher priority
		if scalingPriority > highestPriority {
			highestPriority = scalingPriority
			priorityOptions = []expander.Option{}
		}

		// add node group to options
		priorityOptions = append(priorityOptions, option)
	}

	klog.Infof("Selecting random node from %v", priorityOptions)

	// Pick and forward node group with the highest priority to the fallback strategy
	if len(priorityOptions) > 0 {
		return ps.fallback.BestOption(priorityOptions, nodeInfo)
	}

	// default to fallback strategy
	return ps.fallback.BestOption(expansionOptions, nodeInfo)
}
