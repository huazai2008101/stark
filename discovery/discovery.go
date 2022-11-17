package discovery

import (
	"fmt"
	"sync"
)

type ServiceEventType int32

const (
	// 服务创建事件
	CreateServiceEvent ServiceEventType = 1
	// 服务销毁事件
	DestroyServiceEvent ServiceEventType = 2
)

type ServiceDiscovery interface {
	// 注册服务
	Register() error
	// 已注册服务实例
	ServiceInstances(serviceName string) []ServiceInfo
	SchemeName() string
	SchemeUrl() string
}

type ServiceInfo struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type ServiceSet struct {
	data map[string]ServiceInfo
	lock sync.RWMutex
}

func NewServiceSet() *ServiceSet {
	return &ServiceSet{
		data: make(map[string]ServiceInfo),
	}
}

func (s *ServiceSet) Put(instance ServiceInfo) bool {
	key := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
	s.lock.RLock()
	_, ok := s.data[key]
	s.lock.RUnlock()
	if ok {
		// 如果存在则不重复添加
		return false
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.data[key] = instance
	return true
}

func (s *ServiceSet) Remove(instance ServiceInfo) bool {
	key := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
	s.lock.RLock()
	_, ok := s.data[key]
	s.lock.RUnlock()
	if !ok {
		// 如果存在则不需要执行删除操作
		return false
	}

	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.data, key)
	return true
}

func (s *ServiceSet) List() []ServiceInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()
	list := make([]ServiceInfo, 0, len(s.data))
	for _, v := range s.data {
		list = append(list, v)
	}
	return list
}
