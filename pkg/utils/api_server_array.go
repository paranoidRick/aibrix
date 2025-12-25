package utils

import (
	"sync"

	v1 "k8s.io/api/core/v1"
)

type ApiServer struct {
	*v1.Pod
	Port int
}

// ApiServerArray is a simple implementation of types.PodList indexed by deployment names
type ApiServerArray struct {
	ApiServers []*ApiServer

	deployments      []string
	podsByDeployment map[string][]*v1.Pod
	mu               sync.Mutex
}

func (arr *ApiServerArray) Len() int {
	if arr == nil {
		return 0
	}
	return len(arr.ApiServers)
}

func (arr *ApiServerArray) All() []*ApiServer {
	if arr == nil {
		return nil
	}
	return arr.ApiServers
}

// Indexes TODO: 更改
func (arr *ApiServerArray) Indexes() []string {
	if len(arr.ApiServers) == 0 {
		return nil
	}

	return arr.deployments
}

func (arr *ApiServerArray) ListByIndex(index string) []*ApiServer {
	panic("implement me")
}
