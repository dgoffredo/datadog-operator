// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package agentprofile

import (
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"

	"github.com/DataDog/datadog-operator/apis/datadoghq/common/v1"
	datadoghqv1alpha1 "github.com/DataDog/datadog-operator/apis/datadoghq/v1alpha1"
	"github.com/DataDog/datadog-operator/apis/datadoghq/v2alpha1"
)

const (
	DaemonSetLabelKey   = "agent.datadoghq.com/profile"
	defaultProfileName  = "default"
	daemonSetNamePrefix = "datadog-agent-with-profile-"
)

// ProfilesToApply given a list of profiles, returns the ones that should be
// applied in the cluster.
// - If there are no profiles, it returns the default profile.
// - If there are no conflicting profiles, it returns all the profiles plus the default one.
// - If there are conflicting profiles, it returns a subset that does not
// conflict plus the default one. When there are conflicting profiles, the
// oldest one is the one that takes precedence. When two profiles share an
// identical creation timestamp, the profile whose name is alphabetically first
// is considered to have priority.
func ProfilesToApply(profiles []datadoghqv1alpha1.DatadogAgentProfile, nodes []v1.Node) ([]datadoghqv1alpha1.DatadogAgentProfile, error) {
	var res []datadoghqv1alpha1.DatadogAgentProfile

	nodesWithProfilesApplied := make(map[string]bool, len(nodes))
	for _, node := range nodes {
		nodesWithProfilesApplied[node.Name] = false
	}

	sortedProfiles := sortProfiles(profiles)

	for _, profile := range sortedProfiles {
		conflicts := false
		nodesThatMatchProfile := map[string]bool{}

		for _, node := range nodes {
			matchesNode, err := profileMatchesNode(&profile, &node)
			if err != nil {
				return nil, err
			}

			if matchesNode {
				if nodesWithProfilesApplied[node.Name] {
					// Conflict. This profile should not be applied.
					conflicts = true
					break
				} else {
					nodesThatMatchProfile[node.Name] = true
				}
			}
		}

		if conflicts {
			continue
		}

		for node := range nodesThatMatchProfile {
			nodesWithProfilesApplied[node] = true
		}

		res = append(res, profile)
	}

	return append(res, defaultProfile(res)), nil
}

// ComponentOverrideFromProfile returns the component override that should be
// applied according to the given profile.
func ComponentOverrideFromProfile(profile *datadoghqv1alpha1.DatadogAgentProfile) v2alpha1.DatadogAgentComponentOverride {
	overrideDSName := DaemonSetName(types.NamespacedName{
		Namespace: profile.Namespace,
		Name:      profile.Name,
	})

	return v2alpha1.DatadogAgentComponentOverride{
		Name:       &overrideDSName,
		Affinity:   affinityOverride(profile),
		Containers: containersOverride(profile),
		Labels:     labelsOverride(profile),
	}
}

// DaemonSetName returns the name that the DaemonSet should have according to
// the name of the profile associated with it.
func DaemonSetName(profileNamespacedName types.NamespacedName) string {
	if profileNamespacedName.Name == defaultProfileName {
		return "" // Return empty so it does not override the default DaemonSet name
	}

	return daemonSetNamePrefix + profileNamespacedName.Namespace + "-" + profileNamespacedName.Name
}

// defaultProfile returns the default profile, which is the one to be applied in
// the nodes where none of the profiles received apply.
// Note: this function assumes that the profiles received do not conflict.
func defaultProfile(profiles []datadoghqv1alpha1.DatadogAgentProfile) datadoghqv1alpha1.DatadogAgentProfile {
	var nodeSelectorRequirements []v1.NodeSelectorRequirement

	// TODO: I think this strategy only works if there's only one node selector per profile.
	for _, profile := range profiles {
		if profile.Spec.ProfileAffinity != nil {
			for _, nodeSelectorRequirement := range profile.Spec.ProfileAffinity.ProfileNodeAffinity {
				nodeSelectorRequirements = append(
					nodeSelectorRequirements,
					v1.NodeSelectorRequirement{
						Key:      nodeSelectorRequirement.Key,
						Operator: oppositeOperator(nodeSelectorRequirement.Operator),
						Values:   nodeSelectorRequirement.Values,
					},
				)
			}
		}
	}

	profile := datadoghqv1alpha1.DatadogAgentProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultProfileName,
		},
	}

	if len(nodeSelectorRequirements) > 0 {
		profile.Spec.ProfileAffinity = &datadoghqv1alpha1.ProfileAffinity{
			ProfileNodeAffinity: nodeSelectorRequirements,
		}
	}

	return profile
}

