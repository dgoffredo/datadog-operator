// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package kubernetesstatecore

import (
	"fmt"
	"testing"

	apicommon "github.com/DataDog/datadog-operator/apis/datadoghq/common"
	apicommonv1 "github.com/DataDog/datadog-operator/apis/datadoghq/common/v1"
	"github.com/DataDog/datadog-operator/apis/datadoghq/v1alpha1"
	"github.com/DataDog/datadog-operator/apis/datadoghq/v2alpha1"
	apiutils "github.com/DataDog/datadog-operator/apis/utils"
	"github.com/DataDog/datadog-operator/controllers/datadogagent/feature"
	"github.com/DataDog/datadog-operator/controllers/datadogagent/feature/fake"
	"github.com/DataDog/datadog-operator/controllers/datadogagent/feature/test"
	mergerfake "github.com/DataDog/datadog-operator/controllers/datadogagent/merger/fake"
	"github.com/DataDog/datadog-operator/pkg/controller/utils/comparison"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

const (
	customData = `cluster_check: true
init_config:
instances:
    collectors:
    - pods`
)

func Test_ksmFeature_Configure(t *testing.T) {
	tests := test.FeatureTestSuite{
		//////////////////////////
		// v1Alpha1.DatadogAgent
		//////////////////////////
		{
			Name:          "v1alpha1 ksm-core not enabled",
			DDAv1:         newV1Agent(false, false),
			WantConfigure: false,
		},
		{
			Name:          "v1alpha1 ksm-core enabled",
			DDAv1:         newV1Agent(true, false),
			WantConfigure: true,
			ClusterAgent:  ksmClusterAgentWantFunc(false),
			Agent:         test.NewDefaultComponentTest().WithWantFunc(ksmAgentNodeWantFunc),
		},
		//////////////////////////
		// v2Alpha1.DatadogAgent
		//////////////////////////
		{
			Name:          "v2alpha1 ksm-core not enabled",
			DDAv2:         newV2Agent(false, false),
			WantConfigure: false,
		},
		{
			Name:          "v2alpha1 ksm-core not enabled",
			DDAv2:         newV2MonoAgent(false, false),
			WantConfigure: false,
		},
		{
			Name:          "v2alpha1 ksm-core enabled",
			DDAv2:         newV2Agent(true, false),
			WantConfigure: true,
			ClusterAgent:  ksmClusterAgentWantFunc(false),
			Agent:         test.NewDefaultComponentTest().WithWantFunc(ksmAgentNodeWantFunc),
		},
		{
			Name:          "v2alpha1 ksm-core enabled",
			DDAv2:         newV2MonoAgent(true, false),
			WantConfigure: true,
			ClusterAgent:  ksmClusterAgentWantFunc(false),
			Agent:         test.NewDefaultComponentTest().WithWantFunc(ksmMonoAgentWantFunc),
		},
		{
			Name:          "v2alpha1 ksm-core enabled, custom config",
			DDAv2:         newV2Agent(true, true),
			WantConfigure: true,
			ClusterAgent:  ksmClusterAgentWantFunc(true),
			Agent:         test.NewDefaultComponentTest().WithWantFunc(ksmAgentNodeWantFunc),
		},
		{
			Name:          "v2alpha1 ksm-core enabled, custom config",
			DDAv2:         newV2MonoAgent(true, true),
			WantConfigure: true,
			ClusterAgent:  ksmClusterAgentWantFunc(true),
			Agent:         test.NewDefaultComponentTest().WithWantFunc(ksmMonoAgentWantFunc),
		},
	}

	tests.Run(t, buildKSMFeature)
}

func newV1Agent(enableKSM bool, hasCustomConfig bool) *v1alpha1.DatadogAgent {
	ddaV1 := &v1alpha1.DatadogAgent{
		Spec: v1alpha1.DatadogAgentSpec{
			Features: v1alpha1.DatadogFeatures{
				KubeStateMetricsCore: &v1alpha1.KubeStateMetricsCore{
					Enabled: apiutils.NewBoolPointer(enableKSM),
				},
			},
		},
	}
	if hasCustomConfig {
		ddaV1.Spec.Features.KubeStateMetricsCore.Conf = &v1alpha1.CustomConfigSpec{
			ConfigData: apiutils.NewStringPointer(customData),
		}
	}
	return ddaV1
}

func newV2Agent(enableKSM bool, hasCustomConfig bool) *v2alpha1.DatadogAgent {
	ddaV2 := &v2alpha1.DatadogAgent{
		Spec: v2alpha1.DatadogAgentSpec{
			Features: &v2alpha1.DatadogFeatures{
				KubeStateMetricsCore: &v2alpha1.KubeStateMetricsCoreFeatureConfig{
					Enabled: apiutils.NewBoolPointer(enableKSM),
				},
			},
		},
	}
	if hasCustomConfig {
		ddaV2.Spec.Features.KubeStateMetricsCore.Conf = &v2alpha1.CustomConfig{
			ConfigData: apiutils.NewStringPointer(customData),
		}
	}
	return ddaV2
}

func newV2MonoAgent(enableKSM bool, hasCustomConfig bool) *v2alpha1.DatadogAgent {
	ddaV2 := newV2Agent(enableKSM, hasCustomConfig)
	ddaV2.Spec.Global = &v2alpha1.GlobalConfig{
		ContainerProcessModel: &v2alpha1.ContainerProcessModel{
			UseMultiProcessContainer: apiutils.NewBoolPointer(true),
		},
	}
	return ddaV2
}

func ksmClusterAgentWantFunc(hasCustomConfig bool) *test.ComponentTest {
	return test.NewDefaultComponentTest().WithWantFunc(
		func(t testing.TB, mgrInterface feature.PodTemplateManagers) {
			mgr := mgrInterface.(*fake.PodTemplateManagers)
			dcaEnvVars := mgr.EnvVarMgr.EnvVarsByC[mergerfake.AllContainers]

			want := []*corev1.EnvVar{
				{
					Name:  apicommon.DDKubeStateMetricsCoreEnabled,
					Value: "true",
				},
				{
					Name:  apicommon.DDKubeStateMetricsCoreConfigMap,
					Value: "-kube-state-metrics-core-config",
				},
			}
			assert.True(t, apiutils.IsEqualStruct(dcaEnvVars, want), "DCA envvars \ndiff = %s", cmp.Diff(dcaEnvVars, want))

			if hasCustomConfig {
				customConfig := apicommonv1.CustomConfig{
					ConfigData: apiutils.NewStringPointer(customData),
				}
				hash, err := comparison.GenerateMD5ForSpec(&customConfig)
				assert.NoError(t, err)
				wantAnnotations := map[string]string{
					fmt.Sprintf(apicommon.MD5ChecksumAnnotationKey, feature.KubernetesStateCoreIDType): hash,
				}
				annotations := mgr.AnnotationMgr.Annotations
				assert.True(t, apiutils.IsEqualStruct(annotations, wantAnnotations), "Annotations \ndiff = %s", cmp.Diff(annotations, wantAnnotations))
			}
		},
	)
}

func ksmAgentNodeWantFunc(t testing.TB, mgrInterface feature.PodTemplateManagers) {
	ksmAgentWantFunc(t, mgrInterface, apicommonv1.CoreAgentContainerName)
}

func ksmMonoAgentWantFunc(t testing.TB, mgrInterface feature.PodTemplateManagers) {
	ksmAgentWantFunc(t, mgrInterface, apicommonv1.NonPrivilegedMonoContainerName)
}

func ksmAgentWantFunc(t testing.TB, mgrInterface feature.PodTemplateManagers, agentContainerName apicommonv1.AgentContainerName) {
	mgr := mgrInterface.(*fake.PodTemplateManagers)
	agentEnvVars := mgr.EnvVarMgr.EnvVarsByC[agentContainerName]

	want := []*corev1.EnvVar{
		{
			Name:  apicommon.DDIgnoreAutoConf,
			Value: "kubernetes_state",
		},
	}
	assert.True(t, apiutils.IsEqualStruct(agentEnvVars, want), "Agent envvars \ndiff = %s", cmp.Diff(agentEnvVars, want))
}
