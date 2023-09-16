package bpf_map

import (
	"testcni/utils"
	"unsafe"

	"github.com/cilium/ebpf"
)

type MapsManager struct{}

func (mm *MapsManager) DeleteAllPodMap() (int, error) {
	m := mm.GetPodMap()
	itor := m.Iterate()
	keys := []PodNodeMapKey{}

	var key PodNodeMapKey
	var value PodNodeMapValue
	for itor.Next(&key, &value) {
		keys = append(keys, key)
	}
	return BatchDelKey(m, keys)
}

func (mm *MapsManager) BatchDelLxcMap(keys []EndpointMapKey) (int, error) {
	m := mm.GetLxcMap()
	return BatchDelKey(m, keys)
}

func (mm *MapsManager) BatchDelPodMap(keys []PodNodeMapKey) (int, error) {
	m := mm.GetPodMap()
	return BatchDelKey(m, keys)
}

func (mm *MapsManager) BatchDelNodeLocalMap(keys []LocalNodeMapKey) (int, error) {
	m := mm.GetNodeLocalMap()
	return BatchDelKey(m, keys)
}

func (mm *MapsManager) BatchSetLxcMap(key []EndpointMapKey, value []EndpointMapInfo) (int, error) {
	m := mm.GetLxcMap()
	return BatchSetMap(m, key, value)
}

func (mm *MapsManager) BatchSetPodMap(key []PodNodeMapKey, value []PodNodeMapValue) (int, error) {
	m := mm.GetPodMap()
	return BatchSetMap(m, key, value)
}

func (mm *MapsManager) BatchSetNodeLocalMap(key []LocalNodeMapKey, value []LocalNodeMapValue) (int, error) {
	m := mm.GetNodeLocalMap()
	return BatchSetMap(m, key, value)
}

func (mm *MapsManager) SetLxcMap(key EndpointMapKey, value EndpointMapInfo) error {
	m := mm.GetLxcMap()
	return SetMap(m, key, value)
}

func (mm *MapsManager) SetPodMap(key PodNodeMapKey, value PodNodeMapValue) error {
	m := mm.GetPodMap()
	return SetMap(m, key, value)
}

func (mm *MapsManager) SetNodeLocalMap(key LocalNodeMapKey, value LocalNodeMapValue) error {
	m := mm.GetNodeLocalMap()
	return SetMap(m, key, value)
}

func (mm *MapsManager) DelLxcMap(key EndpointMapKey) error {
	m := mm.GetLxcMap()
	return DelKey(m, key)
}

func (mm *MapsManager) DelPodMap(key PodNodeMapKey) error {
	m := mm.GetPodMap()
	return DelKey(m, key)
}

func (mm *MapsManager) DelNodeLocalMap(key LocalNodeMapKey) error {
	m := mm.GetNodeLocalMap()
	return DelKey(m, key)
}

func (mm *MapsManager) GetLxcMap() *ebpf.Map {
	return GetMapByPinned(LXC_MAP_DEFAULT_PATH)
}

func (mm *MapsManager) GetPodMap() *ebpf.Map {
	return GetMapByPinned(POD_MAP_DEFAULT_PATH)
}

func (mm *MapsManager) GetNodeLocalMap() *ebpf.Map {
	return GetMapByPinned(NODE_LOCAL_MAP_DEFAULT_PATH)
}

func (mm *MapsManager) GetLxcMapValue(key EndpointMapKey) (*EndpointMapInfo, error) {
	m := mm.GetLxcMap()
	value := &EndpointMapInfo{}
	err := GetMapValue(m, key, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (mm *MapsManager) GetPodMapValue(key PodNodeMapKey) (*PodNodeMapValue, error) {
	m := mm.GetPodMap()
	value := &PodNodeMapValue{}
	err := GetMapValue(m, key, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

func (mm *MapsManager) GetNodeLocalMapValue(key LocalNodeMapKey) (*LocalNodeMapValue, error) {
	m := mm.GetNodeLocalMap()
	value := &LocalNodeMapValue{}
	err := GetMapValue(m, key, value)
	if err != nil {
		return nil, err
	}
	return value, nil
}

// 创建一个用来存储本地 veth pair 网卡的 map
func (mm *MapsManager) CreateLxcMap() (*ebpf.Map, error) {
	const (
		pinPath    = LXC_MAP_DEFAULT_PATH
		name       = "lxc_map"
		_type      = ebpf.Hash
		keySize    = uint32(unsafe.Sizeof(EndpointMapKey{}))
		valueSize  = uint32(unsafe.Sizeof(EndpointMapInfo{}))
		maxEntries = MAX_ENTRIES
		flags      = 0
	)

	m, err := CreateOnceMapWithPin(
		pinPath,
		name,
		_type,
		keySize,
		valueSize,
		maxEntries,
		flags,
	)

	if err != nil {
		return nil, err
	}
	return m, nil
}

// 创建一个用来存储集群中其他节点上的 pod ip 的 map
func (mm *MapsManager) CreatePodMap() (*ebpf.Map, error) {
	const (
		pinPath    = POD_MAP_DEFAULT_PATH
		name       = "pod_map"
		_type      = ebpf.Hash
		keySize    = uint32(unsafe.Sizeof(PodNodeMapKey{}))
		valueSize  = uint32(unsafe.Sizeof(PodNodeMapValue{}))
		maxEntries = MAX_ENTRIES
		flags      = 0
	)

	m, err := CreateOnceMapWithPin(
		pinPath,
		name,
		_type,
		keySize,
		valueSize,
		maxEntries,
		flags,
	)

	if err != nil {
		return nil, err
	}
	return m, nil
}

// 创建一个用来存储本机网卡设备的 map
func (mm *MapsManager) CreateNodeLocalMap() (*ebpf.Map, error) {
	const (
		pinPath    = NODE_LOCAL_MAP_DEFAULT_PATH
		name       = "local_map"
		_type      = ebpf.Hash
		keySize    = uint32(unsafe.Sizeof(LocalNodeMapKey{}))
		valueSize  = uint32(unsafe.Sizeof(LocalNodeMapValue{}))
		maxEntries = MAX_ENTRIES
		flags      = 0
	)

	m, err := CreateOnceMapWithPin(
		pinPath,
		name,
		_type,
		keySize,
		valueSize,
		maxEntries,
		flags,
	)

	if err != nil {
		return nil, err
	}
	return m, nil
}

var GetMapsManager = func() func() (*MapsManager, error) {
	var mm *MapsManager
	return func() (*MapsManager, error) {
		if mm != nil {
			return mm, nil
		} else {
			var err error
			mm = &MapsManager{}
			lxcPath := utils.GetParentDirectory(LXC_MAP_DEFAULT_PATH)
			if !utils.PathExists(lxcPath) {
				err = utils.CreateDir(lxcPath)
			}
			podPath := utils.GetParentDirectory(POD_MAP_DEFAULT_PATH)
			if !utils.PathExists(podPath) {
				err = utils.CreateDir(podPath)
			}
			localPath := utils.GetParentDirectory(NODE_LOCAL_MAP_DEFAULT_PATH)
			if !utils.PathExists(localPath) {
				err = utils.CreateDir(localPath)
			}
			if err != nil {
				return nil, err
			}
			return mm, nil
		}
	}
}()
