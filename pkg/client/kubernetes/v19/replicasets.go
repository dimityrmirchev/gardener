// Copyright 2018 The Gardener Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kubernetesv19

import (
	"sort"

	"github.com/gardener/gardener/pkg/client/kubernetes/mapping"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ListReplicaSets returns the list of ReplicaSets in the given <namespace>.
func (c *Client) ListReplicaSets(namespace string, listOptions metav1.ListOptions) ([]*mapping.ReplicaSet, error) {
	var replicasetList []*mapping.ReplicaSet
	replicasets, err := c.Clientset().AppsV1().ReplicaSets(namespace).List(listOptions)
	if err != nil {
		return nil, err
	}
	sort.Slice(replicasets.Items, func(i, j int) bool {
		return replicasets.Items[i].ObjectMeta.CreationTimestamp.Before(&replicasets.Items[j].ObjectMeta.CreationTimestamp)
	})
	for _, replicaset := range replicasets.Items {
		replicasetList = append(replicasetList, mapping.AppsV1ReplicaSet(replicaset))
	}
	return replicasetList, nil
}

// DeleteReplicaSet deletes a ReplicaSet object.
func (c *Client) DeleteReplicaSet(namespace, name string) error {
	return c.Clientset().AppsV1().ReplicaSets(namespace).Delete(name, &defaultDeleteOptions)
}