func oppositeOperator(op v1.NodeSelectorOperator) v1.NodeSelectorOperator {
	switch op {
	case v1.NodeSelectorOpIn:
		return v1.NodeSelectorOpNotIn
	case v1.NodeSelectorOpNotIn:
		return v1.NodeSelectorOpIn
	case v1.NodeSelectorOpExists:
		return v1.NodeSelectorOpDoesNotExist
	case v1.NodeSelectorOpDoesNotExist:
		return v1.NodeSelectorOpExists
	case v1.NodeSelectorOpGt:
		return v1.NodeSelectorOpLt
	case v1.NodeSelectorOpLt:
		return v1.NodeSelectorOpGt
	default:
		return ""
	}
}

func affinityOverride(profile *datadoghqv1alpha1.DatadogAgentProfile) *v1.Affinity {
	if profile.Spec.ProfileAffinity == nil || len(profile.Spec.ProfileAffinity.ProfileNodeAffinity) == 0 {
		return nil
	}

	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{
					{
						MatchExpressions: profile.Spec.ProfileAffinity.ProfileNodeAffinity,
					},
				},
			},
		},
	}
}

func containersOverride(profile *datadoghqv1alpha1.DatadogAgentProfile) map[common.AgentContainerName]*v2alpha1.DatadogAgentGenericContainer {
	if profile.Spec.Config == nil {
		return nil
	}

	nodeAgentOverride, ok := profile.Spec.Config.Override[datadoghqv1alpha1.NodeAgentComponentName]
	if !ok { // We only support overrides for the node agent, if there is no override for it, there's nothing to do
		return nil
	}

	if len(nodeAgentOverride.Containers) == 0 {
		return nil
	}

	containersInNodeAgent := []common.AgentContainerName{
		common.CoreAgentContainerName,
		common.TraceAgentContainerName,
		common.ProcessAgentContainerName,
		common.SecurityAgentContainerName,
		common.SystemProbeContainerName,
	}

	res := map[common.AgentContainerName]*v2alpha1.DatadogAgentGenericContainer{}

	for _, containerName := range containersInNodeAgent {
		if overrideForContainer, overrideIsDefined := nodeAgentOverride.Containers[containerName]; overrideIsDefined {
			res[containerName] = &v2alpha1.DatadogAgentGenericContainer{
				Resources: overrideForContainer.Resources,
			}
		}
	}

	return res
}

func labelsOverride(profile *datadoghqv1alpha1.DatadogAgentProfile) map[string]string {
	if profile.Name == defaultProfileName {
		return nil
	}

	return map[string]string{
		// Can't use the namespaced name because it includes "/" which is not
		// accepted in labels.
		DaemonSetLabelKey: fmt.Sprintf("%s-%s", profile.Namespace, profile.Name),
	}
}

// sortProfiles sorts the profiles by creation timestamp. If two profiles have
// the same creation timestamp, it sorts them by name.
func sortProfiles(profiles []datadoghqv1alpha1.DatadogAgentProfile) []datadoghqv1alpha1.DatadogAgentProfile {
	sortedProfiles := make([]datadoghqv1alpha1.DatadogAgentProfile, len(profiles))
	copy(sortedProfiles, profiles)

	sort.Slice(sortedProfiles, func(i, j int) bool {
		if !sortedProfiles[i].CreationTimestamp.Equal(&sortedProfiles[j].CreationTimestamp) {
			return sortedProfiles[i].CreationTimestamp.Before(&sortedProfiles[j].CreationTimestamp)
		}

		return sortedProfiles[i].Name < sortedProfiles[j].Name
	})

	return sortedProfiles
}

func profileMatchesNode(profile *datadoghqv1alpha1.DatadogAgentProfile, node *v1.Node) (bool, error) {
	if profile.Spec.ProfileAffinity == nil {
		return true, nil
	}

	for _, requirement := range profile.Spec.ProfileAffinity.ProfileNodeAffinity {
		selector, err := labels.NewRequirement(
			requirement.Key,
			nodeSelectorOperatorToSelectionOperator(requirement.Operator),
			requirement.Values,
		)
		if err != nil {
			return false, err
		}

		if !selector.Matches(labels.Set(node.Labels)) {
			return false, nil
		}
	}

	return true, nil
}

func nodeSelectorOperatorToSelectionOperator(op v1.NodeSelectorOperator) selection.Operator {
	switch op {
	case v1.NodeSelectorOpIn:
		return selection.In
	case v1.NodeSelectorOpNotIn:
		return selection.NotIn
	case v1.NodeSelectorOpExists:
		return selection.Exists
	case v1.NodeSelectorOpDoesNotExist:
		return selection.DoesNotExist
	case v1.NodeSelectorOpGt:
		return selection.GreaterThan
	case v1.NodeSelectorOpLt:
		return selection.LessThan
	default:
		return ""
	}
}
