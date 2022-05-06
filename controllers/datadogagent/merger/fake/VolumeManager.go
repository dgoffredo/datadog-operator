package fake

import (
	"testing"

	commonv1 "github.com/DataDog/datadog-operator/apis/datadoghq/common/v1"
	merger "github.com/DataDog/datadog-operator/controllers/datadogagent/merger"

	v1 "k8s.io/api/core/v1"
)

// VolumeManager is an autogenerated mock type for the VolumeManager type
type VolumeManager struct {
	Volumes        []*v1.Volume
	VolumeMountByC map[commonv1.AgentContainerName][]*v1.VolumeMount

	t testing.TB
}

// AddVolume provides a mock function with given fields: volume, volumeMount
func (_m *VolumeManager) AddVolume(volume *v1.Volume, volumeMount *v1.VolumeMount) {
	_m.Volumes = append(_m.Volumes, volume)
	_m.VolumeMountByC[AllContainers] = append(_m.VolumeMountByC[AllContainers], volumeMount)
}

// AddVolumeToContainer provides a mock function with given fields: volume, volumeMount, containerName
func (_m *VolumeManager) AddVolumeToContainer(volume *v1.Volume, volumeMount *v1.VolumeMount, containerName commonv1.AgentContainerName) {
	_m.Volumes = append(_m.Volumes, volume)
	_m.VolumeMountByC[containerName] = append(_m.VolumeMountByC[containerName], volumeMount)
}

// AddVolumeToContainerWithMergeFunc provides a mock function with given fields: volume, volumeMount, containerName, volumeMergeFunc, volumeMountMergeFunc
func (_m *VolumeManager) AddVolumeToContainerWithMergeFunc(volume *v1.Volume, volumeMount *v1.VolumeMount, containerName commonv1.AgentContainerName, volumeMergeFunc merger.VolumeMergeFunction, volumeMountMergeFunc merger.VolumeMountMergeFunction) error {
	if err := _m.volumeMerge(volume, volumeMergeFunc); err != nil {
		return err
	}
	return _m.volumeMountMerge(containerName, volumeMount, volumeMountMergeFunc)
}

func (_m *VolumeManager) volumeMerge(volume *v1.Volume, volumeMergeFunc merger.VolumeMergeFunction) error {
	found := false
	idFound := 0
	for id, v := range _m.Volumes {
		if volume.Name == v.Name {
			found = true
			idFound = id
		}
	}

	if found {
		var err error
		volume, err = volumeMergeFunc(_m.Volumes[idFound], volume)
		_m.Volumes[idFound] = volume
		return err
	}

	_m.Volumes = append(_m.Volumes, volume)
	return nil
}

func (_m *VolumeManager) volumeMountMerge(containerName commonv1.AgentContainerName, volume *v1.VolumeMount, volumeMergeFunc merger.VolumeMountMergeFunction) error {
	found := false
	idFound := 0
	for id, v := range _m.VolumeMountByC[containerName] {
		if volume.Name == v.Name {
			found = true
			idFound = id
		}
	}

	if found {
		var err error
		volume, err = volumeMergeFunc(_m.VolumeMountByC[containerName][idFound], volume)
		_m.VolumeMountByC[containerName][idFound] = volume
		return err
	}

	_m.VolumeMountByC[containerName] = append(_m.VolumeMountByC[containerName], volume)
	return nil
}

// AddVolumeWithMergeFunc provides a mock function with given fields: volume, volumeMount, volumeMergeFunc, volumeMountMergeFunc
func (_m *VolumeManager) AddVolumeWithMergeFunc(volume *v1.Volume, volumeMount *v1.VolumeMount, volumeMergeFunc merger.VolumeMergeFunction, volumeMountMergeFunc merger.VolumeMountMergeFunction) error {
	return _m.AddVolumeToContainerWithMergeFunc(volume, volumeMount, AllContainers, volumeMergeFunc, volumeMountMergeFunc)
}

// NewFakeVolumeManager creates a new instance of VolumeManager. It also registers the testing.TB interface on the mock and a cleanup function to assert the mocks expectations.
func NewFakeVolumeManager(t testing.TB) *VolumeManager {
	return &VolumeManager{
		VolumeMountByC: make(map[commonv1.AgentContainerName][]*v1.VolumeMount),
		t:              t,
	}
}
