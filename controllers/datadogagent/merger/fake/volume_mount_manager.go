package fake

import (
	"testing"

	commonv1 "github.com/DataDog/datadog-operator/apis/datadoghq/common/v1"
	"github.com/DataDog/datadog-operator/controllers/datadogagent/merger"

	v1 "k8s.io/api/core/v1"
)

var initContainerNames = []commonv1.AgentContainerName{commonv1.InitConfigContainerName, commonv1.InitVolumeContainerName, commonv1.SeccompSetupContainerName}

// VolumeMountManager is an autogenerated mock type for the VolumeMountManager type
type VolumeMountManager struct {
	VolumeMountsByC map[commonv1.AgentContainerName][]*v1.VolumeMount

	t testing.TB
}

// AddVolumeMount provides a mock function with given field: volumeMount
func (_m *VolumeMountManager) AddVolumeMount(volumeMount *v1.VolumeMount) {
	_m.VolumeMountsByC[AllContainers] = append(_m.VolumeMountsByC[AllContainers], volumeMount)
}

// AddVolumeMountToContainer provides a mock function with given fields: volumeMount, containerName
func (_m *VolumeMountManager) AddVolumeMountToContainer(volumeMount *v1.VolumeMount, containerName commonv1.AgentContainerName) {
	isInitContainer := false
	for _, initContainerName := range initContainerNames {
		if containerName == initContainerName {
			isInitContainer = true
			break
		}
	}
	if !isInitContainer {
		_m.VolumeMountsByC[containerName] = append(_m.VolumeMountsByC[containerName], volumeMount)
	}
}

// AddVolumeMountToInitContainer provides a mock function with given fields: volumeMount, containerName
func (_m *VolumeMountManager) AddVolumeMountToInitContainer(volumeMount *v1.VolumeMount, containerName commonv1.AgentContainerName) {
	for _, initContainerName := range initContainerNames {
		if containerName == initContainerName {
			_m.VolumeMountsByC[containerName] = append(_m.VolumeMountsByC[containerName], volumeMount)
		}
	}
}

// AddVolumeMountToContainers provides a mock function with given fields: volume, volumeMount, containerNames
func (_m *VolumeMountManager) AddVolumeMountToContainers(volumeMount *v1.VolumeMount, containerNames []commonv1.AgentContainerName) {
	for _, c := range containerNames {
		_m.VolumeMountsByC[c] = append(_m.VolumeMountsByC[c], volumeMount)
	}
}

// AddVolumeMountToContainersWithMergeFunc provides a mock function with given fields: volume, volumeMount, containerNames, volumeMergeFunc, volumeMountMergeFunc
func (_m *VolumeMountManager) AddVolumeMountToContainersWithMergeFunc(volumeMount *v1.VolumeMount, containerNames []commonv1.AgentContainerName, volumeMountMergeFunc merger.VolumeMountMergeFunction) error {
	for _, cName := range containerNames {
		if err := _m.volumeMountMerge(cName, volumeMount, volumeMountMergeFunc); err != nil {
			return err
		}
	}
	return nil
}

// AddVolumeMountToContainerWithMergeFunc provides a mock function with given fields: volume, volumeMount, containerName, volumeMergeFunc, volumeMountMergeFunc
func (_m *VolumeMountManager) AddVolumeMountToContainerWithMergeFunc(volumeMount *v1.VolumeMount, containerName commonv1.AgentContainerName, volumeMountMergeFunc merger.VolumeMountMergeFunction) error {
	return _m.volumeMountMerge(containerName, volumeMount, volumeMountMergeFunc)
}

// RemoveVolumeMount provides a mock function with given fields: volumeMount
func (_m *VolumeMountManager) RemoveVolumeMount(volumeMount string) {
	for id, v := range _m.VolumeMountsByC[AllContainers] {
		if v.Name == volumeMount {
			updatedVolumeMounts := make([]*v1.VolumeMount, 0, len(_m.VolumeMountsByC[AllContainers])-1)
			updatedVolumeMounts = append(updatedVolumeMounts, _m.VolumeMountsByC[AllContainers][:id]...)
			_m.VolumeMountsByC[AllContainers] = append(updatedVolumeMounts, _m.VolumeMountsByC[AllContainers][id+1:]...)
		}
	}
}

func (_m *VolumeMountManager) volumeMountMerge(containerName commonv1.AgentContainerName, volume *v1.VolumeMount, volumeMergeFunc merger.VolumeMountMergeFunction) error {
	found := false
	idFound := 0
	for id, v := range _m.VolumeMountsByC[containerName] {
		if volume.Name == v.Name {
			found = true
			idFound = id
		}
	}

	if found {
		var err error
		volume, err = volumeMergeFunc(_m.VolumeMountsByC[containerName][idFound], volume)
		_m.VolumeMountsByC[containerName][idFound] = volume
		return err
	}

	_m.VolumeMountsByC[containerName] = append(_m.VolumeMountsByC[containerName], volume)
	return nil
}

// NewFakeVolumeMountManager creates a new instance of VolumeMountManager. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewFakeVolumeMountManager(t testing.TB) *VolumeMountManager {
	return &VolumeMountManager{
		VolumeMountsByC: make(map[commonv1.AgentContainerName][]*v1.VolumeMount),
		t:               t,
	}
}
